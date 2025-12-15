package httprpc

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRouterGenerateTSClient_UsesChecksum(t *testing.T) {
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
	if !strings.Contains(string(b), "export const __httprpc_checksum") {
		t.Fatalf("expected checksum in base.ts")
	}

	if _, statErr := os.Stat(filepath.Join(outDir, tsClientChecksumFileName)); statErr != nil {
		t.Fatalf("expected %s to exist: %v", tsClientChecksumFileName, statErr)
	}

	sum1, err := readTSClientChecksum(outDir)
	if err != nil {
		t.Fatalf("read checksum: %v", err)
	}

	RegisterHandler[struct{}, struct{}](r.EndpointGroup, POST(func(context.Context, struct{}) (struct{}, error) {
		return struct{}{}, nil
	}, "/v1/ping2"))

	if genErr := r.GenerateTSClient(); genErr != nil {
		t.Fatalf("generate ts client: %v", genErr)
	}

	sum2, err := readTSClientChecksum(outDir)
	if err != nil {
		t.Fatalf("read checksum: %v", err)
	}
	if sum1 == sum2 {
		t.Fatalf("expected checksum to change when endpoints change")
	}
}
