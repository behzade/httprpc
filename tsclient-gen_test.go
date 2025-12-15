package httprpc

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRouterGenerateTSClient_GeneratesFiles(t *testing.T) {
	outDir := filepath.Join(t.TempDir(), "tsclient")

	r := New()
	r.SetTSClientGenConfig(&TSClientGenConfig{Dir: outDir})

	RegisterHandler[struct{}, struct{}](r.EndpointGroup, POST(func(context.Context, struct{}) (struct{}, error) {
		return struct{}{}, nil
	}, "/v1/ping"))

	if err := r.GenerateTSClient(); err != nil {
		t.Fatalf("generate ts client: %v", err)
	}

	b, err := os.ReadFile(filepath.Clean(filepath.Join(outDir, "base.ts")))
	if err != nil {
		t.Fatalf("read base.ts: %v", err)
	}
	if !strings.Contains(string(b), "export async function request") {
		t.Fatalf("expected request helper in base.ts")
	}

	modFile := filepath.Join(outDir, "v1.ts")
	b, err = os.ReadFile(filepath.Clean(modFile))
	if err != nil {
		t.Fatalf("read module file: %v", err)
	}
	if !strings.Contains(string(b), "async post_v1_ping") {
		t.Fatalf("expected generated endpoint in module")
	}

	if _, statErr := os.Stat(filepath.Join(outDir, ".httprpc-checksum")); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("checksum file should not be created")
	}

	// ensure regeneration updates files when routes change
	RegisterHandler[struct{}, struct{}](r.EndpointGroup, POST(func(context.Context, struct{}) (struct{}, error) {
		return struct{}{}, nil
	}, "/v1/ping2"))

	if genErr := r.GenerateTSClient(); genErr != nil {
		t.Fatalf("generate ts client: %v", genErr)
	}

	b, err = os.ReadFile(filepath.Clean(modFile))
	if err != nil {
		t.Fatalf("read module file after regenerate: %v", err)
	}
	if !strings.Contains(string(b), "async post_v1_ping2") {
		t.Fatalf("expected regenerated client to include new endpoint")
	}
}
