package app

import (
	"context"
	"github.com/wade00754/pikpak-rss-manager/internal/config"
	"github.com/wade00754/pikpak-rss-manager/internal/feed"
	"github.com/wade00754/pikpak-rss-manager/internal/pikpak"
	"github.com/wade00754/pikpak-rss-manager/internal/store"
	"github.com/wade00754/pikpak-rss-manager/internal/web"
	"github.com/wade00754/pikpak-rss-manager/internal/worker"
	"log/slog"
	"net/http"
	"time"
)

func Run(ctx context.Context, c config.Config, version string) error {
	db, err := store.Open(c.DataDir)
	if err != nil {
		return err
	}
	defer db.Close()
	manager := pikpak.NewManager(db)
	defer manager.Close()
	w := worker.New(db, manager, feed.New(false))
	ui, err := web.New(c, db, manager, w, version)
	if err != nil {
		return err
	}
	server := &http.Server{Addr: c.Listen, Handler: ui.Handler(), ReadHeaderTimeout: 10 * time.Second, ReadTimeout: 30 * time.Second, WriteTimeout: 120 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 32 << 10}
	workerCtx, stopWorker := context.WithCancel(ctx)
	defer stopWorker()
	workerDone := make(chan struct{})
	go func() {
		defer close(workerDone)
		select {
		case <-workerCtx.Done():
			return
		case <-ui.Ready():
		}
		w.Gate.Lock()
		initCtx, cancel := context.WithTimeout(workerCtx, 45*time.Second)
		if err := manager.Initialize(initCtx); err != nil {
			slog.Warn("PikPak 尚未連線；可登入面板處理授權", "reason", pikpak.Classify(err).Message)
		}
		cancel()
		w.Gate.Unlock()
		w.Run(workerCtx)
	}()
	serverErr := make(chan error, 1)
	go func() {
		slog.Info("管理介面已啟動", "address", c.Listen, "version", version)
		serverErr <- server.ListenAndServe()
	}()
	select {
	case <-ctx.Done():
	case err := <-serverErr:
		if err != http.ErrServerClosed {
			stopWorker()
			<-workerDone
			return err
		}
	}
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	_ = server.Shutdown(shutdownCtx)
	stopWorker()
	<-workerDone
	return nil
}
