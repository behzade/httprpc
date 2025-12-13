package httprpc

import (
	"context"
	"log/slog"
	"net/http"
)

type Handler[Req any, Res any] interface {
	Handle(ctx context.Context, request Req) (Res, error)
}

type HandlerFunc[Req any, Res any] func(ctx context.Context, request Req) (Res, error)

func (f HandlerFunc[Req, Res]) Handle(ctx context.Context, request Req) (Res, error) {
	return f(ctx, request)
}

func adaptHandler[Req any, Res any](codec Codec[Req, Res], handler Handler[Req, Res]) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		req, err := codec.Decode(r)
		if err != nil {
			encodeErr := codec.EncodeError(w, err)
			if encodeErr != nil {
				slog.Error("failed to encode error response", "error", encodeErr)
			}
		}

		res, err := handler.Handle(r.Context(), req)
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
