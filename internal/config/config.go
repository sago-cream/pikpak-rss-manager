package config

import (
	"errors"
	"net/url"
	"os"
	"strings"
)

// Credentials and application preferences are initialized in the Web UI.
type Config struct{ Listen, DataDir string }

func Load() (Config, error) {
	return Config{Listen: env("APP_LISTEN", "127.0.0.1:8080"), DataDir: env("APP_DATA_DIR", "data")}, nil
}

func NormalizePublicURL(value string) (string, error) {
	value = strings.TrimRight(strings.TrimSpace(value), "/")
	if value == "" {
		return "", nil
	}
	u, err := url.Parse(value)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") || u.Path != "" || u.User != nil || u.RawQuery != "" || u.ForceQuery || strings.Contains(value, "#") {
		return "", errors.New("網站網址必須是沒有路徑的 http/https 網站來源")
	}
	return value, nil
}

func env(k, fallback string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return fallback
}
