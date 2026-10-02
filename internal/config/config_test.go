package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPublicURLIsAnOrigin(t *testing.T) {
	t.Setenv("APP_ADMIN_PASSWORD", "unit-config-password-only")
	t.Setenv("APP_ADMIN_PASSWORD_FILE", "")
	t.Setenv("PIKPAK_TOKEN", "")
	t.Setenv("PIKPAK_TOKEN_FILE", "")
	for _, origin := range []string{"https://rss.example.com", "https://rss.example.com:8443/", "http://127.0.0.1:8080"} {
		t.Setenv("APP_PUBLIC_URL", origin)
		if _, err := Load(); err != nil {
			t.Fatal("rejected valid origin", err)
		}
	}
	for _, origin := range []string{"https://rss.example.com/path", "https://rss.example.com?token=private", "https://rss.example.com?", "https://rss.example.com#fragment", "https://rss.example.com#", "https://user:password@rss.example.com"} {
		t.Setenv("APP_PUBLIC_URL", origin)
		if _, err := Load(); err == nil {
			t.Fatal("accepted non-origin PUBLIC_URL")
		}
	}
}
func TestExplicitEmptyTokenFileDoesNotFallBack(t *testing.T) {
	t.Setenv("APP_ADMIN_PASSWORD", "unit-config-password-only")
	t.Setenv("APP_ADMIN_PASSWORD_FILE", "")
	t.Setenv("APP_PUBLIC_URL", "")
	t.Setenv("PIKPAK_TOKEN", "unit-env-token")
	file := filepath.Join(t.TempDir(), "token.txt")
	if err := os.WriteFile(file, []byte(" \n"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PIKPAK_TOKEN_FILE", file)
	if _, err := Load(); err == nil {
		t.Fatal("empty explicit file fell back to another credential source")
	}
}
