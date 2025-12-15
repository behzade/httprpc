package httprpc

import (
	"context"
	"log/slog"
	"net/http"
)

// Handler is a simple function type for handling requests.
type Handler[Req any, Res any] func(ctx context.Context, request Req) (Res, error)

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
