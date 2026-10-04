package feed

import (
	"context"
	"errors"
	"net/url"
	"path"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode/utf8"

	"github.com/zeebo/bencode"
)

type Sample struct {
	Title    string `json:"title"`
	Filename string `json:"filename"`
	Kind     string `json:"kind"`
}

type Samples struct {
	Items   []Sample `json:"items"`
	Notices []string `json:"notices"`
}

const sampleTorrentConcurrency = 5

// SamplesAll reads one feed snapshot and each distinct torrent once. Five
// workers bound simultaneous requests; request-wide budgets bound retained data.
func (c *Client) SamplesAll(ctx context.Context, feedURL string) (Samples, error) {
	out := Samples{Items: []Sample{}, Notices: []string{}}
	items, err := c.fetch(ctx, feedURL, true)
	if err != nil {
		return out, err
	}
	urls := []string{}
	indices := map[string]int{}
	for _, item := range items {
		if !strings.HasPrefix(item.URL, "magnet:") {
			if _, exists := indices[item.URL]; !exists {
				indices[item.URL] = len(urls)
				urls = append(urls, item.URL)
			}
		}
	}
	results := make([][]string, len(urls))
	readCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	var totalBytes atomic.Int64
	var overBudget atomic.Bool
	queue := make(chan int)
	var workers sync.WaitGroup
	for range min(sampleTorrentConcurrency, len(urls)) {
		workers.Go(func() {
			for index := range queue {
				if readCtx.Err() != nil {
					continue
				}
				torrentCtx, done := context.WithTimeout(readCtx, 15*time.Second)
				metadata, err := c.read(torrentCtx, urls[index])
				done()
				if err != nil {
					continue
				}
				if totalBytes.Add(int64(len(metadata))) > 32<<20 {
					overBudget.Store(true)
					cancel()
					continue
				}
				results[index], _ = torrentNames(metadata)
			}
		})
	}
enqueue:
	for index := range urls {
		select {
		case queue <- index:
		case <-readCtx.Done():
			break enqueue
		}
	}
	close(queue)
	workers.Wait()
	if overBudget.Load() {
		out.Notices = append(out.Notices, "種子中繼資料總量超過 32 MiB，已停止讀取；請使用較小的 RSS。")
	} else if ctx.Err() != nil {
		out.Notices = append(out.Notices, "種子檔名讀取逾時，已保留取得的檔名；可重新讀取。")
	}
	sampleBytes, failed := 0, false
	for _, item := range items {
		samples := []Sample{}
		if strings.HasPrefix(item.URL, "magnet:") {
			if _, err := NormalizeMagnet(item.URL); err == nil {
				u, _ := url.Parse(item.URL)
				if name := u.Query().Get("dn"); validSampleName(name) {
					samples = append(samples, Sample{item.Title, name, "magnet_name"})
				}
			}
		} else {
			for _, name := range results[indices[item.URL]] {
				samples = append(samples, Sample{item.Title, name, "torrent_file"})
			}
			if len(samples) == 0 {
				failed = true
			}
		}
		if len(samples) == 0 && item.Title != "" {
			samples = append(samples, Sample{item.Title, item.Title, "rss_title"})
		}
		for _, sample := range samples {
			sampleBytes += len(sample.Title) + len(sample.Filename)
			if len(out.Items) >= 10000 || sampleBytes > 8<<20 {
				out.Notices = append(out.Notices, "檔名清單超過 10,000 筆或 8 MiB，已保留部分檔名；請使用較小的 RSS。")
				return out, nil
			}
			out.Items = append(out.Items, sample)
		}
	}
	if failed {
		out.Notices = append(out.Notices, "部分種子無法取得或解析檔名，可重新讀取。")
	}
	return out, nil
}

func validSampleName(name string) bool {
	return name != "" && len(name) <= 8192 && utf8.ValidString(name) && !strings.ContainsRune(name, '\x00')
}

func torrentNames(metadata []byte) ([]string, error) {
	if _, err := Torrent(metadata); err != nil {
		return nil, err
	}
	var torrent struct {
		Info struct {
			Name     string `bencode:"name"`
			NameUTF8 string `bencode:"name.utf-8"`
			Files    []struct {
				Path     []string `bencode:"path"`
				PathUTF8 []string `bencode:"path.utf-8"`
			} `bencode:"files"`
			Tree map[string]any `bencode:"file tree"`
		} `bencode:"info"`
	}
	if err := bencode.DecodeBytes(metadata, &torrent); err != nil {
		return nil, errors.New("種子檔名無法解析")
	}
	names := []string{}
	add := func(name string) {
		if validSampleName(name) && name != "." && name != ".." && !strings.ContainsAny(name, `/\`) {
			names = append(names, name)
		}
	}
	info := torrent.Info
	if len(info.Files) > 0 {
		for _, file := range info.Files {
			parts := file.PathUTF8
			if len(parts) == 0 {
				parts = file.Path
			}
			if len(parts) > 0 {
				add(parts[len(parts)-1])
			}
		}
	} else if len(info.Tree) > 0 {
		var visit func(map[string]any, string, int)
		visit = func(tree map[string]any, name string, depth int) {
			if depth > 64 {
				return
			}
			if _, file := tree[""]; file {
				add(name)
				return
			}
			for name, value := range tree {
				if child, ok := value.(map[string]any); ok {
					visit(child, name, depth+1)
				}
			}
		}
		visit(info.Tree, "", 0)
	} else {
		name := info.NameUTF8
		if name == "" {
			name = info.Name
		}
		add(name)
	}
	// Offer media before sidecar files while keeping the result deterministic.
	media := func(name string) bool {
		switch strings.ToLower(path.Ext(name)) {
		case ".mkv", ".mp4", ".avi", ".webm", ".mov", ".m4v", ".ts", ".m2ts", ".wmv":
			return true
		}
		return false
	}
	sort.SliceStable(names, func(i, j int) bool {
		if media(names[i]) != media(names[j]) {
			return media(names[i])
		}
		return names[i] < names[j]
	})
	return names, nil
}
