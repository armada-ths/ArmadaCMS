<!--
Guidance for AI coding agents working on ArmadaCMS.
-->

# ArmadaCMS — Copilot instructions

See [README.md](../README.md) for full getting-started guide, project structure, and ops notes.

## Architecture

Go REST API (Gorilla Mux, GORM, Postgres) + React-Admin SPA in one repo. The Go server exposes:

- `/api/v1/*` — REST endpoints (public site `armada-ths/armada.nu` + admin frontend).
- `/admin/` — serves the built React-Admin SPA from `frontend/dist`.
- `/health` — healthcheck.

Deployed to **Google Cloud Run** (containerised).

- **Database**: Supabase PostgreSQL for both production and staging (staging uses a Supabase branch).
- **File storage**: S3-compatible storage API — local MinIO in Docker dev, Supabase Storage's S3 endpoint in staging/production.

## Developer workflows

**Docker dev (recommended):**

```bash
docker compose -f docker-compose.dev.yml up --build  # first run
docker compose -f docker-compose.dev.yml up           # subsequent
```

This runs the Go API, React-Admin frontend, Postgres, and MinIO together with hot reload.

Default credentials: host `localhost`, db `armadacms`, user/password `postgres`. Copy `.env.example` → `.env`.

`pnpm run build` outputs to `frontend/dist`, which the Go server serves at `/admin/`.

**Tests**: Go unit tests are available (notably in `auth/` and `utils/`). Run `go test -race -count=1 ./...` locally for backend changes. Frontend unit tests live in `frontend/src/` alongside the source files and use **vitest** (`pnpm run test` in `frontend/`). There is no end-to-end/integration test suite yet, so still verify relevant API behavior manually (for example `curl http://localhost:8080/health` and affected `/api/v1` endpoints).

**CI checks**: `.github/workflows/go-checks.yml` runs `go vet`, `golangci-lint`, and `go test -race -count=1 ./...` when Go files change (push to `main`/`staging` and pull requests).

**Swagger UI**: browsable API docs are served at `http://localhost:8080/swagger/index.html` while the server is running. After adding or changing routes, regenerate the spec with:

```bash
swag init --generalInfo main.go --output docs --parseInternal
```

Commit the generated `docs/` files alongside your code. Install the CLI once with `go install github.com/swaggo/swag/cmd/swag@latest`.

**Local data**: `scripts/import-remote-db.ps1` clones a remote Postgres DB into the local container.

**Terraform / HCP Terraform:** active roots are `gcp/prod`, `gcp/staging`, and `supabase/prod`. Avoid running `terraform plan` locally — the CLI-driven remote plan upload is slow. Prefer queueing plans from HCP Terraform when possible, and use local Terraform mainly for `validate`, `import`, or other targeted state operations. See [`infra/terraform/README.md`](../infra/terraform/README.md) and the per-root READMEs for workspace details.

## Backend patterns

- **Routing** (`main.go`): `publicAPI` (no auth) and `protectedAPI` (Bearer JWT) subrouters under `/api/v1`. Read-only list/get routes are typically public; write routes are protected.
- **Controllers** (`Controllers/`): read from `db.DB` (GORM global), JSON-encode responses. Use shared helpers from `Controllers/response_helpers.go` (`writeJSONResponse`, `writeCreatedJSONResponse`, `writeDeleteResponse`).
- **List endpoints**: use `utils.ParseListParams` (parses react-admin `sort`/`range`/`filter` query params) and set `Content-Range` header for react-admin pagination.
- **Models** (`models/`): GORM structs with camelCase JSON tags. Many-to-many via GORM `many2many` tag. Not all files in `models/` are DB models — `person.go` and `token.go` are response shapes.
- **Auto-migration**: every DB model must be registered in `db.DB.AutoMigrate(...)` in `main.go`.
- **Audit system** (critical): all write operations **must** use the generic helpers in `Controllers/audit_write_helpers.go` — `createWithAudit[T]`, `updateWithAudit[T]`, `writeDeleteResponseWithAudit[T]`. These wrap the mutation + audit log insert in one transaction atomically. Do **not** call `db.DB.Create/Save/Delete` directly from controllers. Old logs are automatically pruned on each audit insert (rate-limited to once per hour) based on `AUDIT_LOG_RETENTION_DAYS`. All three helpers accept a variadic `revalidateTags ...string` trailing argument — on success they fire `go utils.RevalidateTag(tag)` for each tag to purge the public site's ISR cache (see _Cache revalidation_ below).
- **File uploads**: controllers accepting files use `multipart/form-data`; files go through the S3-compatible upload helper in `utils/aws_s3.go` (validates MIME, generates timestamped key). In staging/production this is configured against Supabase Storage's S3 endpoint.
- **Auth** (`auth/middleware.go`): validates HS256 JWT (`jwtsecret_laganda` secret), injects `user_id`, `role`, `permissions` into request context. Per-route permission check via `auth.RequirePermission("resource.action", handler)`. Permissions follow `"resource.action"` format; `"*"` grants full access. Use `auth.GetUserIDFromContext` etc. to read from context in controllers.
- **Session tokens**: access tokens expire in 15 minutes; the admin frontend holds a 7-day rotating refresh token in `localStorage`. `POST /api/v1/login` returns both. `GET /api/v1/refreshAccessToken` (public route, `X-RefreshAuthorization: Bearer <token>` header) rotates the refresh token and issues a new access token. Ensure `jwtsecret_laganda` differs between staging and production.
- **Initial admin seeding**: on startup `SeedInitialAdminUser` runs once — it creates a user from `INITIAL_ADMIN_USERNAME` / `INITIAL_ADMIN_PASSWORD` env vars only if no users exist yet. Remove or leave empty once real accounts are created.
- **Eventro integration**: Eventro controllers sync external fair dates, exhibitors, events, members, and recruitments. Uses `EVENTRO_API` and `EVENTRO_ORG` env vars. Fair-scoped syncs require the user to select an active fair instance in the dashboard's `EventroSyncCard`. Fair-date sync atomically replaces all existing fair-date rows from the selected fair's timeline.
- **Feature flags**: `FeatureFlagController` seeds default flags on startup (`models/feature_flag.go`). Exhibitor signup open/closed state is computed on the frontend (`armada.nu`) based on IR/FR date windows from the dates API — it is **not** controlled by a feature flag.
- **Blogpost** (`Controllers/BlogpostController.go`): full CRUD with S3-compatible image upload (`multipart/form-data`) and a dedicated `POST /api/v1/blogimages` endpoint for inline markdown images. Revalidation tag: `"blog-posts"`.
- **Cache revalidation** (`utils/revalidate.go`): `RevalidateTag(tag)` POSTs `{ tag, secret }` to the public site's `/api/revalidate` endpoint (fire-and-forget, 5 s timeout). Requires `REVALIDATION_URL` and `REVALIDATION_SECRET`. On staging/preview, `VERCEL_AUTOMATION_BYPASS_SECRET` is sent as `x-vercel-protection-bypass` header. Silently skipped if env vars are unset. Tag names must match between Go controllers and the Next.js data hooks (see `armada.nu/.github/copilot-instructions.md` for the full tag inventory).
- **Update normalization** (`Controllers/update_normalization_helpers.go`): `NormalizeOptionalStringPointers()` trims whitespace / nils empty strings; `BuildNormalizedSnakeCaseUpdateMap()` converts camelCase form fields to snake_case for GORM partial updates.

## Admin frontend patterns

- **React-Admin v5** with `ra-data-simple-rest` data provider, customised in `frontend/src/dataProvider.ts`.
- **API endpoint**: `frontend/src/context/globalApi.ts` — `localhost:8080/api/v1` in dev, `window.location.origin/api/v1` in prod.
- **Auth**: JWT in `localStorage` (`accessToken`, `refreshToken`). Provider: `frontend/src/context/authProvider.ts`.
- **Resource components**: `frontend/src/components/{Resource}/` — each has `List`, `Create`, `Edit`. `auditlogs` has `List` + `Show` (read-only).
- **Multipart uploads**: `profiles`, `events`, `exhibitors`, `blogposts` use `FormData` (detected by `rawFile` on `ImageInput` values). All other resources use standard JSON. To add a new file-upload resource, add it to the multipart list in `dataProvider.ts`.

## Environment variables

All vars loaded from `.env` (see `.env.example`). Key vars:

| Var                                          | Purpose                                                                         |
| -------------------------------------------- | ------------------------------------------------------------------------------- |
| `DB_HOST/PORT/USER/PASSWORD/NAME/SSLMODE`    | Postgres connection                                                             |
| `jwtsecret_laganda`                          | HMAC-SHA256 secret for JWT signing. **Required.**                               |
| `S3_BUCKET`, `S3_ENDPOINT`, `S3_PUBLIC_URL`  | S3-compatible storage target (MinIO locally, Supabase Storage in hosted envs)   |
| `S3_REGION`                                  | Optional S3 region override when required by the endpoint                       |
| `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY` | Access credentials for the S3-compatible API (e.g. MinIO/Supabase storage keys) |
| `EVENTRO_API`, `EVENTRO_ORG`                 | Eventro API authentication                                                      |
| `AUDIT_LOG_RETENTION_DAYS`                   | Prune audit logs older than N days (default: 7)                                 |
| `PORT`                                       | Server port (default: 8080)                                                     |
| `DB_MAX_OPEN_CONNS`, `DB_MAX_IDLE_CONNS`     | Postgres connection pool tuning (optional)                                      |
| `INITIAL_ADMIN_USERNAME`                     | Username seeded on first startup (if DB empty)                                  |
| `INITIAL_ADMIN_PASSWORD`                     | Password for the seeded admin user (sensitive)                                  |
| `REVALIDATION_URL`                           | Public site revalidation endpoint (e.g. `https://armada.nu/api/revalidate`)     |
| `REVALIDATION_SECRET`                        | Shared secret for revalidation webhook auth                                     |
| `VERCEL_AUTOMATION_BYPASS_SECRET`            | Bypass Vercel Deployment Protection on staging/preview (optional)               |
| `DB_CONN_MAX_LIFETIME_MINUTES`               | Postgres connection max lifetime (optional)                                     |
| `DB_CONN_MAX_IDLE_TIME_MINUTES`              | Postgres idle connection timeout (optional)                                     |

## MCP configuration (`.vscode/mcp.json`)

- This repo and `armada.nu/` are often opened together in one multi-root workspace. VS Code merges MCP servers from **all active scopes** (user `mcp.json`, the `.code-workspace` file, and every folder's `.vscode/mcp.json`). If the **same server name** appears in more than one active scope, VS Code logs `WARN Overwriting mcp server '<name>'` and re-collects on every change, which can spin into an **infinite collection loop** that freezes the renderer (~1 Hz whole-window stutter).
- **Every MCP server name must be unique across all simultaneously-open scopes.** Suffix folder-scoped servers with the repo, e.g. `ESLint (ArmadaCMS)` / `ESLint (armada.nu)`, `markitdown (ArmadaCMS)` / `markitdown (armada.nu)`. Do not reuse a bare name (`ESLint`, `microsoft/markitdown`, `Chromatic`) that also exists in the other repo, the user config, or the workspace file.
- Diagnose suspected loops via `Developer: Toggle Developer Tools` → Console: a line repeating roughly once per second is the tell.

## Cleanup discipline

- When an approach fails, remove every artifact it produced — files created, config keys added, lockfile edits — before finishing the prompt. Do not leave dead configs, unused files, or failed workarounds in the codebase.

## Adding a new resource (checklist)

1. Create model in `models/` with GORM + camelCase JSON tags.
2. Register in `db.DB.AutoMigrate(...)` in `main.go`.
3. Write a SQL migration in `supabase/migrations/` for the schema change.
4. Create controller in `Controllers/` using `response_helpers.go` and **`audit_write_helpers.go`** for all writes.
5. Add routes in `main.go` — public GETs in `publicAPI`, write routes in `protectedAPI` with `auth.RequirePermission("resource.action", handler)`.
6. Create `List`, `Create`, `Edit` in `frontend/src/components/{Resource}/`.
7. Register `<Resource>` in `frontend/src/App.tsx`.
8. If file uploads: add to the multipart resource list in `frontend/src/dataProvider.ts`.
9. If the resource is displayed on the public site, pass the matching cache tag to the audit helper (`revalidateTags ...string`) and ensure the same tag is used in the Next.js data hook.
