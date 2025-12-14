package httprpc

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRouterHandler_TSClientGenConfig_UsesChecksum(t *testing.T) {
	outDir := filepath.Join(t.TempDir(), "tsclient")

	r := NewRouter()
	r.SetTSClientGenConfig(&TSClientGenConfig{Dir: outDir})

	RegisterHandler[struct{}, struct{}](r.EndpointGroup, POST(HandlerFunc[struct{}, struct{}](func(context.Context, struct{}) (struct{}, error) {
		return struct{}{}, nil
	}), "/v1/ping"))

	_ = r.Handler()

	b, err := os.ReadFile(filepath.Join(outDir, "base.ts"))
	if err != nil {
		t.Fatalf("read base.ts: %v", err)
	}
	if !strings.Contains(string(b), "export const __httprpc_checksum") {
		t.Fatalf("expected checksum in base.ts")
	}

	if _, err := os.Stat(filepath.Join(outDir, tsClientChecksumFileName)); err != nil {
		t.Fatalf("expected %s to exist: %v", tsClientChecksumFileName, err)
	}

	sum1, err := readTSClientChecksum(outDir)
	if err != nil {
		t.Fatalf("read checksum: %v", err)
	}

	RegisterHandler[struct{}, struct{}](r.EndpointGroup, POST(HandlerFunc[struct{}, struct{}](func(context.Context, struct{}) (struct{}, error) {
		return struct{}{}, nil
	}), "/v1/ping2"))

	_ = r.Handler()

	sum2, err := readTSClientChecksum(outDir)
	if err != nil {
		t.Fatalf("read checksum: %v", err)
	}
	if sum1 == sum2 {
		t.Fatalf("expected checksum to change when endpoints change")
	}
}
