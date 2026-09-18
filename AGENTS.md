# ConfigGen — AGENTS.md

## Agent guidance

- This repo is a Go HTTP service with Clean Architecture. Work is centered in `cmd/configgen/`, `internal/usecase/`, `internal/infrastructure/`, and `internal/presentation/`.
- Prefer the existing `README.md` for usage, API examples, and development commands. Do not duplicate large docs from the README.
- Runtime configuration is loaded from a root `.env` file if present, plus environment variables. `.env` is gitignored; `.env.example` is the template.
- Supported runtime env vars: `PORT`, `ADDR`, `CONFIGGEN_DB`, `SENTRY_DSN`.
- Use `go fmt ./...`, `go vet ./...`, and `go test ./...` for code changes.

## Quick start

```powershell
go build -o configgen.exe ./cmd/configgen
.\configgen.exe -addr=:8080
```

Verify: `curl http://localhost:8080/health` → `{"status":"ok"}`

## Commands

| Action | Command |
|--------|---------|
| Build | `go build -o configgen.exe ./cmd/configgen` |
| Test all | `go test ./...` |
| Test one package | `go test ./internal/infrastructure/generators/` |
| Static analysis | `go vet ./...` |
| Coverage | `go test -cover ./...` |
| Frontend dev | `cd web && npm run dev` (Vite :5173, proxies `/api` to :8080) |
| Frontend build | `cd web && npm install && npm run build` (output in `web/dist`, embedded via go:embed) |

No formatter or linter config beyond `go fmt` / `go vet`.

## Architecture

Clean Architecture, four layers:

```
cmd/configgen/main.go          — DI assembly: store → generator → usecase → server
config/                        — runtime config: .env loading, listen-addr resolution
internal/domain/               — entities (ConfigRequest, ConfigResult, ConfigRecord) + Store interface
internal/usecase/              — business logic: applyDefaults, GenerateAndSave, GetRecord, ListRecords
internal/infrastructure/       — concrete implementations (generators, SQLite store)
internal/presentation/         — Gin HTTP handlers + request parsing + SPA static/fallback serving
```

Dependency direction: `presentation → usecase → domain ← infrastructure`

## Templates are compiled into the binary

All templates live under `internal/infrastructure/generators/templates/` and are loaded via `//go:embed`. The old root `templates/` directory was deleted.

Same for frontend assets — the Vue SPA build output in `web/dist/` is embedded via `web/embed.go` (`//go:embed all:dist`). `web/dist` is committed so Go-only builds work; rebuild it with `cd web && npm install && npm run build`. Frontend source lives in `web/src` (Vue 3 + Vite + TypeScript + Pinia + Naive UI + Vue Router, primary color #7DAEE1). Dev mode: `cd web && npm run dev` serves Vite on :5173 and proxies `/api` to the Go backend on :8080. `web/node_modules` is gitignored.

There is no runtime template hot-reload.

## Key model quirks

- `Port` has `validate:"min=0,max=65535"` (not `required`). Default is applied by `usecase.applyDefaults()` via `Prepare()`. Validation runs AFTER `Prepare()`.
- `ConfigRequest` fields are grouped by type (common / k8s / dockerfile) in struct comments at `internal/domain/types.go`.
- `binding:` tags were removed from `ConfigRequest` — handlers validate manually with `validator.Struct()`, Gin's `ShouldBindJSON` is never used. Do not re-add them.
- `GenerateAndSave` does NOT apply defaults. Callers must call `Prepare()` first (presentation does, before validation).
- `K8sEnable` was removed — do not add it back.

## Generators

Four supported types: `compose`, `k8s`, `dockerfile`, `kustomize`.

Adding a new generator type:
1. Implement `Generator` interface from `internal/infrastructure/generators/interface.go`
2. Call `Register("type_name", &MyGenerator{})` — preferably in an `init()` function
3. Add templates under `internal/infrastructure/generators/templates/<type>/`

The K8s generator supports **20 resource kinds** (Deployment, Service, ConfigMap, Secret, …). Each kind has its own `.yaml.tmpl` file. The template cache uses sync.RWMutex with double-check locking — two goroutines do not parse the same template twice.

## K8s Resource Relationships

When generating K8s or Kustomize configs, `ExtractK8sRelationships()` derives `ResourceNode` and `ResourceEdge` lists from the requested resource types. These are stored inline in `ConfigResult.Resources` and `ConfigResult.Edges`.

API endpoints for relationships:
- `GET /api/v1/configs/:id/resources` — list of resource nodes
- `GET /api/v1/configs/:id/edges` — list of edges
- `GET /api/v1/configs/:id/graph` — combined `{resources, edges}` for visualization

Relationship rules are defined in `k8s.go:ExtractK8sRelationships()` covering Service→Deployment, Ingress→Service, Deployment→ConfigMap, RBAC bindings, etc.

## Env var ordering

Compose and Dockerfile generators sort env keys alphabetically before rendering. Output is deterministic, not map-iteration-order dependent.

## Not implemented (do not claim support)

The original design doc (since deleted) listed these as planned: Nginx config, Docker Daemon config, Containerd config, ZIP packaging, template hot-reload, an external plugin system, and i18n. **None are implemented.** Only compose / k8s / dockerfile / kustomize work. The "plugin architecture" in commit history is the in-process registry (`Register()` + `init()`), not external plugins. The current design doc lives at `docs/DESIGN.md` (system + frontend design).

## Test coverage

Four packages have tests:
- `internal/infrastructure/generators/` — 89.5%
- `internal/infrastructure/storage/` — 74.5%
- `internal/usecase/` — new (mock-based)
- `internal/presentation/` — new (interface + httptest)

## Previous bugs fixed (do not reintroduce)

1. `applyDefaults` ran AFTER validation → moved `Prepare()` before `validate.Struct()`
2. `getConfig` returned 404 for all errors → now 404 only for `sql.ErrNoRows`, 500 otherwise
3. DSN assumed no `?` → now checks `strings.Contains(dsn, "?")`
4. Template cache TOCTOU → double-check inside write lock
5. Env map iteration non-deterministic → sorted keys
6. No graceful shutdown → signal handler with 5s timeout
7. `applyDefaults` `default` case incorrectly set `needsPort=true` for all resource types → removed the default case, only Service/Ingress set `needsPort`
8. `GenerateAndSave` re-applied defaults after the handler's `Prepare()` → removed the duplicate call; `Prepare()` is now the single place defaults are applied

## SQLite

- WAL journal mode enabled
- DSN separator handled: if DSN already has `?`, next param uses `&`
- Timestamps stored as RFC3339Nano text; `scanRecord` returns error on parse failure (no longer swallowed)
