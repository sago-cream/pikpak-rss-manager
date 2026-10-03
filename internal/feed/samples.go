package feed

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"path"
	"sort"
	"strconv"
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
	Items      []Sample `json:"items"`
	Notices    []string `json:"notices"`
	NextCursor string   `json:"next_cursor,omitempty"`
}

const sampleEntriesPerPage = 5
const sampleTorrentsPerPage = 3
const sampleNamesPerPage = 30

// Samples reads feed/torrent metadata only. Magnet display names and RSS titles
// are labeled separately because they need not match a downloaded filename.
func (c *Client) Samples(ctx context.Context, feedURL string) (Samples, error) {
	return c.SamplesPage(ctx, feedURL, "")
}

// SamplesPage continues at the next entry or filename without skipping sources
// when a page's metadata budget is exhausted. Cursors contain only offsets and
// content digests, never feed credentials or resource URLs.
func (c *Client) SamplesPage(ctx context.Context, feedURL, cursor string) (Samples, error) {
	out := Samples{Items: []Sample{}, Notices: []string{}}
	index, fileOffset, revision, namesRevision, err := parseSampleCursor(cursor)
	if err != nil {
		return out, err
	}
	items, err := c.fetch(ctx, feedURL, true)
	if err != nil {
		return out, err
	}
	currentRevision := sampleRevision(items)
	if cursor != "" && (revision != currentRevision || index >= len(items)) {
		return out, errors.New("RSS 內容已更新，請重新讀取 RSS 範例")
	}
	torrentReads := 0
	for scanned := 0; index < len(items) && scanned < sampleEntriesPerPage && len(out.Items) < sampleNamesPerPage; scanned++ {
		if err := ctx.Err(); err != nil {
			return out, errors.New("來源範例讀取逾時，請稍後再試")
		}
		item := items[index]
		if strings.HasPrefix(item.URL, "magnet:") {
			if fileOffset != 0 {
				return out, errors.New("來源範例續讀位置無效，請重新讀取 RSS 範例")
			}
			if _, err := NormalizeMagnet(item.URL); err == nil {
				u, _ := url.Parse(item.URL)
				if name := u.Query().Get("dn"); validSampleName(name) {
					out.Items = append(out.Items, Sample{item.Title, name, "magnet_name"})
					index++
					continue
				}
			}
		} else {
			if torrentReads >= sampleTorrentsPerPage {
				break
			}
			torrentReads++
			metadata, err := c.read(ctx, item.URL)
			var names []string
			if err == nil {
				names, err = torrentNames(metadata)
				if err == nil && len(names) > 0 {
					currentNamesRevision := filenameRevision(names)
					if fileOffset >= len(names) || (fileOffset > 0 && namesRevision != currentNamesRevision) {
						return out, errors.New("種子內容已更新，請重新讀取 RSS 範例")
					}
					end := min(len(names), fileOffset+sampleNamesPerPage-len(out.Items))
					for _, name := range names[fileOffset:end] {
						out.Items = append(out.Items, Sample{item.Title, name, "torrent_file"})
					}
					if end < len(names) {
						out.NextCursor = sampleCursor(index, end, currentRevision, currentNamesRevision)
						return out, nil
					}
					fileOffset, namesRevision = 0, ""
					index++
					continue
				}
			}
			if fileOffset > 0 {
				return out, errors.New("種子檔名續讀失敗，請稍後再試或重新讀取 RSS 範例")
			}
			out.Notices = append(out.Notices, "部分種子中繼資料無法取得，可手動輸入檔名。")
		}
		if item.Title != "" {
			out.Items = append(out.Items, Sample{item.Title, item.Title, "rss_title"})
		}
		index++
	}
	if index < len(items) {
		out.NextCursor = sampleCursor(index, 0, currentRevision, "")
	}
	return out, nil
}

func sampleRevision(items []Item) string {
	h := sha256.New()
	for _, item := range items {
		fmt.Fprintf(h, "%s\x00%s\x00%s\x00", item.Fingerprint, item.Title, item.URL)
	}
	return hex.EncodeToString(h.Sum(nil))
}

func filenameRevision(names []string) string {
	digest := sha256.Sum256([]byte(strings.Join(names, "\x00")))
	return hex.EncodeToString(digest[:])
}

func sampleCursor(index, fileOffset int, revision, namesRevision string) string {
	return fmt.Sprintf("%d:%d:%s:%s", index, fileOffset, revision, namesRevision)
}

func parseSampleCursor(cursor string) (index, fileOffset int, revision, namesRevision string, err error) {
	if cursor == "" {
		return
	}
	err = errors.New("來源範例續讀位置無效，請重新讀取 RSS 範例")
	if len(cursor) > 160 {
		return
	}
	parts := strings.Split(cursor, ":")
	if len(parts) != 4 {
		return
	}
	index, indexErr := strconv.Atoi(parts[0])
	fileOffset, offsetErr := strconv.Atoi(parts[1])
	revisionBytes, revisionErr := hex.DecodeString(parts[2])
	namesBytes, namesErr := hex.DecodeString(parts[3])
	if indexErr != nil || offsetErr != nil || index < 0 || index >= 2000 || fileOffset < 0 || revisionErr != nil || len(revisionBytes) != sha256.Size || namesErr != nil || (fileOffset == 0 && len(namesBytes) != 0) || (fileOffset > 0 && len(namesBytes) != sha256.Size) {
		return
	}
	return index, fileOffset, parts[2], parts[3], nil
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
