package httprpc

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDefaultCodecDecode_EmptyBodyDoesNotError(t *testing.T) {
	type Req struct {
		A int `json:"a"`
	}

	codec := DefaultCodec[Req, struct{}]{}
	req := httptest.NewRequest(http.MethodPost, "/", http.NoBody)

	got, err := codec.DecodeBody(req)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if got.A != 0 {
		t.Fatalf("expected zero value request, got %+v", got)
	}
}

func TestDefaultCodecDecode_ReqStructDoesNotRequireBody(t *testing.T) {
	codec := DefaultCodec[struct{}, struct{}]{}
	req := httptest.NewRequest(http.MethodGet, "/", strings.NewReader(`{"ignored":true}`))

	_, err := codec.DecodeBody(req)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

func TestDefaultCodecEncode_ResStructWritesJSON(t *testing.T) {
	codec := DefaultCodec[struct{}, struct{}]{}
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

func TestDefaultCodecEncodeError_UsesStatusError(t *testing.T) {
	codec := DefaultCodec[struct{}, struct{}]{}
	rec := httptest.NewRecorder()

	_ = codec.EncodeError(rec, StatusError{Status: http.StatusBadRequest, Err: errors.New("bad")})

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestDefaultCodecDecode_QueryParamsForGET(t *testing.T) {
	type Req struct {
		Name   string   `json:"name"`
		Age    int      `json:"age"`
		Active bool     `json:"active"`
		Tags   []string `json:"tags"`
	}

	codec := DefaultCodec[Req, struct{}]{}
	req := httptest.NewRequest(http.MethodGet, "/?name=alice&age=30&active=true&tags=a&tags=b", http.NoBody)

	got, err := codec.DecodeQuery(req)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if got.Name != "alice" || got.Age != 30 || !got.Active || len(got.Tags) != 2 || got.Tags[0] != "a" || got.Tags[1] != "b" {
		t.Fatalf("unexpected decoded req: %+v", got)
	}
}

func TestDefaultCodecDecode_QueryTagPreferred(t *testing.T) {
	type Req struct {
		Query string `query:"q" json:"query"`
		Page  int    `json:"page"`
	}

	codec := DefaultCodec[Req, struct{}]{}
	req := httptest.NewRequest(http.MethodGet, "/?q=search&page=2", http.NoBody)

	got, err := codec.DecodeQuery(req)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if got.Query != "search" || got.Page != 2 {
		t.Fatalf("unexpected decoded req: %+v", got)
	}
}

func TestDefaultCodecDecode_FieldNameFallback(t *testing.T) {
	type Req struct {
		PlainValue string
	}

	codec := DefaultCodec[Req, struct{}]{}
	req := httptest.NewRequest(http.MethodGet, "/?plain_value=test", http.NoBody)

	got, err := codec.DecodeQuery(req)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if got.PlainValue != "test" {
		t.Fatalf("unexpected decoded req: %+v", got)
	}
}
