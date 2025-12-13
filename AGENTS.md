# httprpc agent notes

## Conventions
- Filenames use hyphens (`-`) as separators; only `_test.go` uses underscores.
- Keep code minimal and composable; prefer `http.Handler` as the primary serving output, with an optional `http.Server` convenience wrapper.

## Routing / registration model
- Typed handlers: `Handler[Req, Res]` is always `ctx, req -> (res, err)` and carries no extra metadata.
- Endpoint metadata is captured at registration time in `RegisterHandler` (`endpoint.go`), stored on the root router/group:
  - `Method`, `Path`
  - `Req`/`Res` Go types via reflection (used for TS generation)
  - `Consumes`/`Produces` content-type hints (optional via codec)
- Endpoint groups:
  - `EndpointGroup.Group(prefix)` creates subgroups; registration always appends to the root group.
  - Group/Router middlewares are untyped (`func(http.Handler) http.Handler`) and apply at HTTP layer.
  - Per-endpoint typed middlewares exist via `HandlerMiddleware[Req,Res]` and are applied during `RegisterHandler`.

## HTTP semantics
- Decode failures are treated as `400 Bad Request` via `StatusError` wrapping.
- JSON decoding treats `io.EOF` as “empty body” (zero-value request) and does not error.

## TypeScript generation
- TS generation is reflection-based and exposed as methods on `Router`:
  - `GenTSDir(dir, opts)` generates multiple files split by the first path segment (with optional `SkipPathSegments` for `/v1/...` style routes).
- TS templates are stored in `templates/ts/*.tmpl` and embedded via `go:embed`.
- There is intentionally no central `types.ts`; types are emitted into the same module file as their routes (by path segment), even if this duplicates shared types across modules.

## Tests / inspection
- Golden-style TS snapshots can be generated for manual inspection with:
  - `UPDATE_GOLDEN=1 go test ./... -run TestTSGenGolden`
- Local Go caches are expected to be set via env vars (`GOCACHE`, `GOMODCACHE`) when running in restricted environments; repo includes `.gocache/` and `.gomodcache/` patterns in `.gitignore`.

