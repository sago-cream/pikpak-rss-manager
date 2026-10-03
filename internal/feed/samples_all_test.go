package feed

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/zeebo/bencode"
)

func TestSamplesAllReadsOnceConcurrentlyAndKeepsEveryFilename(t *testing.T) {
	var feedReads, active, peak atomic.Int32
	reads := make([]atomic.Int32, 7)
	release := make(chan struct{})
	var once sync.Once
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/feed" {
			feedReads.Add(1)
			fmt.Fprint(w, `<rss version="2.0"><channel><title>All</title>`)
			for i := 0; i < 8; i++ {
				fmt.Fprintf(w, `<item><guid>%d</guid><title>Episode %d</title><enclosure url="%d.torrent?private=sentinel" type="application/x-bittorrent"/></item>`, i, i, i%7)
			}
			fmt.Fprint(w, `<item><title>Display only</title><link>magnet:?xt=urn:btih:`+testHash+`&amp;dn=Display</link></item></channel></rss>`)
			return
		}
		i, _ := strconv.Atoi(strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/"), ".torrent"))
		reads[i].Add(1)
		n := active.Add(1)
		defer active.Add(-1)
		for old := peak.Load(); n > old && !peak.CompareAndSwap(old, n); old = peak.Load() {
		}
		if n == 5 {
			once.Do(func() { close(release) })
		}
		select {
		case <-release:
		case <-r.Context().Done():
			return
		}
		files := []any{}
		for j := 0; j < 65; j++ {
			files = append(files, map[string]any{"length": 1, "path": []string{fmt.Sprintf("episode-%d-%02d.mkv", i, j)}})
		}
		metadata, _ := bencode.EncodeBytes(map[string]any{"info": map[string]any{"name": "pack", "files": files, "pieces": "01234567890123456789", "piece length": 16384}})
		_, _ = w.Write(metadata)
	}))
	defer srv.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	out, err := New(true).SamplesAll(ctx, srv.URL+"/feed")
	if err != nil || len(out.Items) != 8*65+1 || len(out.Notices) != 0 || out.NextCursor != "" || feedReads.Load() != 1 || peak.Load() != 5 {
		t.Fatal("all-source read failed", len(out.Items), out.Notices, feedReads.Load(), peak.Load(), err)
	}
	for i := range reads {
		if reads[i].Load() != 1 {
			t.Fatal("torrent requested more than once", i, reads[i].Load())
		}
	}
	for i, sample := range out.Items[:8*65] {
		if sample.Kind != "torrent_file" || sample.Filename != fmt.Sprintf("episode-%d-%02d.mkv", (i/65)%7, i%65) {
			t.Fatal("filename missing or order changed", i, sample)
		}
	}
	body, _ := json.Marshal(out)
	if strings.Contains(string(body), "sentinel") || strings.Contains(string(body), testHash) {
		t.Fatal("private source escaped")
	}
}

func TestSamplesAllRetainsSuccessOnFailuresAndCancellation(t *testing.T) {
	metadata, _ := bencode.EncodeBytes(map[string]any{"info": map[string]any{"name": "actual.mkv", "length": 1, "pieces": "01234567890123456789", "piece length": 16384}})
	blocked := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/ok.torrent":
			w.Write(metadata)
		case "/bad.torrent":
			w.Write([]byte("invalid"))
		case "/slow.torrent":
			close(blocked)
			<-r.Context().Done()
		default:
			fmt.Fprint(w, `<rss version="2.0"><channel><title>Errors</title><item><title>OK</title><enclosure url="ok.torrent"/></item><item><title>Bad</title><enclosure url="bad.torrent"/></item><item><title>Slow</title><enclosure url="slow.torrent"/></item></channel></rss>`)
		}
	}))
	defer srv.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { <-blocked; time.Sleep(300 * time.Millisecond); cancel() }()
	start := time.Now()
	out, err := New(true).SamplesAll(ctx, srv.URL+"/feed")
	if err != nil || len(out.Items) != 3 || out.Items[0].Filename != "actual.mkv" || len(out.Notices) == 0 || time.Since(start) > 3*time.Second {
		t.Fatal("failed/cancelled source hid successful filenames", out, err)
	}
}

func TestSamplesAllKeepsPrivateNetworkProtection(t *testing.T) {
	var reads atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { reads.Add(1) }))
	defer srv.Close()
	if _, err := New(false).SamplesAll(context.Background(), srv.URL); err == nil || reads.Load() != 0 {
		t.Fatal("all-source read bypassed private-network protection")
	}
}

func TestSamplesAllMakesOutputLimitExplicit(t *testing.T) {
	files := []any{}
	for i := 0; i < 10001; i++ {
		files = append(files, map[string]any{"length": 1, "path": []string{fmt.Sprintf("file-%05d.mkv", i)}})
	}
	metadata, _ := bencode.EncodeBytes(map[string]any{"info": map[string]any{"name": "large", "files": files, "pieces": "01234567890123456789", "piece length": 16384}})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/all.torrent" {
			w.Write(metadata)
			return
		}
		fmt.Fprint(w, `<rss version="2.0"><channel><title>Large</title><item><title>Large</title><enclosure url="all.torrent"/></item></channel></rss>`)
	}))
	defer srv.Close()
	out, err := New(true).SamplesAll(context.Background(), srv.URL+"/feed")
	if err != nil || len(out.Items) != 10000 || len(out.Notices) == 0 || !strings.Contains(out.Notices[0], "10,000") {
		t.Fatal("output limit silently discarded filenames", len(out.Items), out.Notices, err)
	}
}

func TestSamplesAllBoundsTotalMetadata(t *testing.T) {
	metadata, _ := bencode.EncodeBytes(map[string]any{"padding": strings.Repeat("x", 1<<20), "info": map[string]any{"name": "actual.mkv", "length": 1, "pieces": "01234567890123456789", "piece length": 16384}})
	var requests atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, ".torrent") {
			requests.Add(1)
			w.Write(metadata)
			return
		}
		fmt.Fprint(w, `<rss version="2.0"><channel><title>Budget</title>`)
		for i := 0; i < 60; i++ {
			fmt.Fprintf(w, `<item><guid>%d</guid><title>Episode</title><enclosure url="%d.torrent"/></item>`, i, i)
		}
		fmt.Fprint(w, `</channel></rss>`)
	}))
	defer srv.Close()
	out, err := New(true).SamplesAll(context.Background(), srv.URL+"/feed")
	if err != nil || requests.Load() > 36 || requests.Load() < 32 || len(out.Notices) == 0 || !strings.Contains(out.Notices[0], "32 MiB") {
		t.Fatal("total metadata budget not enforced", requests.Load(), out.Notices, err)
	}
}
