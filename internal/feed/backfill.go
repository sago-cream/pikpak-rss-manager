package feed

import (
	"context"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// DownloadChoice selects a whole torrent. Filenames are metadata, never guessed
// from release titles. URLs remain private and are kept only in the server plan.
type DownloadChoice struct {
	Item      Item
	Resource  Resource
	Filenames []string
	Error     string
}

func (c *Client) DownloadChoices(ctx context.Context, source string) ([]DownloadChoice, []string, error) {
	items, err := c.Fetch(ctx, source)
	if err != nil {
		return nil, nil, err
	}
	urls := []string{}
	indices := map[string]int{}
	for _, item := range items {
		u := item.MetadataURL
		if u == "" && !strings.HasPrefix(item.URL, "magnet:") {
			u = item.URL
		}
		if u != "" {
			if _, ok := indices[u]; !ok {
				indices[u] = len(urls)
				urls = append(urls, u)
			}
		}
	}
	type result struct {
		resource Resource
		names    []string
		err      error
	}
	results := make([]result, len(urls))
	readCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	var total atomic.Int64
	var limited atomic.Bool
	queue := make(chan int)
	var workers sync.WaitGroup
	for range min(5, len(urls)) {
		workers.Go(func() {
			for i := range queue {
				if readCtx.Err() != nil {
					continue
				}
				one, done := context.WithTimeout(readCtx, 15*time.Second)
				b, e := c.read(one, urls[i])
				done()
				results[i].err = e
				if e != nil {
					continue
				}
				if total.Add(int64(len(b))) > 32<<20 {
					limited.Store(true)
					cancel()
					continue
				}
				results[i].resource, results[i].err = Torrent(b)
				if results[i].err == nil {
					results[i].names, results[i].err = torrentNames(b)
				}
			}
		})
	}
enqueue:
	for i := range urls {
		select {
		case queue <- i:
		case <-readCtx.Done():
			break enqueue
		}
	}
	close(queue)
	workers.Wait()
	notices := []string{}
	if limited.Load() {
		notices = append(notices, "種子中繼資料總量超過 32 MiB，已停止讀取；請使用較小的 RSS。")
	}
	if ctx.Err() != nil {
		notices = append(notices, "種子檔名讀取逾時，已保留取得的檔名；可重新讀取。")
	}
	out := []DownloadChoice{}
	count, size := 0, 0
	for _, item := range items {
		choice := DownloadChoice{Item: item, Filenames: []string{}}
		u := item.MetadataURL
		if u == "" && !strings.HasPrefix(item.URL, "magnet:") {
			u = item.URL
		}
		if u != "" {
			r := results[indices[u]]
			choice.Filenames = r.names
			choice.Resource = r.resource
			if r.err != nil || r.resource.Key == "" {
				choice.Error = "無法取得種子檔名"
			}
		}
		if strings.HasPrefix(item.URL, "magnet:") {
			choice.Resource, err = NormalizeMagnet(item.URL)
			if err != nil {
				choice.Error = err.Error()
			} else if u != "" && results[indices[u]].resource.Key != "" && results[indices[u]].resource.Key != choice.Resource.Key {
				choice.Filenames = nil
				choice.Error = "種子中繼資料與下載來源不一致"
			}
		}
		if choice.Resource.Key == "" && choice.Error == "" {
			choice.Error = "無法取得種子檔名"
		}
		count += len(choice.Filenames)
		size += len(item.Title) + len(choice.Resource.URL) + len(item.URL) + len(item.MetadataURL)
		for _, name := range choice.Filenames {
			size += len(name)
		}
		if count > 10000 || size > 8<<20 {
			notices = append(notices, "檔名清單超過 10,000 筆或 8 MiB，已保留部分檔名；請使用較小的 RSS。")
			break
		}
		out = append(out, choice)
	}
	return out, notices, nil
}
