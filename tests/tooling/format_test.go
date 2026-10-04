package tooling

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestGoFormatting(t *testing.T) {
	if os.Getenv("FORMAT_CHECK") != "1" {
		t.Skip("run make check-format to check formatting")
	}
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "gofmt", "-l", "cmd", "internal", "tests")
	cmd.Dir = filepath.Join("..", "..")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("gofmt failed: %v\n%s", err, output)
	}
	if paths := strings.TrimSpace(string(output)); paths != "" {
		t.Fatalf("Go files require formatting:\n%s", paths)
	}
}
