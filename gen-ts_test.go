package httprpc

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type pingReq struct {
	Name string `json:"name"`
}

type pingRes struct {
	Ok bool `json:"ok"`
}

func TestRouterGenTS_EmitsClientAndTypes(t *testing.T) {
	r := New()
	RegisterHandler(r.EndpointGroup, POST(func(context.Context, pingReq) (pingRes, error) {
		return pingRes{Ok: true}, nil
	}, "/ping"))

	outDir := t.TempDir()
	if err := r.GenTSDir(outDir, TSGenOptions{PackageName: "httprpc-test", ClientName: "API"}); err != nil {
		t.Fatalf("GenTSDir error: %v", err)
	}

	index, err := os.ReadFile(filepath.Clean(filepath.Join(outDir, "index.ts")))
	if err != nil {
		t.Fatalf("read index.ts: %v", err)
	}
	mod, err := os.ReadFile(filepath.Clean(filepath.Join(outDir, "ping.ts")))
	if err != nil {
		t.Fatalf("read ping.ts: %v", err)
	}

	if !strings.Contains(string(index), "export class API") {
		t.Fatalf("expected API class in index.ts")
	}
	if !strings.Contains(string(mod), "export class PingClient") {
		t.Fatalf("expected PingClient class in ping.ts")
	}
	if !strings.Contains(string(mod), "async post_ping") {
		t.Fatalf("expected endpoint method")
	}
	if !strings.Contains(string(mod), "export interface pingReq") || !strings.Contains(string(mod), "export interface pingRes") {
		t.Fatalf("expected request/response interfaces")
	}
	if !strings.Contains(string(mod), "name: string") || !strings.Contains(string(mod), "ok: boolean") {
		t.Fatalf("expected field type mappings")
	}
	if !strings.Contains(string(mod), "/ping") {
		t.Fatalf("expected route literal")
	}
}

func TestRouterGenTS_EmitsPathParams(t *testing.T) {
	type getUserReq struct {
		ID int `json:"id"`
	}
	type getUserRes struct {
		ID int `json:"id"`
	}

	r := New()
	RegisterHandler(r.EndpointGroup, GET(func(context.Context, getUserReq) (getUserRes, error) {
		return getUserRes{}, nil
	}, "/users/:id"))

	outDir := t.TempDir()
	if err := r.GenTSDir(outDir, TSGenOptions{PackageName: "httprpc-test", ClientName: "API"}); err != nil {
		t.Fatalf("GenTSDir error: %v", err)
	}

	mod, err := os.ReadFile(filepath.Clean(filepath.Join(outDir, "users.ts")))
	if err != nil {
		t.Fatalf("read users.ts: %v", err)
	}
	if !strings.Contains(string(mod), "params: {id: string | number }") {
		t.Fatalf("expected path params signature in generated client")
	}
	if !strings.Contains(string(mod), "/users/:id") {
		t.Fatalf("expected path template in generated client")
	}
}

func TestRouterGenTS_EmitsHeaderParams(t *testing.T) {
	type getUserReq struct {
		ID int `json:"id"`
	}
	type getUserMeta struct {
		Auth      string `header:"authorization"`
		RequestID string `header:"x-request-id,omitempty"`
	}
	type getUserRes struct {
		ID int `json:"id"`
	}

	r := New()
	RegisterHandlerWithMeta(r.EndpointGroup, GETM(func(context.Context, getUserReq, getUserMeta) (getUserRes, error) {
		return getUserRes{}, nil
	}, "/users/:id"))

	outDir := t.TempDir()
	if err := r.GenTSDir(outDir, TSGenOptions{PackageName: "httprpc-test", ClientName: "API"}); err != nil {
		t.Fatalf("GenTSDir error: %v", err)
	}

	mod, err := os.ReadFile(filepath.Clean(filepath.Join(outDir, "users.ts")))
	if err != nil {
		t.Fatalf("read users.ts: %v", err)
	}
	if !strings.Contains(string(mod), "headers: {authorization: string, \"x-request-id\"?: string }") {
		t.Fatalf("expected headers signature in generated client")
	}
}
