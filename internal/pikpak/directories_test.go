package pikpak_test

import (
	"context"
	"errors"
	"testing"

	"github.com/wade00754/pikpak-rss-manager/internal/pikpak"
	"github.com/wade00754/pikpak-rss-manager/internal/testutil"
)

type lostDirectoryResponse struct{ *testutil.Cloud }

func (p lostDirectoryResponse) CreateFolder(ctx context.Context, parent, name string) (pikpak.File, error) {
	_, err := p.Cloud.CreateFolder(ctx, parent, name)
	if err != nil {
		return pikpak.File{}, err
	}
	return pikpak.File{}, errors.New("response lost after creation")
}

func TestUncertainDirectoryCreationDoesNotResubmit(t *testing.T) {
	c := testutil.NewCloud()
	api := lostDirectoryResponse{c}
	ctx := context.Background()
	if _, err := pikpak.AddDirectory(ctx, api, "", "新作品"); err == nil || c.Calls["mkdir"] != 1 {
		t.Fatal("uncertain creation was retried or reported as successful")
	}
	listing, err := pikpak.BrowseDirectories(ctx, api, "", "")
	if err != nil || len(listing.Folders) != 1 || listing.Folders[0].Name != "新作品" {
		t.Fatal("created folder could not be reconciled through browsing")
	}
	if _, err := pikpak.AddDirectory(ctx, api, "", "新作品"); err == nil || c.Calls["mkdir"] != 1 {
		t.Fatal("manual retry submitted a duplicate folder")
	}
}

type directoryPages struct{ *testutil.Cloud }

func (p directoryPages) List(ctx context.Context, parent, token string) (pikpak.Page, error) {
	if parent != "" {
		return p.Cloud.List(ctx, parent, token)
	}
	if token == "next" {
		return pikpak.Page{Files: []pikpak.File{{ID: "second", Name: "Z", Kind: "drive#folder"}}}, nil
	}
	return pikpak.Page{Files: []pikpak.File{{ID: "first", Name: "A", Kind: "drive#folder"}, {ID: "video", Name: "movie.mp4", Kind: "drive#file"}}, Next: "next"}, nil
}

func TestDirectoryBrowsingAndCreation(t *testing.T) {
	ctx := context.Background()
	c := testutil.NewCloud()
	c.Files["anime"] = pikpak.File{ID: "anime", Name: "Anime", Kind: "drive#folder"}
	c.Files["one"] = pikpak.File{ID: "one", Name: "作品", ParentID: "anime", Kind: "drive#folder"}
	c.Files["two"] = pikpak.File{ID: "two", Name: "作品", ParentID: "anime", Kind: "drive#folder"}
	c.Files["video"] = pikpak.File{ID: "video", Name: "movie.mp4", ParentID: "anime", Kind: "drive#file"}
	listing, err := pikpak.BrowseDirectories(ctx, c, "anime", "")
	if err != nil || len(listing.Folders) != 2 || len(listing.Breadcrumbs) != 2 || listing.Current.Path != "Anime" {
		t.Fatal(listing, err)
	}
	crumbs, err := pikpak.DirectoryPath(ctx, c, "two")
	if err != nil || crumbs[len(crumbs)-1].ID != "two" || crumbs[len(crumbs)-1].Path != "Anime/作品" {
		t.Fatal("duplicate name lost identity", err)
	}
	folder, err := pikpak.AddDirectory(ctx, c, "one", "新作品")
	if err != nil || folder.Path != "Anime/作品/新作品" || c.Files[folder.ID].ParentID != "one" {
		t.Fatal(folder, err)
	}
	if _, err := pikpak.AddDirectory(ctx, c, "one", "新作品"); err == nil || c.Calls["mkdir"] != 1 {
		t.Fatal("duplicate creation was sent")
	}
	if _, err := pikpak.AddDirectory(ctx, c, "video", "child"); err == nil {
		t.Fatal("created inside a file")
	}
	for _, name := range []string{"", ".", "..", "a/b", "a\\b", "a\nb", " trailing "} {
		if _, err := pikpak.AddDirectory(ctx, c, "", name); err == nil {
			t.Fatal("unsafe folder name accepted")
		}
	}
	c.Files["cycle"] = pikpak.File{ID: "cycle", Name: "cycle", Kind: "drive#folder", ParentID: "cycle"}
	if _, err := pikpak.DirectoryPath(ctx, c, "cycle"); err == nil {
		t.Fatal("cyclic parents accepted")
	}
	paged := directoryPages{c}
	first, err := pikpak.BrowseDirectories(ctx, paged, "", "")
	if err != nil || len(first.Folders) != 1 || first.NextToken != "next" {
		t.Fatal("first folder page", err)
	}
	second, err := pikpak.BrowseDirectories(ctx, paged, "", first.NextToken)
	if err != nil || len(second.Folders) != 1 || second.Folders[0].ID != "second" || second.NextToken != "" {
		t.Fatal("second folder page", err)
	}
}
