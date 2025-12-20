package httprpc

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

type statusCodec struct{}

func (statusCodec) DecodeBody(*http.Request) (struct{}, error)  { return struct{}{}, nil }
func (statusCodec) DecodeQuery(*http.Request) (struct{}, error) { return struct{}{}, nil }
func (statusCodec) Encode(w http.ResponseWriter, res int) error {
	w.WriteHeader(res)
	return nil
}

func (statusCodec) EncodeError(w http.ResponseWriter, err error) error {
	http.Error(w, err.Error(), http.StatusInternalServerError)
	return nil
}

func TestRouterHandler_DispatchesAndMethodNotAllowed(t *testing.T) {
	r := New()
	RegisterHandler[struct{}, int](r.EndpointGroup, GET(func(context.Context, struct{}) (int, error) {
		return http.StatusOK, nil
	}, "/ping"), WithCodec[struct{}, int](statusCodec{}))

	h, err := r.Handler()
	if err != nil {
		t.Fatalf("handler build error: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ping", http.NoBody)
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, rec.Code)
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/ping", http.NoBody)
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected %d, got %d", http.StatusMethodNotAllowed, rec.Code)
	}
	if allow := rec.Header().Get("Allow"); allow != http.MethodGet {
		t.Fatalf("expected Allow %q, got %q", http.MethodGet, allow)
	}
}

func TestRouterDescribe_CollectsTypeInfo(t *testing.T) {
	r := New()
	RegisterHandler[struct{}, int](r.EndpointGroup, GET(func(context.Context, struct{}) (int, error) {
		return http.StatusOK, nil
	}, "/ping"))

	desc := r.Describe()
	if len(desc) != 1 {
		t.Fatalf("expected 1 endpoint description, got %d", len(desc))
	}
	if desc[0].Method != http.MethodGet || desc[0].Path != "/ping" {
		t.Fatalf("unexpected description: %+v", desc[0])
	}
	if desc[0].Res.String == "" {
		t.Fatalf("expected res type string to be set")
	}
	if len(desc[0].Consumes) == 0 || len(desc[0].Produces) == 0 {
		t.Fatalf("expected consumes/produces to be set")
	}
}

func TestRouterFallback(t *testing.T) {
	r := New()

	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.Header().Set("X-Middleware", "applied")
			next.ServeHTTP(w, req)
		})
	})

	RegisterHandler[struct{}, int](r.EndpointGroup, GET(func(context.Context, struct{}) (int, error) {
		return http.StatusOK, nil
	}, "/ping"), WithCodec[struct{}, int](statusCodec{}))

	r.SetFallback(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	}))

	h, err := r.Handler()
	if err != nil {
		t.Fatalf("handler build error: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/missing", http.NoBody)
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusTeapot {
		t.Fatalf("expected fallback status %d, got %d", http.StatusTeapot, rec.Code)
	}
	if rec.Header().Get("X-Middleware") != "applied" {
		t.Fatalf("expected middleware to run on fallback")
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/ping", http.NoBody)
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, rec.Code)
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/ping", http.NoBody)
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected %d, got %d", http.StatusMethodNotAllowed, rec.Code)
	}
}

func TestRouterHandler_PathParams(t *testing.T) {
	type userReq struct {
		ID int `query:"id"`
	}
	type userMeta struct {
		ID int `path:"id"`
	}
	type userRes struct {
		ID      int    `json:"id"`
		QueryID int    `json:"query_id"`
		Param   string `json:"param"`
	}

	r := New()
	RegisterHandlerM[userReq, userMeta, userRes](r.EndpointGroup, GETM(func(_ context.Context, req userReq, meta userMeta) (userRes, error) {
		return userRes{ID: meta.ID, QueryID: req.ID, Param: strconv.Itoa(meta.ID)}, nil
	}, "/users/:id"))

	h, err := r.Handler()
	if err != nil {
		t.Fatalf("handler build error: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/users/42?id=5", http.NoBody)
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, rec.Code)
	}

	var res userRes
	if err := json.NewDecoder(rec.Body).Decode(&res); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if res.ID != 42 {
		t.Fatalf("expected id 42, got %d", res.ID)
	}
	if res.QueryID != 5 {
		t.Fatalf("expected query id 5, got %d", res.QueryID)
	}
	if res.Param != strconv.Itoa(res.ID) {
		t.Fatalf("expected param %q, got %q", strconv.Itoa(res.ID), res.Param)
	}
}

func TestRouterHandler_MetaHeaders(t *testing.T) {
	type req struct {
		Name string `query:"name"`
	}
	type meta struct {
		Auth string `header:"authorization"`
	}
	type res struct {
		Auth string `json:"auth"`
		Name string `json:"name"`
	}

	r := New()
	RegisterHandlerM[req, meta, res](r.EndpointGroup, GETM(func(_ context.Context, req req, meta meta) (res, error) {
		return res{Auth: meta.Auth, Name: req.Name}, nil
	}, "/me"))

	h, err := r.Handler()
	if err != nil {
		t.Fatalf("handler build error: %v", err)
	}

	rec := httptest.NewRecorder()
	reqHTTP := httptest.NewRequest(http.MethodGet, "/me?name=alice", http.NoBody)
	h.ServeHTTP(rec, reqHTTP)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d", http.StatusBadRequest, rec.Code)
	}

	rec = httptest.NewRecorder()
	reqHTTP = httptest.NewRequest(http.MethodGet, "/me?name=alice", http.NoBody)
	reqHTTP.Header.Set("Authorization", "Bearer token")
	h.ServeHTTP(rec, reqHTTP)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, rec.Code)
	}

	var got res
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.Auth != "Bearer token" {
		t.Fatalf("expected auth %q, got %q", "Bearer token", got.Auth)
	}
	if got.Name != "alice" {
		t.Fatalf("expected name %q, got %q", "alice", got.Name)
	}
}

func TestRouterHandler_PathParams_InvalidPattern(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		errHint string
	}{
		{name: "brace-syntax", path: "/users/{id}", errHint: "use :name"},
		{name: "not-snake-case", path: "/users/:UserID", errHint: "snake_case"},
		{name: "duplicate-name", path: "/users/:id/:id", errHint: "duplicate path param"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := New()
			RegisterHandler[struct{}, struct{}](r.EndpointGroup, GET(func(context.Context, struct{}) (struct{}, error) {
				return struct{}{}, nil
			}, tt.path))

			if _, err := r.Handler(); err == nil || !strings.Contains(err.Error(), tt.errHint) {
				t.Fatalf("expected error containing %q, got %v", tt.errHint, err)
			}
		})
	}
}

func TestRouterHandler_PathParams_Ambiguous(t *testing.T) {
	r := New()
	RegisterHandler[struct{}, struct{}](r.EndpointGroup, GET(func(context.Context, struct{}) (struct{}, error) {
		return struct{}{}, nil
	}, "/users/:id"))
	RegisterHandler[struct{}, struct{}](r.EndpointGroup, GET(func(context.Context, struct{}) (struct{}, error) {
		return struct{}{}, nil
	}, "/users/:user_id"))

	if _, err := r.Handler(); err == nil || !strings.Contains(err.Error(), "ambiguous route") {
		t.Fatalf("expected ambiguous route error, got %v", err)
	}
}
