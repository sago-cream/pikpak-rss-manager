package feed

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
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
	samples, err := New(true).SamplesAll(context.Background(), srv.URL+"/feed")
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
	samples, err := New(true).SamplesAll(context.Background(), srv.URL+"/feed")
	if err != nil || len(samples.Items) != 7 || len(samples.Notices) != 1 || requests.Load() != 8 {
		t.Fatal("sample network bounds or title fallback failed", samples, requests.Load(), err)
	}
	if _, err := New(false).SamplesAll(context.Background(), srv.URL+"/feed"); err == nil {
		t.Fatal("sample fetching bypassed private-network protection")
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
	samples, err := client.SamplesAll(context.Background(), srv.URL+"/feed")
	if err != nil || len(samples.Items) != 1 || samples.Items[0].Kind != "torrent_file" || samples.Items[0].Filename != "Actual filename.mkv" {
		t.Fatal("available torrent filenames lost behind Magnet", samples, err)
	}
}
