// Package main provides an example usage of the httprpc library.
package main

import (
	"context"
	"embed"
	"flag"
	"io/fs"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/behzade/httprpc"
)

const (
	exampleReadTimeout  = 30 * time.Second
	exampleWriteTimeout = 30 * time.Second
)

//go:embed frontend/dist
var embeddedFrontend embed.FS

func main() {
	shouldGen := flag.Bool("gen", false, "generate TypeScript client and exit")
	flag.Parse()

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

	if *shouldGen {
		if err := router.GenerateTSClient(); err != nil {
			panic(err)
		}
		return
	}

	apiHandler := router.Handler()

	staticFS, err := fs.Sub(embeddedFrontend, "frontend/dist")
	if err != nil {
		panic(err)
	}
	frontendHandler := spaHandler(staticFS)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isAPIRequest(r.URL.Path) {
			apiHandler.ServeHTTP(w, r)
			return
		}
		frontendHandler.ServeHTTP(w, r)
	})

	server := &http.Server{
		Addr:         ":18080",
		Handler:      handler,
		ReadTimeout:  exampleReadTimeout,
		WriteTimeout: exampleWriteTimeout,
	}
	err = server.ListenAndServe()
	if err != nil {
		panic(err)
	}
}

func isAPIRequest(path string) bool {
	return path == "/api" || strings.HasPrefix(path, "/api/")
}

func spaHandler(staticFS fs.FS) http.Handler {
	fileServer := http.FileServer(http.FS(staticFS))

	serveIndex := func(w http.ResponseWriter, r *http.Request) {
		req := r.Clone(r.Context())
		req.URL.Path = "/"
		http.ServeFileFS(w, req, staticFS, "index.html")
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		target := path.Clean(strings.TrimPrefix(r.URL.Path, "/"))
		switch {
		case target == ".", target == "index.html", strings.HasPrefix(target, "../"):
			serveIndex(w, r)
			return
		default:
			info, err := fs.Stat(staticFS, target)
			if err == nil {
				if info.IsDir() {
					fileServer.ServeHTTP(w, r)
					return
				}
				req := r.Clone(r.Context())
				req.URL.Path = "/" + target
				fileServer.ServeHTTP(w, req)
				return
			}
			serveIndex(w, r)
			return
		}
	})
}
