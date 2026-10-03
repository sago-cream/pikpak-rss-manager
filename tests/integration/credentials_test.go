package integration

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/wade00754/pikpak-rss-manager/internal/store"
)

func liveToken(t *testing.T) string {
	t.Helper()
	dir := os.Getenv("PIKPAK_LIVE_DATA_DIR")
	if dir == "" {
		dir = filepath.Join("..", "..", "data")
	}
	if _, err := os.Stat(filepath.Join(dir, "manager.db")); err != nil {
		t.Fatal("initialize the Web UI and bind PAT first; set PIKPAK_LIVE_DATA_DIR for a custom data directory")
	}
	db, err := store.Open(dir)
	if err != nil {
		t.Fatal("encrypted local credentials unavailable")
	}
	defer db.Close()
	token, err := db.Setting(context.Background(), "pikpak_token")
	if err != nil || token == "" {
		t.Fatal("bind PAT in the Web UI before running live tests")
	}
	return token
}
