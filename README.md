# httprpc

Typed HTTP RPC routing with reflection-based TypeScript client generation.

## TypeScript generation

This package can only generate a TS client from endpoints that have been registered on a `Router`. If you're publishing `httprpc` as a library, the routes live in the consuming module, so `httprpc` cannot generate clients by itself without executing the consumer's router construction code.

### Option A: `go:generate` (recommended)

Add a small generator `main` in your service that constructs your router (calls all `RegisterHandler(...)`) and then calls `router.GenTSDir(...)`.

Example:

- `internal/api/gen-ts/main.go` (or similar): builds router and calls `GenTSDir`.
- In a Go file next to your router wiring:

```go
//go:generate go run ./internal/api/gen-ts
```

### Option B: regenerate on serve (opt-in)

You can opt in to TS regeneration at runtime by setting a `TSClientGenConfig` on the router. When configured, `Router.Handler()` attempts to generate the TS client before returning the handler.

```go
r := httprpc.NewRouter()
// ... RegisterHandler calls ...
r.SetTSClientGenConfig(&httprpc.TSClientGenConfig{
  Dir: "web/src/api",
  Options: httprpc.TSGenOptions{
    ClientName: "API",
    SkipPathSegments: 1,
  },
  OnError: func(err error) { /* log if you want */ },
})
```

To avoid regenerating unnecessarily, `httprpc` computes a checksum from registered endpoint metadata + generation options and embeds it as `__httprpc_checksum` in the generated client.
It also writes the checksum to `./.httprpc-checksum` in the output directory to make skip-checks cheap and robust.
