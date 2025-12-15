package httprpc

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRegisterHandler_TypedMiddlewareOrder(t *testing.T) {
	r := New()

	var calls []string
	mw1 := func(next Handler[struct{}, int]) Handler[struct{}, int] {
		return func(ctx context.Context, req struct{}) (int, error) {
			calls = append(calls, "mw1-before")
			res, err := next(ctx, req)
			calls = append(calls, "mw1-after")
			if err != nil {
				return res, fmt.Errorf("mw1: %w", err)
			}
			return res, nil
		}
	}
	mw2 := func(next Handler[struct{}, int]) Handler[struct{}, int] {
		return func(ctx context.Context, req struct{}) (int, error) {
			calls = append(calls, "mw2-before")
			res, err := next(ctx, req)
			calls = append(calls, "mw2-after")
			if err != nil {
				return res, fmt.Errorf("mw2: %w", err)
			}
			return res, nil
		}
	}

	RegisterHandler[struct{}, int](
		r.EndpointGroup,
		GET(func(context.Context, struct{}) (int, error) {
			calls = append(calls, "handler")
			return http.StatusOK, nil
		}, "/ping"),
		WithCodec[struct{}, int](statusCodec{}),
		WithMiddlewares[struct{}, int](mw1, mw2),
	)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ping", http.NoBody)
	r.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, rec.Code)
	}

	want := []string{"mw1-before", "mw2-before", "handler", "mw2-after", "mw1-after"}
	if len(calls) != len(want) {
		t.Fatalf("calls: %v (want %v)", calls, want)
	}
	for i := range want {
		if calls[i] != want[i] {
			t.Fatalf("calls[%d]=%q want %q (calls=%v)", i, calls[i], want[i], calls)
		}
	}
}
