package httprpc

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHTTPMiddleware_OrderGroupAndPriority(t *testing.T) {
	r := New()

	var calls []string
	mw := func(name string) Middleware {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				calls = append(calls, name+"-before")
				next.ServeHTTP(w, req)
				calls = append(calls, name+"-after")
			})
		}
	}

	// Higher priority wraps outer, regardless of registration order.
	r.Use(mw("root-high"), Priority(10))
	r.Use(mw("root-low"), Priority(0))

	g := r.Group("/v1")
	g.Use(mw("group"))

	RegisterHandler(g, GET(func(context.Context, struct{}) (int, error) {
		calls = append(calls, "handler")
		return http.StatusOK, nil
	}, "/ping"), WithCodec[struct{}, int](statusCodec{}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/ping", http.NoBody)
	h, err := r.Handler()
	if err != nil {
		t.Fatalf("handler build error: %v", err)
	}
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, rec.Code)
	}

	want := []string{
		"root-high-before",
		"root-low-before",
		"group-before",
		"handler",
		"group-after",
		"root-low-after",
		"root-high-after",
	}
	if len(calls) != len(want) {
		t.Fatalf("calls: %v (want %v)", calls, want)
	}
	for i := range want {
		if calls[i] != want[i] {
			t.Fatalf("calls[%d]=%q want %q (calls=%v)", i, calls[i], want[i], calls)
		}
	}
}
