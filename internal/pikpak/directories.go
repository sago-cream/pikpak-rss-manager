package pikpak

import (
	"context"
	"errors"
	"sort"
	"strings"
	"unicode/utf8"
)

type Directory struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Path string `json:"path"`
}

type Directories struct {
	Current     Directory   `json:"current"`
	Breadcrumbs []Directory `json:"breadcrumbs"`
	Folders     []Directory `json:"folders"`
	NextToken   string      `json:"next_token"`
	AccountRef  string      `json:"account_ref"`
}

// DirectoryPath resolves by ID, so duplicate folder names remain distinguishable.
func DirectoryPath(ctx context.Context, api API, id string) ([]Directory, error) {
	root := Directory{Name: "我的 PikPak"}
	reversed := []File{}
	seen := map[string]bool{}
	for id != "" {
		if len(id) > 300 || seen[id] || len(reversed) >= 32 {
			return nil, errors.New("資料夾路徑過深或有循環，請在 PikPak 檢查")
		}
		seen[id] = true
		f, err := api.Get(ctx, id)
		if err != nil {
			return nil, Classify(err)
		}
		if f.ID != id || !f.Folder() {
			return nil, errors.New("選取的項目不是有效資料夾")
		}
		reversed = append(reversed, f)
		id = f.ParentID
	}
	out := []Directory{root}
	parts := []string{}
	for i := len(reversed) - 1; i >= 0; i-- {
		parts = append(parts, reversed[i].Name)
		out = append(out, Directory{ID: reversed[i].ID, Name: reversed[i].Name, Path: strings.Join(parts, "/")})
	}
	return out, nil
}

func BrowseDirectories(ctx context.Context, api API, parent, token string) (Directories, error) {
	out := Directories{Folders: []Directory{}}
	if len(token) > 4096 {
		return out, errors.New("資料夾分頁資訊過長")
	}
	crumbs, err := DirectoryPath(ctx, api, parent)
	if err != nil {
		return out, err
	}
	out.Breadcrumbs = crumbs
	out.Current = crumbs[len(crumbs)-1]
	page, err := api.List(ctx, parent, token)
	if err != nil {
		return out, Classify(err)
	}
	if token != "" && page.Next == token {
		return out, errors.New("資料夾分頁重複，請重新整理")
	}
	for _, file := range page.Files {
		if !file.Folder() || file.ID == "" {
			continue
		}
		path := file.Name
		if out.Current.Path != "" {
			path = out.Current.Path + "/" + path
		}
		out.Folders = append(out.Folders, Directory{ID: file.ID, Name: file.Name, Path: path})
	}
	sort.Slice(out.Folders, func(i, j int) bool {
		a, b := strings.ToLower(out.Folders[i].Name), strings.ToLower(out.Folders[j].Name)
		if a == b {
			return out.Folders[i].ID < out.Folders[j].ID
		}
		return a < b
	})
	out.NextToken = page.Next
	return out, nil
}

func ValidateDirectoryName(name string) error {
	if name == "" || name == "." || name == ".." || strings.TrimSpace(name) != name || len(name) > 240 || !utf8.ValidString(name) || strings.ContainsAny(name, "/\\\x00\r\n") {
		return errors.New("請填寫單一資料夾名稱，不能包含 /、\\、換行、. 或 ..")
	}
	return nil
}

func AddDirectory(ctx context.Context, api API, parent, name string) (Directory, error) {
	var out Directory
	if err := ValidateDirectoryName(name); err != nil {
		return out, err
	}
	crumbs, err := DirectoryPath(ctx, api, parent)
	if err != nil {
		return out, err
	}
	files, err := ListAll(ctx, api, parent)
	if err != nil {
		return out, Classify(err)
	}
	for _, file := range files {
		if strings.EqualFold(file.Name, name) {
			return out, errors.New("此位置已有同名項目，請選取現有資料夾或使用其他名稱")
		}
	}
	if err := ctx.Err(); err != nil {
		return out, Classify(err)
	}
	file, err := api.CreateFolder(ctx, parent, name)
	if err != nil {
		return out, Classify(err)
	}
	if file.ID != "" && (file.Kind == "" || file.Name == "") {
		file, err = api.Get(ctx, file.ID)
		if err != nil {
			return out, Classify(err)
		}
	}
	if file.ID == "" || !file.Folder() {
		return out, errors.New("建立資料夾結果不明，請重新整理確認")
	}
	path := name
	if current := crumbs[len(crumbs)-1]; current.Path != "" {
		path = current.Path + "/" + name
	}
	return Directory{ID: file.ID, Name: file.Name, Path: path}, nil
}
