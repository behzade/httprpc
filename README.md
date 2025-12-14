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
    r := httprpc.NewRouter()

    httprpc.RegisterHandler(r, httprpc.POST(
        httprpc.HandlerFunc[CreateUserRequest, User](func(ctx context.Context, req CreateUserRequest) (User, error) {
            // Your business logic here
            return User{ID: 1, Name: req.Name, Email: req.Email}, nil
        }),
        "/users",
    ))

    http.ListenAndServe(":8080", r.Handler())
}
```

## Core Concepts

### Handlers

Handlers are typed functions that take a context and a request type, returning a response type and an error:

```go
type Handler[Req any, Res any] interface {
    Handle(ctx context.Context, request Req) (Res, error)
}

type HandlerFunc[Req any, Res any] func(ctx context.Context, request Req) (Res, error)
```

### Endpoints

Endpoints combine a handler with an HTTP method and path:

```go
endpoint := httprpc.POST(handler, "/path")
```

Supported methods: GET, POST, PUT, DELETE, PATCH, OPTIONS, HEAD.

### Registration

Register endpoints on a router or endpoint group:

```go
httprpc.RegisterHandler(router, endpoint)
```

### Router

The router manages endpoints and provides the HTTP handler:

```go
r := httprpc.NewRouter()
// Register endpoints...
handler := r.Handler()
```

### Server

For convenience, create a configured `http.Server`:

```go
server := r.Server(":8080")
server.ListenAndServe()
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

### Typed Middleware

Apply per-endpoint typed middleware:

```go
httprpc.RegisterHandler(r, endpoint, httprpc.WithMiddleware[Req, Res](func(next httprpc.Handler[Req, Res]) httprpc.Handler[Req, Res] {
    return httprpc.HandlerFunc[Req, Res](func(ctx context.Context, req Req) (Res, error) {
        // Typed middleware logic
        return next.Handle(ctx, req)
    })
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
httprpc.RegisterHandler(r, endpoint, httprpc.WithCodec[Req, Res](customCodec))
```

Implement the `Codec[Req, Res]` interface for custom codecs.

## Error Handling

Use `StatusError` to return HTTP status codes:

```go
return nil, httprpc.StatusError{Status: http.StatusBadRequest, Err: errors.New("invalid input")}
```

Decode failures automatically return 400 Bad Request.

## TypeScript Client Generation

Generate TypeScript clients from registered endpoints.

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
    r := httprpc.NewRouter()
    // Register your endpoints here
    if err := r.GenTSDir("../client", httprpc.TSGenOptions{}); err != nil {
        log.Fatal(err)
    }
}
```

Then run `go generate` in the directory containing the comment.

### Runtime Generation

Configure the router for runtime TS generation:

```go
r.SetTSClientGenConfig(&httprpc.TSClientGenConfig{
    Dir: "client",
    Options: httprpc.TSGenOptions{
        ClientName: "API",
        SkipPathSegments: 1,
    },
    OnError: func(err error) { log.Println(err) },
})
```

The client is regenerated when the router handler is first called, if the checksum changes.

## Requirements

- Go 1.25.4 or later
- JSON tags on struct fields must be snake_case (e.g., `json:"field_name"`)

## License

See LICENSE file.</content>
<parameter name="filePath">README.md