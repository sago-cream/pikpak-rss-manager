package feed

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/zeebo/bencode"
)

func TestDownloadChoicesRetainSuccessfulMetadataAndProvenance(t *testing.T) {
	metadata, _ := bencode.EncodeBytes(map[string]any{"info": map[string]any{"name": "Fixture", "meta version": 2, "file tree": map[string]any{"Episode.mkv": map[string]any{"": map[string]any{"length": 1}}}}})
	var reads atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/feed":
			fmt.Fprint(w, `<rss version="2.0"><channel><title>Fixture</title><item><guid>one</guid><title>Good</title><enclosure url="/one.torrent" type="application/x-bittorrent"/></item><item><guid>two</guid><title>Duplicate URL</title><enclosure url="/one.torrent" type="application/x-bittorrent"/></item><item><title>Bad</title><enclosure url="/bad.torrent" type="application/x-bittorrent"/></item><item><title>Mismatched</title><link>magnet:?xt=urn:btih:0123456789abcdef0123456789abcdef01234567</link><enclosure url="/one.torrent" type="application/x-bittorrent"/></item><item><title>Unknown names</title><link>magnet:?xt=urn:btih:0123456789abcdef0123456789abcdef01234567&amp;dn=Not-an-actual-filename</link></item></channel></rss>`)
		case "/one.torrent":
			reads.Add(1)
			w.Write(metadata)
		default:
			w.Write([]byte(strings.Repeat("x", MaxMetadata+1)))
		}
	}))
	defer server.Close()
	blocked := New(false)
	defer blocked.HTTP.CloseIdleConnections()
	if _, _, err := blocked.DownloadChoices(context.Background(), server.URL+"/feed"); err == nil {
		t.Fatal("private feed accepted")
	}
	c := New(true)
	defer c.HTTP.CloseIdleConnections()
	choices, _, err := c.DownloadChoices(context.Background(), server.URL+"/feed")
	if err != nil || len(choices) != 5 || reads.Load() != 1 {
		t.Fatal("snapshot or distinct metadata reads", err, len(choices), reads.Load())
	}
	if choices[0].Filenames[0] != "Episode.mkv" || !strings.HasPrefix(choices[0].Resource.Key, "btmh:") || choices[1].Resource.Key != choices[0].Resource.Key {
		t.Fatal("v2 filenames or resource identity lost")
	}
	if choices[2].Error == "" || choices[2].Resource.Key != "" {
		t.Fatal("oversized metadata selectable")
	}
	if choices[3].Error == "" || len(choices[3].Filenames) != 0 {
		t.Fatal("mismatched metadata shown as downloaded filenames")
	}
	if len(choices[4].Filenames) != 0 || choices[4].Resource.Key == "" {
		t.Fatal("magnet display name guessed as a real filename")
	}
}
