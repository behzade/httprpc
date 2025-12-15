package httprpc

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

// Codec defines the interface for encoding and decoding HTTP requests and responses.
type Codec[Req any, Res any] interface {
	DecodeBody(*http.Request) (Req, error)
	DecodeQuery(*http.Request) (Req, error)
	Encode(http.ResponseWriter, Res) error
	EncodeError(http.ResponseWriter, error) error
}

// DefaultCodec implements Codec using JSON for bodies and structured query decoding.
type DefaultCodec[Req any, Res any] struct {
	Status int
}

// Consumes returns the content types this codec can decode.
func (c DefaultCodec[Req, Res]) Consumes() []string { return []string{"application/json"} }

// Produces returns the content types this codec can encode.
func (c DefaultCodec[Req, Res]) Produces() []string { return []string{"application/json"} }

// DecodeBody decodes the request body into the request type using JSON.
func (c DefaultCodec[Req, Res]) DecodeBody(r *http.Request) (Req, error) {
	var req Req
	if r.Body == nil {
		return req, nil
	}
	defer func() { _ = r.Body.Close() }()

	err := json.NewDecoder(r.Body).Decode(&req)
	if errors.Is(err, io.EOF) {
		return req, nil
	}
	if err != nil {
		return req, fmt.Errorf("decode request: %w", err)
	}
	return req, nil
}

// DecodeQuery decodes query parameters into the request type.
func (c DefaultCodec[Req, Res]) DecodeQuery(r *http.Request) (Req, error) {
	return decodeQueryParams[Req](r)
}

// Encode encodes the response into the HTTP response writer.
func (c DefaultCodec[Req, Res]) Encode(w http.ResponseWriter, res Res) error {
	w.Header().Set("Content-Type", "application/json")
	if c.Status != 0 {
		w.WriteHeader(c.Status)
	}
	if err := json.NewEncoder(w).Encode(res); err != nil {
		return fmt.Errorf("encode response: %w", err)
	}
	return nil
}

// EncodeError encodes an error into the HTTP response writer.
func (c DefaultCodec[Req, Res]) EncodeError(w http.ResponseWriter, err error) error {
	w.Header().Set("Content-Type", "application/json")
	status := http.StatusInternalServerError
	var se StatusError
	if errors.As(err, &se) && se.Status != 0 {
		status = se.Status
	}
	w.WriteHeader(status)
	if encErr := json.NewEncoder(w).Encode(map[string]string{"error": err.Error()}); encErr != nil {
		return fmt.Errorf("encode error response: %w", encErr)
	}
	return nil
}
