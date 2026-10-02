package config

import (
	"errors"
	"github.com/joho/godotenv"
	"net/url"
	"os"
	"strings"
)

type Config struct {
	Listen, DataDir, AdminPassword, PublicURL, Token, TokenSource string
	AllowPrivateFeeds                                             bool
}

func Load() (Config, error) {
	// An existing process environment always wins over the local development file.
	if _, err := os.Stat(".env"); err == nil {
		if err := godotenv.Load(".env"); err != nil {
			return Config{}, errors.New("無法解析 .env 設定（內容已隱藏）")
		}
	}
	c := Config{Listen: env("APP_LISTEN", "127.0.0.1:8080"), DataDir: env("APP_DATA_DIR", "data"), AdminPassword: os.Getenv("APP_ADMIN_PASSWORD"), PublicURL: strings.TrimRight(os.Getenv("APP_PUBLIC_URL"), "/"), AllowPrivateFeeds: os.Getenv("APP_ALLOW_PRIVATE_FEEDS") == "true"}
	if f := os.Getenv("APP_ADMIN_PASSWORD_FILE"); f != "" {
		b, err := os.ReadFile(f)
		if err != nil {
			return c, errors.New("無法讀取管理密碼檔案")
		}
		c.AdminPassword = strings.TrimSpace(string(b))
	}
	if len(c.AdminPassword) < 12 || len(c.AdminPassword) > 72 {
		return c, errors.New("請設定 12–72 位元組的 APP_ADMIN_PASSWORD 或 APP_ADMIN_PASSWORD_FILE；沒有預設密碼")
	}
	if c.PublicURL != "" {
		u, err := url.Parse(c.PublicURL)
		if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") || u.Path != "" || u.User != nil || u.RawQuery != "" || u.ForceQuery || strings.Contains(c.PublicURL, "#") {
			return c, errors.New("APP_PUBLIC_URL 必須是沒有路徑的 http/https 網站來源")
		}
	}
	if f := os.Getenv("PIKPAK_TOKEN_FILE"); f != "" {
		b, err := os.ReadFile(f)
		if err != nil {
			return c, errors.New("無法讀取 PikPak 權杖檔案")
		}
		c.Token, c.TokenSource = strings.TrimSpace(string(b)), "file"
		if c.Token == "" {
			return c, errors.New("PikPak 權杖檔案為空；不會改用其他帳號的已保存權杖")
		}
	} else if t := strings.TrimSpace(os.Getenv("PIKPAK_TOKEN")); t != "" {
		c.Token, c.TokenSource = t, "environment"
	}
	return c, nil
}

func env(k, fallback string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return fallback
}
