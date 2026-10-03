package feed

import (
	"context"
	"errors"
	"net/url"
	"path"
	"sort"
	"strings"
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

// Samples reads feed/torrent metadata only. Magnet display names and RSS titles
// are labeled separately because they need not match a downloaded filename.
func (c *Client) Samples(ctx context.Context, feedURL string) (Samples, error) {
	out := Samples{Items: []Sample{}, Notices: []string{}}
	items, err := c.Fetch(ctx, feedURL)
	if err != nil {
		return out, err
	}
	torrentReads := 0
	for i, item := range items {
		if i >= 5 || len(out.Items) >= 30 {
			break
		}
		if err := ctx.Err(); err != nil {
			return out, errors.New("來源範例讀取逾時，請稍後再試")
		}
		if strings.HasPrefix(item.URL, "magnet:") {
			if _, err := NormalizeMagnet(item.URL); err == nil {
				u, _ := url.Parse(item.URL)
				if name := u.Query().Get("dn"); validSampleName(name) {
					out.Items = append(out.Items, Sample{item.Title, name, "magnet_name"})
					continue
				}
			}
		} else if torrentReads < 3 {
			torrentReads++
			metadata, err := c.read(ctx, item.URL)
			if err == nil {
				var names []string
				names, err = torrentNames(metadata)
				if err == nil && len(names) > 0 {
					for _, name := range names {
						if len(out.Items) >= 30 {
							break
						}
						out.Items = append(out.Items, Sample{item.Title, name, "torrent_file"})
					}
					continue
				}
			}
			out.Notices = append(out.Notices, "部分種子中繼資料無法取得，該項目改以 RSS 標題示意。")
		}
		if item.Title != "" {
			out.Items = append(out.Items, Sample{item.Title, item.Title, "rss_title"})
		}
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
	if len(names) > 30 {
		names = names[:30]
	}
	return names, nil
}
