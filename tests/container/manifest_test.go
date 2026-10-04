package container

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"
	"time"
)

func checkArchitectures(data string) error {
	var index struct {
		Manifests []struct {
			Platform struct{ OS, Architecture string }
		}
	}
	if err := json.Unmarshal([]byte(data), &index); err != nil {
		return errors.New("invalid image manifest JSON")
	}
	platforms := make(map[string]bool)
	for _, manifest := range index.Manifests {
		platforms[manifest.Platform.OS+"/"+manifest.Platform.Architecture] = true
	}
	if !platforms["linux/amd64"] || !platforms["linux/arm64"] {
		return errors.New("image must include both linux/amd64 and linux/arm64")
	}
	return nil
}

func TestPublishedArchitectures(t *testing.T) {
	if os.Getenv("CONTAINER_MANIFEST_TEST") != "1" {
		t.Skip("run make test-manifest to inspect a published image with Docker")
	}
	image := os.Getenv("SMOKE_IMAGE")
	if strings.TrimSpace(image) == "" {
		t.Fatal("SMOKE_IMAGE is required")
	}
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
	defer cancel()
	manifest, err := runDocker(ctx, "buildx", "imagetools", "inspect", image, "--raw")
	if err != nil {
		t.Fatal(err)
	}
	if err := checkArchitectures(manifest); err != nil {
		t.Fatal(err)
	}
}

func TestArchitectureValidation(t *testing.T) {
	for _, test := range []struct {
		name, manifest string
		valid          bool
	}{
		{"both with attestation", `{"manifests":[{"platform":{"os":"linux","architecture":"arm64"}},{"platform":{"os":"linux","architecture":"amd64"}},{"platform":{"os":"unknown","architecture":"unknown"}}]}`, true},
		{"missing arm64", `{"manifests":[{"platform":{"os":"linux","architecture":"amd64"}}]}`, false},
		{"wrong OS", `{"manifests":[{"platform":{"os":"windows","architecture":"amd64"}},{"platform":{"os":"linux","architecture":"arm64"}}]}`, false},
		{"single image", `{"schemaVersion":2,"config":{}}`, false},
		{"malformed", `{"manifests":`, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			if err := checkArchitectures(test.manifest); (err == nil) != test.valid {
				t.Fatalf("unexpected architecture validation result: %v", err)
			}
		})
	}
}
