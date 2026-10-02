package main

import (
	"context"
	"fmt"
	"github.com/wade00754/pikpak-rss-manager/internal/app"
	"github.com/wade00754/pikpak-rss-manager/internal/config"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	_ "time/tzdata"
)

var version = "0.1.0-dev"

func main() {
	command := "serve"
	if len(os.Args) > 1 {
		command = os.Args[1]
	}
	switch command {
	case "version", "--version":
		fmt.Println(version)
		return
	case "healthcheck":
		address := os.Getenv("APP_LISTEN")
		if address == "" {
			address = "127.0.0.1:8080"
		}
		host, port, err := net.SplitHostPort(address)
		if err != nil {
			os.Exit(1)
		}
		if host == "" || host == "0.0.0.0" || host == "::" {
			host = "127.0.0.1"
		}
		c := &http.Client{Timeout: 3 * time.Second}
		resp, err := c.Get("http://" + net.JoinHostPort(host, port) + "/healthz")
		if err != nil {
			os.Exit(1)
		}
		resp.Body.Close()
		if resp.StatusCode != 200 {
			os.Exit(1)
		}
		return
	case "serve":
	default:
		fmt.Fprintln(os.Stderr, "用法：pikpak-rss-manager [serve|healthcheck|version]")
		os.Exit(2)
	}
	c, err := config.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	if err := app.Run(ctx, c, version); err != nil {
		fmt.Fprintln(os.Stderr, "服務啟動失敗：", err)
		os.Exit(1)
	}
}
