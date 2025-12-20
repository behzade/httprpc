package httprpc

import (
	"context"
	_ "embed"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

//go:embed testdata/tsclient/runtime-test.ts
var tsClientRuntimeSource string

func TestTSClientRuntime(t *testing.T) {
	if _, err := exec.LookPath("esbuild"); err != nil {
		t.Fatalf("esbuild not available: %v", err)
	}
	if _, err := exec.LookPath("node"); err != nil {
		t.Fatalf("node not available: %v", err)
	}

	type req struct {
		Q string `json:"q"`
	}
	type meta struct {
		Authorization string `header:"authorization"`
	}
	type res struct {
		ID int `json:"id"`
	}

	r := New()
	RegisterHandlerM[req, meta, res](r.EndpointGroup, GETM(func(context.Context, req, meta) (res, error) {
		return res{ID: 1}, nil
	}, "/users/:id"))

	outDir := t.TempDir()
	if err := r.GenTSDir(outDir, TSGenOptions{PackageName: "httprpc-test", ClientName: "API"}); err != nil {
		t.Fatalf("GenTSDir error: %v", err)
	}

	entry := tsClientRuntimeSource

	entryPath := filepath.Join(outDir, "runtime-test.ts")
	if err := os.WriteFile(entryPath, []byte(entry), 0o644); err != nil {
		t.Fatalf("write entry: %v", err)
	}

	outFile := filepath.Join(outDir, "runtime-test.mjs")
	runCmd := func(args ...string) {
		t.Helper()
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = outDir
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("%s: %v\n%s", strings.Join(args, " "), err, string(out))
		}
	}

	runCmd("esbuild", entryPath, "--bundle", "--platform=node", "--format=esm", "--outfile="+outFile)
	runCmd("node", outFile)
}
