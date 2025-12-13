package httprpc

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

type Codec[Req any, Res any] interface {
	Decode(*http.Request) (Req, error)
	Encode(http.ResponseWriter, Res) error
	EncodeError(http.ResponseWriter, error) error
}

type JSONCodec[Req any, Res any] struct {
	Status int
}

func (c JSONCodec[Req, Res]) Consumes() []string { return []string{"application/json"} }
func (c JSONCodec[Req, Res]) Produces() []string { return []string{"application/json"} }

func (c JSONCodec[Req, Res]) Decode(r *http.Request) (Req, error) {
	var req Req
	if r.Body == nil {
		return req, nil
	}
	defer r.Body.Close()

	err := json.NewDecoder(r.Body).Decode(&req)
	if errors.Is(err, io.EOF) {
		return req, nil
	}
	return req, err
}

func (c JSONCodec[Req, Res]) Encode(w http.ResponseWriter, res Res) error {
	w.Header().Set("Content-Type", "application/json")
	if c.Status != 0 {
		w.WriteHeader(c.Status)
	}
	return json.NewEncoder(w).Encode(res)
}

func (c JSONCodec[Req, Res]) EncodeError(w http.ResponseWriter, err error) error {
	w.Header().Set("Content-Type", "application/json")
	status := http.StatusInternalServerError
	var se StatusError
	if errors.As(err, &se) && se.Status != 0 {
		status = se.Status
	}
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}
