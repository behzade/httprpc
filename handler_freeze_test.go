package httprpc

import (
	"context"
	"testing"
)

func TestRegisterAfterHandlerPanics(t *testing.T) {
	r := New()
	_, err := r.Handler()
	if err != nil {
		t.Fatalf("handler build error: %v", err)
	}

	defer func() {
		if rec := recover(); rec == nil {
			t.Fatalf("expected panic when registering after handler build")
		}
	}()

	RegisterHandler(r.EndpointGroup, GET(func(context.Context, struct{}) (struct{}, error) {
		return struct{}{}, nil
	}, "/late"))
}
