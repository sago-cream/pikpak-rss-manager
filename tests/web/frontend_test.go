package web_test

import (
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/wade00754/pikpak-rss-manager/internal/store"
	"github.com/wade00754/pikpak-rss-manager/internal/web"
)

func TestEmbeddedFrontendAssets(t *testing.T) {
	db, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	server, err := web.New(db, nil, nil, "frontend-fixture")
	if err != nil {
		t.Fatal(err)
	}
	handler := server.Handler()
	page := httptest.NewRecorder()
	handler.ServeHTTP(page, httptest.NewRequest("GET", "/", nil))
	body := page.Body.String()
	if page.Code != 200 || !strings.Contains(body, `data-page="setup"`) || !strings.Contains(body, `data-version="frontend-fixture"`) || strings.Contains(body, "{{") {
		t.Fatal("frontend bootstrap was not rendered")
	}
	if page.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("bootstrap may be cached across authentication states")
	}
	paths := regexp.MustCompile(`(?:src|href)="(/ui/[^"]+)"`).FindAllStringSubmatch(body, -1)
	if len(paths) < 3 {
		t.Fatal("compiled frontend asset references missing")
	}
	for _, match := range paths {
		asset := httptest.NewRecorder()
		handler.ServeHTTP(asset, httptest.NewRequest("GET", match[1], nil))
		if asset.Code != 200 || asset.Body.Len() == 0 {
			t.Fatalf("embedded asset unavailable: %s", match[1])
		}
		if strings.HasSuffix(match[1], ".js") && !strings.Contains(asset.Header().Get("Content-Type"), "javascript") {
			t.Fatal("incorrect module content type")
		}
	}
	protected := httptest.NewRecorder()
	handler.ServeHTTP(protected, httptest.NewRequest("GET", "/api/subscriptions", nil))
	if protected.Code != 401 {
		t.Fatal("static asset access bypassed API authentication")
	}
}
