// Package main provides an example usage of the httprpc library.
package main

import (
	"context"
	"net/http"
	"time"

	"github.com/behzade/httprpc"
)

const (
	exampleReadTimeout  = 30 * time.Second
	exampleWriteTimeout = 30 * time.Second
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
			httprpc.HandlerFunc[struct{}, struct{}](func(_ context.Context, _ struct{}) (struct{}, error) {
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
			httprpc.HandlerFunc[Echo, Echo](func(_ context.Context, req Echo) (Echo, error) {
				return req, nil
			}),
			"/echo",
		),
	)

	server := &http.Server{
		Addr:         ":18080",
		Handler:      router.Handler(),
		ReadTimeout:  exampleReadTimeout,
		WriteTimeout: exampleWriteTimeout,
	}
	err := server.ListenAndServe()
	if err != nil {
		panic(err)
	}
}
