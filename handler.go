package httprpc

import (
	"context"
	"log/slog"
	"net/http"
)

type pathParamsKey struct{}

// Handler is a simple function type for handling requests.
type Handler[Req any, Res any] func(ctx context.Context, request Req) (Res, error)

// HandlerWithMeta is a handler that receives request metadata decoded from path/header tags.
type HandlerWithMeta[Req, Meta, Res any] func(ctx context.Context, request Req, meta Meta) (Res, error)

// PathParam returns a path parameter by name.
func PathParam(ctx context.Context, name string) (string, bool) {
	values, _ := ctx.Value(pathParamsKey{}).(map[string]string)
	if values == nil {
		return "", false
	}
	val, ok := values[name]
	return val, ok
}

// PathParams returns a copy of all path params.
func PathParams(ctx context.Context) map[string]string {
	values, _ := ctx.Value(pathParamsKey{}).(map[string]string)
	if len(values) == 0 {
		return nil
	}
	out := make(map[string]string, len(values))
	for k, v := range values {
		out[k] = v
	}
	return out
}

func withPathParams(r *http.Request, params map[string]string) *http.Request {
	if len(params) == 0 {
		return r
	}
	return r.WithContext(context.WithValue(r.Context(), pathParamsKey{}, params))
}

func adaptHandler[Req, Res any](codec Codec[Req, Res], handler Handler[Req, Res]) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var (
			req Req
			err error
		)
		if r.Method == http.MethodGet {
			req, err = codec.DecodeQuery(r)
		} else {
			req, err = codec.DecodeBody(r)
		}
		if err != nil {
			if encodeErr := codec.EncodeError(w, StatusError{Status: http.StatusBadRequest, Err: err}); encodeErr != nil {
				slog.Error("failed to encode error response", "error", encodeErr)
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
			return
		}

		res, err := handler(r.Context(), req)
		if err != nil {
			if encodeErr := codec.EncodeError(w, err); encodeErr != nil {
				slog.Error("failed to encode error response", "error", encodeErr)
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
			return
		}

		err = codec.Encode(w, res)
		if err != nil {
			slog.Error("failed to encode response", "error", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
	})
}

func adaptHandlerWithMeta[Req, Meta, Res any](codec Codec[Req, Res], handler HandlerWithMeta[Req, Meta, Res]) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var (
			req Req
			err error
		)
		if r.Method == http.MethodGet {
			req, err = codec.DecodeQuery(r)
		} else {
			req, err = codec.DecodeBody(r)
		}
		var meta Meta
		if err == nil {
			meta, err = decodeRequestMeta[Meta](r)
		}
		if err != nil {
			if encodeErr := codec.EncodeError(w, StatusError{Status: http.StatusBadRequest, Err: err}); encodeErr != nil {
				slog.Error("failed to encode error response", "error", encodeErr)
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
			return
		}

		res, err := handler(r.Context(), req, meta)
		if err != nil {
			if encodeErr := codec.EncodeError(w, err); encodeErr != nil {
				slog.Error("failed to encode error response", "error", encodeErr)
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
			return
		}

		err = codec.Encode(w, res)
		if err != nil {
			slog.Error("failed to encode response", "error", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
	})
}
