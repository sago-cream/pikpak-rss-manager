package feed

import (
	"bytes"
	"context"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base32"
	"encoding/hex"
	"errors"
	"github.com/mmcdole/gofeed"
	"github.com/zeebo/bencode"
	"html"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

const MaxMetadata = 2 << 20

type Item struct{ Fingerprint, Title, URL string }
type Resource struct{ Key, URL string }
type Client struct{ HTTP *http.Client }

func New(allowPrivate bool) *Client {
	dialer := &net.Dialer{Timeout: 10 * time.Second}
	transport := &http.Transport{TLSHandshakeTimeout: 10 * time.Second, ResponseHeaderTimeout: 15 * time.Second, MaxIdleConns: 8, IdleConnTimeout: 60 * time.Second}
	transport.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(address)
		if err != nil {
			return nil, err
		}
		ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
		if err != nil {
			return nil, errors.New("RSS 主機無法解析")
		}
		if len(ips) == 0 {
			return nil, errors.New("RSS 主機沒有位址")
		}
		for _, ip := range ips {
			if !allowPrivate && (ip.IP.IsPrivate() || ip.IP.IsLoopback() || ip.IP.IsLinkLocalUnicast() || ip.IP.IsLinkLocalMulticast() || ip.IP.IsUnspecified() || ip.IP.IsMulticast()) {
				return nil, errors.New("RSS 位址指向私人或保留網路；需要明確允許")
			}
		}
		return dialer.DialContext(ctx, network, net.JoinHostPort(ips[0].IP.String(), port))
	}
	return &Client{HTTP: &http.Client{Transport: transport, Timeout: 25 * time.Second, CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) >= 5 {
			return errors.New("RSS 重新導向過多")
		}
		return ValidateURL(req.URL.String())
	}}}
}
func ValidateURL(v string) error {
	u, err := url.Parse(v)
	if err != nil || u.Hostname() == "" || (u.Scheme != "http" && u.Scheme != "https") || u.User != nil {
		return errors.New("RSS／種子連結必須是有效的 http 或 https URL，不支援網址內的帳號密碼")
	}
	return nil
}
func (c *Client) read(ctx context.Context, v string) ([]byte, error) {
	if err := ValidateURL(v); err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, "GET", v, nil)
	if err != nil {
		return nil, errors.New("無法建立來源請求")
	}
	req.Header.Set("User-Agent", "PikPak-RSS-Manager/0.1 (+https://github.com/wade00754/pikpak-rss-manager)")
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, errors.New("無法取得來源，請檢查網路或來源位址")
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, errors.New("來源回傳非成功狀態")
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, MaxMetadata+1))
	if err != nil {
		return nil, errors.New("來源讀取失敗")
	}
	if len(b) > MaxMetadata {
		return nil, errors.New("來源超過 2 MiB 中繼資料限制")
	}
	return b, nil
}

var magnetRE = regexp.MustCompile(`magnet:\?[^\s"'<>]+`)

func sourceLink(item *gofeed.Item, base *url.URL) string {
	for _, v := range []string{item.Link, item.Description, item.Content} {
		if m := magnetRE.FindString(html.UnescapeString(v)); m != "" {
			return m
		}
	}
	for _, enclosure := range item.Enclosures {
		u, err := url.Parse(enclosure.URL)
		if err != nil {
			continue
		}
		if strings.HasPrefix(u.String(), "magnet:") {
			return u.String()
		}
		if strings.Contains(strings.ToLower(enclosure.Type), "bittorrent") || strings.HasSuffix(strings.ToLower(u.Path), ".torrent") {
			return base.ResolveReference(u).String()
		}
	}
	if u, err := url.Parse(item.Link); err == nil && strings.HasSuffix(strings.ToLower(u.Path), ".torrent") {
		return base.ResolveReference(u).String()
	}
	return ""
}
func (c *Client) Fetch(ctx context.Context, v string) ([]Item, error) {
	b, err := c.read(ctx, v)
	if err != nil {
		return nil, err
	}
	return Parse(b, v)
}
func Parse(b []byte, baseURL string) ([]Item, error) {
	if len(b) > MaxMetadata {
		return nil, errors.New("RSS 過大")
	}
	f, err := gofeed.NewParser().Parse(bytes.NewReader(b))
	if err != nil {
		return nil, errors.New("RSS／Atom 格式無法解析")
	}
	base, err := url.Parse(baseURL)
	if err != nil {
		return nil, errors.New("RSS 位址無效")
	}
	out := []Item{}
	if len(f.Items) > 2000 {
		return nil, errors.New("RSS 項目超過 2000 筆")
	}
	for _, item := range f.Items {
		source := sourceLink(item, base)
		if source == "" {
			continue
		}
		key := item.GUID
		if key == "" {
			key = item.Title + "\n" + source
		}
		h := sha256.Sum256([]byte(key))
		title := strings.TrimSpace(item.Title)
		if len(title) > 8192 {
			continue
		}
		out = append(out, Item{hex.EncodeToString(h[:]), title, source})
	}
	return out, nil
}
func NormalizeMagnet(v string) (Resource, error) {
	u, err := url.Parse(html.UnescapeString(v))
	if err != nil || u.Scheme != "magnet" {
		return Resource{}, errors.New("Magnet 無效")
	}
	q, err := url.ParseQuery(u.RawQuery)
	if err != nil {
		return Resource{}, errors.New("Magnet 參數無效")
	}
	for _, xt := range q["xt"] {
		if strings.HasPrefix(strings.ToLower(xt), "urn:btih:") {
			hash := xt[len("urn:btih:"):]
			var b []byte
			if len(hash) == 40 {
				b, err = hex.DecodeString(hash)
			} else if len(hash) == 32 {
				b, err = base32.StdEncoding.DecodeString(strings.ToUpper(hash))
			} else {
				continue
			}
			if err == nil && len(b) == 20 {
				return Resource{"btih:" + hex.EncodeToString(b), u.String()}, nil
			}
		}
	}
	for _, xt := range q["xt"] {
		if strings.HasPrefix(strings.ToLower(xt), "urn:btmh:1220") {
			hash := strings.TrimPrefix(strings.ToLower(xt), "urn:btmh:1220")
			b, e := hex.DecodeString(hash)
			if e == nil && len(b) == 32 {
				return Resource{"btmh:1220" + hash, u.String()}, nil
			}
		}
	}
	return Resource{}, errors.New("Magnet 缺少有效的 v1/v2 infohash")
}
func (c *Client) Resolve(ctx context.Context, v string) (Resource, error) {
	if strings.HasPrefix(v, "magnet:") {
		return NormalizeMagnet(v)
	}
	b, err := c.read(ctx, v)
	if err != nil {
		return Resource{}, err
	}
	return Torrent(b)
}
func Torrent(b []byte) (Resource, error) {
	if len(b) == 0 || len(b) > MaxMetadata {
		return Resource{}, errors.New("種子大小無效")
	}
	if err := boundedBencode(b); err != nil {
		return Resource{}, err
	}
	var torrent struct {
		Info         bencode.RawMessage `bencode:"info"`
		Announce     string             `bencode:"announce"`
		AnnounceList [][]string         `bencode:"announce-list"`
	}
	if err := bencode.DecodeBytes(b, &torrent); err != nil || len(torrent.Info) == 0 {
		return Resource{}, errors.New("種子缺少有效 info 區段")
	}
	var info struct {
		Name        string `bencode:"name"`
		MetaVersion int    `bencode:"meta version"`
		Pieces      string `bencode:"pieces"`
	}
	if err := bencode.DecodeBytes(torrent.Info, &info); err != nil || info.Name == "" {
		return Resource{}, errors.New("種子內容無效")
	}
	q := url.Values{"dn": {info.Name}}
	var key string
	if info.MetaVersion == 2 && info.Pieces == "" {
		h := sha256.Sum256(torrent.Info)
		key = "btmh:1220" + hex.EncodeToString(h[:])
		q.Add("xt", "urn:"+key)
	} else {
		h := sha1.Sum(torrent.Info)
		key = "btih:" + hex.EncodeToString(h[:])
		q.Add("xt", "urn:"+key)
	}
	if torrent.Announce != "" {
		q.Add("tr", torrent.Announce)
	}
	for _, tier := range torrent.AnnounceList {
		for _, tracker := range tier {
			if len(q["tr"]) < 20 {
				q.Add("tr", tracker)
			}
		}
	}
	return Resource{key, "magnet:?" + q.Encode()}, nil
}

// Bound lengths and nesting before the reusable decoder handles untrusted bytes.
func boundedBencode(b []byte) error {
	depth := 0
	for i := 0; i < len(b); {
		ch := b[i]
		switch {
		case ch == 'd' || ch == 'l':
			depth++
			if depth > 64 {
				return errors.New("種子巢狀結構過深")
			}
			i++
		case ch == 'e':
			depth--
			i++
		case ch == 'i':
			end := bytes.IndexByte(b[i+1:], 'e')
			if end < 0 || end > 24 {
				return errors.New("種子整數無效")
			}
			i += end + 2
		case ch >= '0' && ch <= '9':
			length := 0
			count := 0
			for i < len(b) && b[i] >= '0' && b[i] <= '9' {
				count++
				if count > 8 {
					return errors.New("種子字串長度無效")
				}
				length = length*10 + int(b[i]-'0')
				i++
			}
			if i >= len(b) || b[i] != ':' || length > len(b)-i-1 {
				return errors.New("種子字串超出限制")
			}
			i += length + 1
		default:
			return errors.New("種子不是有效 bencode")
		}
		if depth < 0 {
			return errors.New("種子結構無效")
		}
	}
	if depth != 0 {
		return errors.New("種子結構不完整")
	}
	return nil
}
