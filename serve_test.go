package httprpc

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

type statusCodec struct{}

func (statusCodec) Decode(*http.Request) (struct{}, error) { return struct{}{}, nil }
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

	h := r.Handler()

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
