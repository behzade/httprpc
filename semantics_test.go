package httprpc

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestJSONCodecDecode_EmptyBodyDoesNotError(t *testing.T) {
	type Req struct {
		A int `json:"a"`
	}

	codec := JSONCodec[Req, struct{}]{}
	req := httptest.NewRequest(http.MethodPost, "/", http.NoBody)

	got, err := codec.Decode(req)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if got.A != 0 {
		t.Fatalf("expected zero value request, got %+v", got)
	}
}

func TestJSONCodecDecode_ReqStructDoesNotRequireBody(t *testing.T) {
	codec := JSONCodec[struct{}, struct{}]{}
	req := httptest.NewRequest(http.MethodGet, "/", strings.NewReader(`{"ignored":true}`))

	_, err := codec.Decode(req)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

func TestJSONCodecEncode_ResStructWritesJSON(t *testing.T) {
	codec := JSONCodec[struct{}, struct{}]{}
	rec := httptest.NewRecorder()

	if err := codec.Encode(rec, struct{}{}); err != nil {
		t.Fatalf("encode error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if body := rec.Body.String(); body == "" {
		t.Fatalf("expected non-empty body")
	}
}

func TestJSONCodecEncodeError_UsesStatusError(t *testing.T) {
	codec := JSONCodec[struct{}, struct{}]{}
	rec := httptest.NewRecorder()

	_ = codec.EncodeError(rec, StatusError{Status: http.StatusBadRequest, Err: errors.New("bad")})

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}
