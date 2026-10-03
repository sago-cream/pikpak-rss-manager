package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStartupIgnoresLegacyCredentialsAndDotEnv(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := os.WriteFile(".env", []byte("invalid dotenv syntax\nAPP_ADMIN_PASSWORD=do-not-import\nPIKPAK_TOKEN=do-not-import\nAPP_LISTEN=invalid\n"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("APP_LISTEN", "127.0.0.1:8080")
	t.Setenv("APP_DATA_DIR", "data")
	t.Setenv("APP_ADMIN_PASSWORD", "ignored-legacy-password")
	t.Setenv("APP_ADMIN_PASSWORD_FILE", filepath.Join(t.TempDir(), "missing-password"))
	t.Setenv("PIKPAK_TOKEN", "ignored-legacy-token")
	t.Setenv("PIKPAK_TOKEN_FILE", filepath.Join(t.TempDir(), "missing-token"))
	t.Setenv("APP_PUBLIC_URL", "invalid-ignored-legacy-origin")
	t.Setenv("APP_ALLOW_PRIVATE_FEEDS", "true")
	cfg, err := Load()
	if err != nil || cfg.Listen != "127.0.0.1:8080" || cfg.DataDir != "data" {
		t.Fatal("startup still requires external credentials", err)
	}
	contents, err := os.ReadFile(".env")
	if err != nil || len(contents) == 0 {
		t.Fatal("legacy file was altered")
	}
}

func TestPublicURLIsAnOrigin(t *testing.T) {
	for _, origin := range []string{"", "https://rss.example.com", "https://rss.example.com:8443/", "http://127.0.0.1:8080"} {
		if _, err := NormalizePublicURL(origin); err != nil {
			t.Fatal("rejected valid origin", err)
		}
	}
	for _, origin := range []string{"https://rss.example.com/path", "https://rss.example.com?token=private", "https://rss.example.com?", "https://rss.example.com#fragment", "https://rss.example.com#", "https://user:password@rss.example.com"} {
		if _, err := NormalizePublicURL(origin); err == nil {
			t.Fatal("accepted non-origin URL")
		}
	}
}
