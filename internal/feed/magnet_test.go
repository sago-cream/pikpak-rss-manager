package feed

import (
	"net/url"
	"reflect"
	"strings"
	"testing"

	"github.com/zeebo/bencode"
)

func TestCanonicalMagnetPreservesParametersAndHybridTopics(t *testing.T) {
	v2 := "urn:btmh:1220" + strings.Repeat("a", 64)
	q := url.Values{"dn": {"[Fixture] A / B & C.mkv"}, "xt": {v2, "urn:btih:" + strings.ToUpper(testHash)}, "tr": {"https://tracker.test/announce?x=1&y=2", "udp://tracker.test:80"}, "xs": {"https://fixture.test/file.torrent?token=fixture"}}
	r, err := NormalizeMagnet("magnet:?" + q.Encode())
	if err != nil || !strings.HasPrefix(r.URL, "magnet:?xt=urn:btih:"+testHash+"&") || r.Key != "btih:"+testHash {
		t.Fatal("conventional normalized infohash prefix missing", err)
	}
	u, _ := url.Parse(r.URL)
	got := u.Query()
	for _, k := range []string{"dn", "tr", "xs"} {
		if !reflect.DeepEqual(got[k], q[k]) {
			t.Fatal("Magnet parameter changed", k)
		}
	}
	if !reflect.DeepEqual(got["xt"], []string{"urn:btih:" + testHash, v2}) {
		t.Fatal("hybrid topics lost")
	}
	again, err := NormalizeMagnet(r.URL)
	if err != nil || again != r {
		t.Fatal("canonicalization is not idempotent")
	}
	r, err = NormalizeMagnet("magnet:?dn=Fixture&xt=" + url.QueryEscape(v2))
	if err != nil || !strings.HasPrefix(r.URL, "magnet:?xt="+v2+"&") {
		t.Fatal("v2 Magnet prefix missing")
	}
}

func TestTorrentMagnetStartsWithInfohash(t *testing.T) {
	b, err := bencode.EncodeBytes(map[string]any{"info": map[string]any{"name": "Fixture.txt", "length": 3, "piece length": 16384, "pieces": strings.Repeat("a", 20)}, "announce": "https://tracker.test/announce"})
	if err != nil {
		t.Fatal(err)
	}
	r, err := Torrent(b)
	if err != nil || !strings.HasPrefix(r.URL, "magnet:?xt=urn:"+r.Key+"&dn=") {
		t.Fatal("torrent-generated Magnet lacks conventional prefix", err)
	}
}
