package feed

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/zeebo/bencode"
)

func TestSourceSamplesDistinguishFilenamesFromTitles(t *testing.T) {
	metadata, err := bencode.EncodeBytes(map[string]any{"info": map[string]any{
		"name": "發布合集", "pieces": "01234567890123456789", "piece length": 16384,
		"files": []any{
			map[string]any{"length": 1, "path": []string{"subs", "字幕.ass"}},
			map[string]any{"length": 1, "path": []string{"original.mkv"}, "path.utf-8": []string{"作品 [01].mkv"}},
		}}})
	if err != nil {
		t.Fatal(err)
	}
	var requests atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if r.URL.Path == "/episode.torrent" {
			_, _ = w.Write(metadata)
			return
		}
		fmt.Fprint(w, `<rss version="2.0"><channel><title>測試</title>
<item><title>發布名稱 / 不是檔名</title><enclosure url="episode.torrent?private=metadata-only" type="application/x-bittorrent"/></item>
<item><title>Magnet 發布名稱</title><link>magnet:?xt=urn:btih:`+testHash+`&amp;dn=Display%20Name.mkv</link></item>
<item><title>[字幕組] 作品 / Work [01]</title><link>magnet:?xt=urn:btih:`+testHash+`</link></item>
</channel></rss>`)
	}))
	defer srv.Close()
	samples, err := New(true).Samples(context.Background(), srv.URL+"/feed")
	if err != nil || len(samples.Items) != 4 || requests.Load() != 2 {
		t.Fatal(samples, requests.Load(), err)
	}
	wants := []Sample{{"發布名稱 / 不是檔名", "作品 [01].mkv", "torrent_file"}, {"發布名稱 / 不是檔名", "字幕.ass", "torrent_file"}, {"Magnet 發布名稱", "Display Name.mkv", "magnet_name"}, {"[字幕組] 作品 / Work [01]", "[字幕組] 作品 / Work [01]", "rss_title"}}
	for i, want := range wants {
		if samples.Items[i] != want {
			t.Fatalf("sample %d: %+v", i, samples.Items[i])
		}
	}
	body, _ := json.Marshal(samples)
	if strings.Contains(string(body), "private=metadata-only") || strings.Contains(string(body), testHash) {
		t.Fatal("source URLs or infohash escaped in filename samples")
	}
}

func TestTorrentFileSamplesV2AndBadMetadata(t *testing.T) {
	metadata, _ := bencode.EncodeBytes(map[string]any{"info": map[string]any{
		"name": "v2 pack", "meta version": 2,
		"file tree": map[string]any{"season": map[string]any{"影片.mp4": map[string]any{"": map[string]any{"length": 1}}}},
	}})
	names, err := torrentNames(metadata)
	if err != nil || len(names) != 1 || names[0] != "影片.mp4" {
		t.Fatal("v2 file tree failed", names, err)
	}
	if _, err := torrentNames([]byte(`d4:info99999999:x`)); err == nil {
		t.Fatal("unsafe torrent bytes accepted")
	}
	var requests atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if r.URL.Path != "/feed" {
			http.Error(w, "private-response", 503)
			return
		}
		fmt.Fprint(w, `<rss version="2.0"><channel><title>Test</title>`)
		for i := 0; i < 7; i++ {
			fmt.Fprintf(w, `<item><title>Episode %d</title><enclosure url="%d.torrent" type="application/x-bittorrent"/></item>`, i, i)
		}
		fmt.Fprint(w, `</channel></rss>`)
	}))
	defer srv.Close()
	samples, err := New(true).Samples(context.Background(), srv.URL+"/feed")
	if err != nil || len(samples.Items) != 3 || len(samples.Notices) != 3 || requests.Load() != 4 || samples.NextCursor == "" {
		t.Fatal("sample network bounds or title fallback failed", samples, requests.Load(), err)
	}
	second, err := New(true).SamplesPage(context.Background(), srv.URL+"/feed", samples.NextCursor)
	if err != nil || len(second.Items) != 3 || len(second.Notices) != 3 || requests.Load() != 8 || second.NextCursor == "" {
		t.Fatal("failed torrents were skipped between pages", second, requests.Load(), err)
	}
	last, err := New(true).SamplesPage(context.Background(), srv.URL+"/feed", second.NextCursor)
	if err != nil || len(last.Items) != 1 || last.Items[0].Filename != "Episode 6" || requests.Load() != 10 || last.NextCursor != "" {
		t.Fatal("last failed torrent was not visited", last, requests.Load(), err)
	}
	if _, err := New(false).Samples(context.Background(), srv.URL+"/feed"); err == nil {
		t.Fatal("sample fetching bypassed private-network protection")
	}
}

func TestSamplePagingReadsEveryTorrentWithinPageBudgets(t *testing.T) {
	var mu sync.Mutex
	reads := map[string]int{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/feed" {
			mu.Lock()
			reads[r.URL.Path]++
			mu.Unlock()
			metadata, _ := bencode.EncodeBytes(map[string]any{"info": map[string]any{"name": strings.TrimPrefix(r.URL.Path, "/") + ".mkv", "length": 1, "piece length": 16384, "pieces": "01234567890123456789"}})
			_, _ = w.Write(metadata)
			return
		}
		fmt.Fprint(w, `<rss version="2.0"><channel><title>分頁測試</title>`)
		for i := 0; i < 2; i++ {
			fmt.Fprintf(w, `<item><guid>magnet-%d</guid><title>Magnet %d</title><link>magnet:?xt=urn:btih:%s&amp;dn=Display%d</link></item>`, i, i, testHash, i)
		}
		for i := 0; i < 7; i++ {
			fmt.Fprintf(w, `<item><guid>torrent-%d</guid><title>Episode %d</title><enclosure url="%d.torrent?private=torrent-sentinel" type="application/x-bittorrent"/></item>`, i, i, i)
		}
		fmt.Fprint(w, `</channel></rss>`)
	}))
	defer srv.Close()
	client := New(true)
	cursor := ""
	filenames := []string{}
	for page := 0; page < 3; page++ {
		samples, err := client.SamplesPage(context.Background(), srv.URL+"/feed?private=feed-sentinel", cursor)
		if err != nil || len(samples.Items) > sampleNamesPerPage {
			t.Fatal("page failed", samples, err)
		}
		torrents := 0
		for _, sample := range samples.Items {
			if sample.Kind == "torrent_file" {
				torrents++
				filenames = append(filenames, sample.Filename)
			}
		}
		if torrents > sampleTorrentsPerPage {
			t.Fatal("page exceeded torrent budget", torrents)
		}
		body, _ := json.Marshal(samples)
		if strings.Contains(string(body), "sentinel") || strings.Contains(string(body), srv.URL) {
			t.Fatal("paging exposed source URLs")
		}
		cursor = samples.NextCursor
	}
	if cursor != "" || len(filenames) != 7 {
		t.Fatal("torrent pages were truncated", filenames, cursor)
	}
	mu.Lock()
	defer mu.Unlock()
	for i, filename := range filenames {
		if filename != fmt.Sprintf("%d.torrent.mkv", i) || reads[fmt.Sprintf("/%d.torrent", i)] != 1 {
			t.Fatal("torrent skipped or fetched twice", i, filename, reads)
		}
	}
}

func TestSamplePagingContinuesInsideLargeTorrentAndDetectsChanges(t *testing.T) {
	var feedVersion, namesVersion atomic.Int32
	var torrentReads atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/pack.torrent" {
			torrentReads.Add(1)
			files := []any{}
			for i := 0; i < 65; i++ {
				files = append(files, map[string]any{"length": 1, "path": []string{fmt.Sprintf("v%d-episode-%02d.mkv", namesVersion.Load(), i)}})
			}
			metadata, _ := bencode.EncodeBytes(map[string]any{"info": map[string]any{"name": "Collection", "files": files, "piece length": 16384, "pieces": "01234567890123456789"}})
			_, _ = w.Write(metadata)
			return
		}
		fmt.Fprintf(w, `<rss version="2.0"><channel><title>合集</title><item><guid>v%d</guid><title>Collection</title><enclosure url="pack.torrent" type="application/x-bittorrent"/></item></channel></rss>`, feedVersion.Load())
	}))
	defer srv.Close()
	client := New(true)
	cursor := ""
	all := []Sample{}
	for _, count := range []int{30, 30, 5} {
		page, err := client.SamplesPage(context.Background(), srv.URL+"/feed", cursor)
		if err != nil || len(page.Items) != count {
			t.Fatal("multi-file page failed", page, err)
		}
		all = append(all, page.Items...)
		cursor = page.NextCursor
	}
	if cursor != "" || len(all) != 65 || torrentReads.Load() != 3 {
		t.Fatal("multi-file pagination did not finish", len(all), torrentReads.Load())
	}
	for i, sample := range all {
		if sample.Filename != fmt.Sprintf("v0-episode-%02d.mkv", i) {
			t.Fatal("multi-file name skipped or duplicated", i, sample)
		}
	}
	first, err := client.Samples(context.Background(), srv.URL+"/feed")
	if err != nil {
		t.Fatal(err)
	}
	feedVersion.Add(1)
	if _, err := client.SamplesPage(context.Background(), srv.URL+"/feed", first.NextCursor); err == nil || !strings.Contains(err.Error(), "RSS 內容已更新") || torrentReads.Load() != 4 {
		t.Fatal("changed feed accepted old cursor or read more torrents", err, torrentReads.Load())
	}
	feedVersion.Store(0)
	namesVersion.Add(1)
	if _, err := client.SamplesPage(context.Background(), srv.URL+"/feed", first.NextCursor); err == nil || !strings.Contains(err.Error(), "種子內容已更新") {
		t.Fatal("changed torrent accepted old file offset", err)
	}
}

func TestSamplePagingRejectsMalformedCursorBeforeRequests(t *testing.T) {
	var requests atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { requests.Add(1) }))
	defer srv.Close()
	revision := strings.Repeat("a", 64)
	for _, cursor := range []string{"bad", strings.Repeat(":", 1000), "-1:0:" + revision + ":", "2000:0:" + revision + ":", "0:-1:" + revision + ":", "0:1:" + revision + ":", "0:0:" + revision + ":" + revision} {
		if _, err := New(true).SamplesPage(context.Background(), srv.URL, cursor); err == nil {
			t.Fatal("malformed cursor accepted")
		}
	}
	if requests.Load() != 0 {
		t.Fatal("malformed cursors triggered network requests")
	}
}

func TestSamplePrefersTorrentWithoutChangingDownloadSource(t *testing.T) {
	metadata, _ := bencode.EncodeBytes(map[string]any{"info": map[string]any{"name": "Actual filename.mkv", "length": 1, "piece length": 16384, "pieces": "01234567890123456789"}})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/episode.torrent" {
			_, _ = w.Write(metadata)
			return
		}
		fmt.Fprint(w, `<rss version="2.0"><channel><title>Both sources</title><item><title>Episode</title><link>magnet:?xt=urn:btih:`+testHash+`&amp;dn=Different%20display%20name</link><enclosure url="episode.torrent" type="application/x-bittorrent"/></item></channel></rss>`)
	}))
	defer srv.Close()
	client := New(true)
	items, err := client.Fetch(context.Background(), srv.URL+"/feed")
	if err != nil || len(items) != 1 || !strings.HasPrefix(items[0].URL, "magnet:") {
		t.Fatal("download source priority changed", items, err)
	}
	samples, err := client.Samples(context.Background(), srv.URL+"/feed")
	if err != nil || len(samples.Items) != 1 || samples.Items[0].Kind != "torrent_file" || samples.Items[0].Filename != "Actual filename.mkv" {
		t.Fatal("available torrent filenames lost behind Magnet", samples, err)
	}
}
