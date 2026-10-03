package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAdminPasswordsHaveNoLengthOrCharacterRules(t *testing.T) {
	t.Setenv("APP_ADMIN_PASSWORD_FILE", "")
	t.Setenv("APP_PUBLIC_URL", "")
	t.Setenv("PIKPAK_TOKEN", "")
	t.Setenv("PIKPAK_TOKEN_FILE", "")
	for name, password := range map[string]string{
		"one character": "x",
		"unicode":       "密碼🔑",
		"long":          strings.Repeat("任意長密碼", 30),
		"spaces":        " leading and trailing spaces ",
	} {
		t.Run(name, func(t *testing.T) {
			t.Setenv("APP_ADMIN_PASSWORD", password)
			c, err := Load()
			if err != nil || c.AdminPassword != password {
				t.Fatal("environment password rejected or changed")
			}
			file := filepath.Join(t.TempDir(), "password.txt")
			if err := os.WriteFile(file, []byte(password+"\r\n"), 0600); err != nil {
				t.Fatal(err)
			}
			t.Setenv("APP_ADMIN_PASSWORD_FILE", file)
			c, err = Load()
			if err != nil || c.AdminPassword != password {
				t.Fatal("file password rejected or changed")
			}
		})
	}
	t.Setenv("APP_ADMIN_PASSWORD", "")
	if _, err := Load(); err == nil {
		t.Fatal("unconfigured password accepted")
	}
}

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
