package main

import (
	"context"
	"net/http"

	"github.com/behzade/httprpc"
)

func main() {
	router := httprpc.New()

	router.SetTSClientGenConfig(&httprpc.TSClientGenConfig{
		Dir: "./frontend/lib/api",
	})

	apiGroup := router.Group("/api")

	httprpc.RegisterHandler(
		apiGroup,
		httprpc.GET(
			httprpc.HandlerFunc[struct{}, struct{}](func(ctx context.Context, req struct{}) (struct{}, error) {
				return struct{}{}, nil
			}),
			"/ping",
		),
	)

	type Echo struct {
		Message string `json:"message"`
	}

	httprpc.RegisterHandler(
		apiGroup,
		httprpc.POST(
			httprpc.HandlerFunc[Echo, Echo](func(ctx context.Context, req Echo) (Echo, error) {
				return req, nil
			}),
			"/echo",
		),
	)

	err := http.ListenAndServe(":18080", router.Handler())
	if err != nil {
		panic(err)
	}
}
