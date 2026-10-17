package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/wade00754/pikpak-rss-manager/internal/config"
	"github.com/wade00754/pikpak-rss-manager/internal/store"
)

func issueRecoveryCode() error {
	c, err := config.Load()
	if err != nil {
		return err
	}
	// Recovery must use the existing installation, never create a new database
	// or replace a missing encryption key in a mistyped data directory.
	for _, name := range []string{"manager.db", "secret.key"} {
		if _, err := os.Stat(filepath.Join(c.DataDir, name)); err != nil {
			return fmt.Errorf("cannot find the existing %s in APP_DATA_DIR", name)
		}
	}
	db, err := store.Open(c.DataDir)
	if err != nil {
		return err
	}
	defer db.Close()
	code, err := db.IssuePasswordRecovery(context.Background(), time.Now())
	if err != nil {
		return err
	}
	fmt.Printf("Recovery code (valid for 15 minutes, one use):\n%s\n", code)
	return nil
}
