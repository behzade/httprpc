// Package httprpc provides tests for correctness of the httprpc library.
package httprpc

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

type testCodec[Req any, Res any] struct {
	decodeErr error

	decodeCalls      int
	encodeCalls      int
	encodeErrorCalls int
}

func (c *testCodec[Req, Res]) DecodeBody(*http.Request) (Req, error) {
	var zero Req
	c.decodeCalls++
	if c.decodeErr != nil {
		return zero, c.decodeErr
	}
	return zero, nil
}

func (c *testCodec[Req, Res]) DecodeQuery(*http.Request) (Req, error) {
	var zero Req
	c.decodeCalls++
	if c.decodeErr != nil {
		return zero, c.decodeErr
	}
	return zero, nil
}

func (c *testCodec[Req, Res]) Encode(http.ResponseWriter, Res) error {
	c.encodeCalls++
	return nil
}

func (c *testCodec[Req, Res]) EncodeError(w http.ResponseWriter, err error) error {
	c.encodeErrorCalls++
	http.Error(w, err.Error(), http.StatusBadRequest)
	return nil
}

func TestAdaptHandler_ReturnsAfterDecodeError(t *testing.T) {
	codec := &testCodec[struct{}, struct{}]{decodeErr: errors.New("bad request")}
	called := false
	handler := func(context.Context, struct{}) (struct{}, error) {
		called = true
		return struct{}{}, nil
	}

	h := adaptHandler[struct{}, struct{}](codec, handler)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", http.NoBody)
	h.ServeHTTP(rec, req)

	if called {
		t.Fatalf("handler called after decode error")
	}
	if codec.encodeErrorCalls != 1 {
		t.Fatalf("expected EncodeError called once, got %d", codec.encodeErrorCalls)
	}
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestGroupRegistration_AppendsToRootHandlers(t *testing.T) {
	r := New()
	g := r.Group("/v1")

	RegisterHandler(g, GET(func(context.Context, struct{}) (struct{}, error) {
		return struct{}{}, nil
	}, "/ping"))

	if got := len(r.Handlers); got != 1 {
		t.Fatalf("expected router to have 1 handler, got %d", got)
	}
	if r.Handlers[0].Path != "/v1/ping" {
		t.Fatalf("expected full path %q, got %q", "/v1/ping", r.Handlers[0].Path)
	}
}
