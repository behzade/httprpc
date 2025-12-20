# httprpc

httprpc is a Go library for building typed HTTP RPC services with reflection-based TypeScript client generation.

## Installation

```bash
go get github.com/behzade/httprpc
```

## Quick Start

```go
package main

import (
    "context"
    "net/http"

    "github.com/behzade/httprpc"
)

type CreateUserRequest struct {
    Name  string `json:"name"`
    Email string `json:"email"`
}

type User struct {
    ID    int    `json:"id"`
    Name  string `json:"name"`
    Email string `json:"email"`
}

func main() {
    r := httprpc.New()

    httprpc.RegisterHandler(r.EndpointGroup, httprpc.POST(
        func(ctx context.Context, req CreateUserRequest) (User, error) {
            // Your business logic here
            return User{ID: 1, Name: req.Name, Email: req.Email}, nil
        },
        "/users",
    ))

http.ListenAndServe(":8080", r.HandlerMust())
}
```

## Core Concepts

### Handlers

Handlers are typed functions that take a context and a request type, returning a response type and an error:

```go
type Handler[Req any, Res any] func(ctx context.Context, request Req) (Res, error)
```

For endpoints that need typed path/header metadata, use `HandlerWithMeta`:

```go
type HandlerWithMeta[Req any, Meta any, Res any] func(ctx context.Context, request Req, meta Meta) (Res, error)
```

### Endpoints

Endpoints combine a handler with an HTTP method and path:

```go
endpoint := httprpc.POST(handler, "/path")
```

Supported methods: GET, POST, PUT, DELETE, PATCH, OPTIONS, HEAD.

For meta-aware handlers, use `GETWithMeta`/`POSTWithMeta` and `RegisterHandlerWithMeta`.

### Path params

You can register routes with path parameters using `:name` segments (snake_case):

```go
type GetUserMeta struct {
	ID int `path:"id"`
}

httprpc.RegisterHandlerWithMeta(router.EndpointGroup, httprpc.GETWithMeta(
	func(ctx context.Context, _ struct{}, meta GetUserMeta) (User, error) {
		return userService.Get(ctx, meta.ID)
	},
	"/users/:id",
))
```

Path parameters are decoded into the meta struct using the `path` tag (snake_case). They are not merged into the request body/query. You can also read them directly via `httprpc.PathParam(ctx, "id")` if you need access in untyped middleware.

Meta structs can also decode headers:

```go
type AuthMeta struct {
	Authorization string `header:"authorization"`
	RequestID     string `header:"x-request-id,omitempty"`
}

httprpc.RegisterHandlerWithMeta(router.EndpointGroup, httprpc.GETWithMeta(
	func(ctx context.Context, _ struct{}, meta AuthMeta) (User, error) {
		return userService.GetAuthorized(ctx, meta.Authorization)
	},
	"/me",
))
```

Header fields without `omitempty` are required; missing headers return `400 Bad Request`.

### Registration

Register endpoints on a router or endpoint group:

```go
httprpc.RegisterHandler(router.EndpointGroup, endpoint)
// or
group := router.Group("/api")
httprpc.RegisterHandler(group, endpoint)
```

For meta-aware endpoints, use `RegisterHandlerWithMeta`.

### Router

The router manages endpoints and provides the HTTP handler:

```go
r := httprpc.New()
// Register endpoints...
handler := r.HandlerMust()
```

### Server

For convenience, create a configured `http.Server`:

```go
server := r.Server(":8080")
server.ListenAndServe()
```

Or use `RunServer` for automatic graceful shutdown on SIGINT/SIGTERM:

```go
// Simple usage with defaults (graceful shutdown with 30s timeout)
if err := r.RunServer(":8080"); err != nil {
    log.Fatal(err)
}

// Custom shutdown timeout
r.RunServer(":8080", httprpc.WithGracefulShutdown(60*time.Second))

// Custom logger
r.RunServer(":8080", httprpc.WithLogger(myLogger))

// Combine options
r.RunServer(":8080",
    httprpc.WithGracefulShutdown(60*time.Second),
    httprpc.WithLogger(myLogger),
)
```

## Middleware

### Untyped Middleware

Apply HTTP-level middleware to routers or groups:

```go
r.Use(func(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Middleware logic
        next.ServeHTTP(w, r)
    })
})
```

Middleware priority controls execution order (higher priority runs earlier):

```go
r.Use(middleware, httprpc.Priority(10))
```

Built-in middlewares (in `github.com/behzade/httprpc/middleware`):

- `Recover(logger)` – panic recovery with 500 fallback.
- `Logging(logger)` – request/response logging (includes request ID when set).
- `RequestID(header)` – propagates/generates request IDs (default header: `X-Request-ID`).
- `RequestSizeLimit(maxBytes)` – wraps `http.MaxBytesReader`.
- `Timeout(d)` – adds a per-request context timeout.
- `CORS(cfg)` – simple configurable CORS handling.

### Typed Middleware

Apply per-endpoint typed middleware:

```go
httprpc.RegisterHandler(r, endpoint, httprpc.WithMiddleware[Req, Res](func(next httprpc.Handler[Req, Res]) httprpc.Handler[Req, Res] {
    return func(ctx context.Context, req Req) (Res, error) {
        // Typed middleware logic
        return next(ctx, req)
    }
}))
```

For meta-aware handlers, use `WithMetaMiddleware` and `HandlerWithMeta`:

```go
httprpc.RegisterHandlerWithMeta(r, endpoint, httprpc.WithMetaMiddleware[Req, Meta, Res](func(next httprpc.HandlerWithMeta[Req, Meta, Res]) httprpc.HandlerWithMeta[Req, Meta, Res] {
	return func(ctx context.Context, req Req, meta Meta) (Res, error) {
		// Typed middleware logic with meta
		return next(ctx, req, meta)
	}
}))
```

## Endpoint Groups

Organize endpoints with groups and prefixes:

```go
api := r.Group("/api")
v1 := api.Group("/v1")

httprpc.RegisterHandler(v1, httprpc.GET(handler, "/users"))
// Registers at /api/v1/users
```

Groups inherit middleware from parents.

## Codecs

Codecs handle request/response encoding/decoding. JSON is used by default:

```go
// DefaultCodec: JSON bodies, query param decoding for GET.
// Custom codecs implement DecodeBody/DecodeQuery/Encode/EncodeError.
httprpc.RegisterHandler(r, endpoint, httprpc.WithCodec[Req, Res](customCodec))
```

Implement the `Codec[Req, Res]` interface for custom codecs.

For meta-aware handlers:

```go
httprpc.RegisterHandlerWithMeta(r, endpoint, httprpc.WithCodecWithMeta[Req, Meta, Res](customCodec))
```

## Error Handling

Use `StatusError` to return HTTP status codes:

```go
return nil, httprpc.StatusError{Status: http.StatusBadRequest, Err: errors.New("invalid input")}
```

Decode failures automatically return 400 Bad Request.

## TypeScript Client Generation

Generate TypeScript clients from registered endpoints.
Path params come from route patterns, and header tags on meta structs become typed `headers` parameters in the generated client.

### Single File

```go
var buf bytes.Buffer
opts := httprpc.TSGenOptions{
    ClientName: "APIClient",
}
if err := r.GenTS(&buf, opts); err != nil {
    log.Fatal(err)
}
```

### Multi-File (by Path Segment)

```go
opts := httprpc.TSGenOptions{
    SkipPathSegments: 1, // Skip /api/v1/ prefix
}
if err := r.GenTSDir("client", opts); err != nil {
    log.Fatal(err)
}
```

This generates:
- `base.ts`: Base client class
- `<module>.ts`: Module-specific clients and types
- `index.ts`: Main export

### go:generate

Create a generator file:

```go
//go:generate go run ./gen
package main

import (
    "log"
    "github.com/behzade/httprpc"
)

func main() {
    r := httprpc.New()
    // Register your endpoints here
    if err := r.GenTSDir("../client", httprpc.TSGenOptions{}); err != nil {
        log.Fatal(err)
    }
}
```

Then run `go generate` in the directory containing the comment.

### Runtime Generation

Configure the router and invoke generation explicitly (e.g., behind a CLI flag):

```go
r.SetTSClientGenConfig(&httprpc.TSClientGenConfig{
    Dir: "client",
    Options: httprpc.TSGenOptions{
        ClientName: "API",
        SkipPathSegments: 1,
    },
})
if err := r.GenerateTSClient(); err != nil {
    log.Fatal(err)
}
```

## Requirements

- Go 1.25.4 or later
- JSON tags on struct fields must be snake_case (e.g., `json:"field_name"`)

## License

See LICENSE file.</content>
<parameter name="filePath">README.md
