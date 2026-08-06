# Tennon Web UI

Vite + React + TypeScript SPA embedded into the `tennon` binary.

## Development

```bash
cd web/ui
bun install
bun run dev     # dev server on http://localhost:5173
```

## Build

```bash
cd web/ui
bun run build   # outputs to dist/
```

The built assets are embedded via `go:embed` in `web/web.go` and served at `/` on the API port. The Go binary includes the built UI, so you don't need Node at runtime.

## Rebuild workflow

1. Make changes in `src/`
2. `bun run build`
3. `go build ./...` to pick up the new embedded assets
4. Commit both source and `dist/`

## Structure

- `src/` — TypeScript source
- `src/pages/` — React pages (Connectors, AddConnector, Secrets)
- `src/types.ts` — API types
- `dist/` — Built assets (committed for `go build` without a Node toolchain)
