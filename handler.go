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
		req, err := codec.Decode(r)
		if err != nil {
			encodeErr := codec.EncodeError(w, StatusError{Status: http.StatusBadRequest, Err: err})
			if encodeErr != nil {
				slog.Error("failed to encode error response", "error", encodeErr)
			}
			return
		}

		res, err := handler(r.Context(), req)
		if err != nil {
			encodeErr := codec.EncodeError(w, err)
			if encodeErr != nil {
				slog.Error("failed to encode error response", "error", encodeErr)
			}
			return
		}

		err = codec.Encode(w, res)
		if err != nil {
			slog.Error("failed to encode response", "error", err)
		}
	})
}
