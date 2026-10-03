package integration

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/wade00754/pikpak-rss-manager/internal/pikpak"
	"github.com/wade00754/pikpak-rss-manager/internal/store"
)

func TestLiveFolderBrowsingAndCreation(t *testing.T) {
	if os.Getenv("PIKPAK_LIVE_FOLDERS_TEST") != "1" {
		t.Skip("opt-in real PikPak directory test")
	}
	// Read the encrypted UI credential in memory; never print it or remote names.
	token := liveToken(t)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	api, err := pikpak.New(ctx, token)
	if err != nil {
		t.Fatal("official MCP authorization failed")
	}
	defer api.Close()
	base, err := pikpak.EnsureFolder(ctx, api, "", "_pikpak-rss-manager-test")
	if err != nil {
		t.Fatal("test namespace unavailable")
	}
	name := "folders-run-" + time.Now().UTC().Format("20060102T150405") + "-" + store.ID()[:8]
	run, err := pikpak.AddDirectory(ctx, api, base, name)
	if err != nil {
		t.Fatal("test directory creation failed", err)
	}
	child, err := pikpak.AddDirectory(ctx, api, run.ID, "目錄選擇測試")
	if err != nil {
		t.Fatal("child directory creation failed", err)
	}
	listing, err := pikpak.BrowseDirectories(ctx, api, run.ID, "")
	if err != nil || len(listing.Folders) != 1 || listing.Folders[0].ID != child.ID {
		t.Fatal("official folder listing did not return created child")
	}
	crumbs, err := pikpak.DirectoryPath(ctx, api, child.ID)
	if err != nil || crumbs[len(crumbs)-1].Path != child.Path {
		t.Fatal("official ID/path lookup mismatch")
	}
	if _, err := pikpak.AddDirectory(ctx, api, run.ID, "目錄選擇測試"); err == nil {
		t.Fatal("duplicate folder creation was accepted")
	}
	t.Log("Verified official browse/create/ID lookup; preserved test directory:", run.Path)
}
