package httprpc

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

type createUserReq struct {
	Name string `json:"name"`
}

type createUserRes struct {
	ID string `json:"id"`
}

type searchHotelsReq struct {
	City string `json:"city"`
}

type searchHotelsRes struct {
	IDs []string `json:"ids"`
}

func TestTSGenGolden(t *testing.T) {
	r := New()

	RegisterHandler(r.EndpointGroup, POST(
		func(context.Context, createUserReq) (createUserRes, error) {
			return createUserRes{}, nil
		},
		"/v1/users/create",
	))

	RegisterHandler(r.EndpointGroup, POST(
		func(context.Context, searchHotelsReq) (searchHotelsRes, error) {
			return searchHotelsRes{}, nil
		},
		"/v1/hotels/search",
	))

	outDir := t.TempDir()
	if err := r.GenTSDir(outDir, TSGenOptions{
		PackageName:      "httprpc-test",
		ClientName:       "API",
		SkipPathSegments: 1, // drop "v1"
	}); err != nil {
		t.Fatalf("GenTSDir error: %v", err)
	}

	goldenDir := filepath.Join("testdata", "tsgen")
	wantFiles := []string{tsClientChecksumFileName, "base.ts", "index.ts", "users.ts", "hotels.ts"}

	update := os.Getenv("UPDATE_GOLDEN") != ""
	if update {
		if err := os.MkdirAll(goldenDir, 0o750); err != nil {
			t.Fatalf("mkdir golden dir: %v", err)
		}
		for _, f := range wantFiles {
			b, err := os.ReadFile(filepath.Clean(filepath.Join(outDir, f)))
			if err != nil {
				t.Fatalf("read generated %s: %v", f, err)
			}
			if err := os.WriteFile(filepath.Join(goldenDir, f), b, 0o600); err != nil {
				t.Fatalf("write golden %s: %v", f, err)
			}
		}
	}

	for _, f := range wantFiles {
		gotPath := filepath.Join(outDir, f)
		wantPath := filepath.Join(goldenDir, f)

		got, err := os.ReadFile(filepath.Clean(gotPath))
		if err != nil {
			t.Fatalf("read generated %s: %v", gotPath, err)
		}
		want, err := os.ReadFile(filepath.Clean(wantPath))
		if err != nil {
			if !update {
				t.Skipf("missing golden %s (run with UPDATE_GOLDEN=1 to generate)", wantPath)
			}
			t.Fatalf("missing golden %s: %v", wantPath, err)
		}

		gs := normalizeLF(string(got))
		ws := normalizeLF(string(want))
		if gs != ws {
			t.Fatalf("%s mismatch\n--- want (%s)\n+++ got (%s)", f, wantPath, gotPath)
		}
	}

	// Ensure generator doesn't emit unexpected files for this router.
	entries, err := os.ReadDir(outDir)
	if err != nil {
		t.Fatalf("readdir: %v", err)
	}
	var gotNames []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		gotNames = append(gotNames, e.Name())
	}
	sort.Strings(gotNames)
	sort.Strings(wantFiles)
	if strings.Join(gotNames, ",") != strings.Join(wantFiles, ",") {
		t.Fatalf("unexpected generated files: %v (want %v)", gotNames, wantFiles)
	}
}

func normalizeLF(s string) string {
	return strings.ReplaceAll(s, "\r\n", "\n")
}
