package feed

import (
	"context"
	"crypto/sha1"
	"encoding/base32"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/zeebo/bencode"
)

const testHash = "0123456789abcdef0123456789abcdef01234567"

func TestRSSAndAtom(t *testing.T) {
	rss := `<rss version="2.0"><channel><title>測試</title><item><guid>one</guid><title>第1話</title><description><![CDATA[<a href="magnet:?xt=urn:btih:` + testHash + `&amp;dn=Episode">下載</a>]]></description></item><item><guid>two</guid><title>第2話</title><enclosure url="/files/2.torrent?secret=hidden" type="application/x-bittorrent"/></item><item><title>News</title><link>https://example.org/article</link></item></channel></rss>`
	items, err := Parse([]byte(rss), "https://example.org/rss/index.xml")
	if err != nil || len(items) != 2 {
		t.Fatal(len(items), err)
	}
	if !strings.Contains(items[0].URL, "&dn=Episode") || items[1].URL != "https://example.org/files/2.torrent?secret=hidden" {
		t.Fatal("RSS source extraction failed")
	}
	atom := `<feed xmlns="http://www.w3.org/2005/Atom"><title>Atom</title><entry><id>same</id><title>S01E03</title><link rel="enclosure" type="application/x-bittorrent" href="episode.torrent"/></entry></feed>`
	items, err = Parse([]byte(atom), "https://example.org/feed/")
	if err != nil || len(items) != 1 || items[0].URL != "https://example.org/feed/episode.torrent" {
		t.Fatal(items, err)
	}
}
func TestNormalizeMagnet(t *testing.T) {
	bytes, _ := hex.DecodeString(testHash)
	b32 := base32.StdEncoding.EncodeToString(bytes)
	for _, v := range []string{strings.ToUpper(testHash), b32, strings.ToLower(b32)} {
		r, err := NormalizeMagnet("magnet:?xt=urn:btih:" + v + "&tr=https%3A%2F%2Ftracker.test")
		if err != nil || r.Key != "btih:"+testHash {
			t.Fatal(r, err)
		}
	}
	if _, err := NormalizeMagnet("magnet:?xt=urn:btih:not-a-hash"); err == nil {
		t.Fatal("accepted invalid infohash")
	}
	r, err := NormalizeMagnet("magnet:?xt=urn:btmh:1220" + strings.Repeat("a", 64))
	if err != nil || !strings.HasPrefix(r.Key, "btmh:") {
		t.Fatal(r, err)
	}
}
func TestTorrentUsesOriginalInfoBytes(t *testing.T) {
	info := []byte("d6:lengthi3e4:name5:a.txt12:piece lengthi16384e6:pieces20:01234567890123456789e")
	b, err := bencode.EncodeBytes(map[string]any{"info": bencode.RawMessage(info), "announce": "https://tracker.test/announce"})
	if err != nil {
		t.Fatal(err)
	}
	r, err := Torrent(b)
	h := sha1.Sum(info)
	if err != nil || r.Key != "btih:"+hex.EncodeToString(h[:]) {
		t.Fatal(r, err)
	}
	for _, bad := range []string{"d4:info99999999:x", strings.Repeat("d", 65), "di999999999999999999999999999e"} {
		if _, err := Torrent([]byte(bad)); err == nil {
			t.Fatal("accepted malformed or allocation-bomb torrent")
		}
	}
}
func TestNetworkBoundsAndPrivateFeedOptIn(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, strings.Repeat("x", MaxMetadata+1)) }))
	defer srv.Close()
	if _, err := New(false).read(context.Background(), srv.URL); err == nil {
		t.Fatal("private addresses accepted without opt-in")
	}
	if _, err := New(true).read(context.Background(), srv.URL); err == nil || !strings.Contains(err.Error(), "2 MiB") {
		t.Fatal("metadata size bound failed", err)
	}
}
