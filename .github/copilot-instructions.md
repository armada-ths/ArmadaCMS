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

Deployed to **Google Cloud Run** (containerised). File storage: AWS S3.
- **Production**: DB is AWS RDS PostgreSQL.
- **Staging** (`staging.cms.armada.nu`): DB is Supabase PostgreSQL (no RDS).

See [docs/cloud-run-migration-plan.md](../docs/cloud-run-migration-plan.md) for infra context.

## Developer workflows

**Docker dev (recommended):**

```bash
docker compose -f docker-compose.dev.yml up --build  # first run
docker compose -f docker-compose.dev.yml up           # subsequent
```

This runs the Go API, React-Admin frontend, Postgres, and MinIO together with hot reload.

Default credentials: host `localhost`, db `armadacms`, user/password `postgres`. Copy `.env.example` → `.env`.

`npm run build` outputs to `frontend/dist`, which the Go server serves at `/admin/`.

**Tests**: none. Verify manually — `curl http://localhost:8080/health` and affected `/api/v1` endpoints.

**Swagger UI**: browsable API docs are served at `http://localhost:8080/swagger/index.html` while the server is running. After adding or changing routes, regenerate the spec with:

```bash
swag init --generalInfo main.go --output docs --parseInternal
```

Commit the generated `docs/` files alongside your code. Install the CLI once with `go install github.com/swaggo/swag/cmd/swag@latest`.

**Local data**: `scripts/import-remote-db.ps1` clones a remote Postgres DB into the local container.

**Terraform / HCP Terraform:** there are four roots (`gcp/prod`, `gcp/staging`, `aws/prod`, `aws/staging`). Avoid running `terraform plan` locally — the CLI-driven remote plan upload is slow. Prefer queueing plans from HCP Terraform when possible, and use local Terraform mainly for `validate`, `import`, or other targeted state operations. See [`infra/terraform/README.md`](../infra/terraform/README.md) and the per-root READMEs for workspace details.

## Backend patterns

- **Routing** (`main.go`): `publicAPI` (no auth) and `protectedAPI` (Bearer JWT) subrouters under `/api/v1`. Read-only list/get routes are typically public; write routes are protected.
- **Controllers** (`Controllers/`): read from `db.DB` (GORM global), JSON-encode responses. Use shared helpers from `Controllers/response_helpers.go` (`writeJSONResponse`, `writeCreatedJSONResponse`, `writeDeleteResponse`).
- **List endpoints**: use `utils.ParseListParams` (parses react-admin `sort`/`range`/`filter` query params) and set `Content-Range` header for react-admin pagination.
- **Models** (`models/`): GORM structs with camelCase JSON tags. Many-to-many via GORM `many2many` tag. Not all files in `models/` are DB models — `person.go` and `token.go` are response shapes.
- **Auto-migration**: every DB model must be registered in `db.DB.AutoMigrate(...)` in `main.go`.
- **Audit system** (critical): all write operations **must** use the generic helpers in `Controllers/audit_write_helpers.go` — `createWithAudit[T]`, `updateWithAudit[T]`, `writeDeleteResponseWithAudit[T]`. These wrap the mutation + audit log insert in one transaction atomically. Do **not** call `db.DB.Create/Save/Delete` directly from controllers. Old logs are automatically pruned on each audit insert (rate-limited to once per hour) based on `AUDIT_LOG_RETENTION_DAYS`.
- **File uploads**: controllers accepting files use `multipart/form-data`; files go to AWS S3 via `utils/aws_s3.go` (validates MIME, generates timestamped key).
- **Auth** (`auth/middleware.go`): validates HS256 JWT (`jwtsecret_laganda` secret), injects `user_id`, `role`, `permissions` into request context. Per-route permission check via `auth.RequirePermission("resource.action", handler)`. Permissions follow `"resource.action"` format; `"*"` grants full access. Use `auth.GetUserIDFromContext` etc. to read from context in controllers.
- **Session tokens**: access tokens expire in 15 minutes; the admin frontend holds a 7-day rotating refresh token in `localStorage`. `POST /api/v1/login` returns both. `GET /api/v1/refreshAccessToken` (public route, `X-RefreshAuthorization: Bearer <token>` header) rotates the refresh token and issues a new access token. Ensure `jwtsecret_laganda` differs between staging and production.
- **Initial admin seeding**: on startup `SeedInitialAdminUser` runs once — it creates a user from `INITIAL_ADMIN_USERNAME` / `INITIAL_ADMIN_PASSWORD` env vars only if no users exist yet. Remove or leave empty once real accounts are created.
- **Eventro integration**: `Controllers/EventroController.go` proxies the external Eventro API (exhibitors/events/members/recruitments). Uses `EVENTRO_API`, `EVENTRO_FAIR_ID`, `EVENTRO_ORG` env vars. Triggered from the `EventroSync` admin page.
- **Feature flags**: `FeatureFlagController` seeds default flags on startup (`models/feature_flag.go`). Exhibitor signup open/closed state is computed on the frontend (`armada.nu`) based on IR/FR date windows from the dates API — it is **not** controlled by a feature flag.
- **Blogpost model** (`models/blogpost.go`) exists and is auto-migrated but has no controller, no API routes, and no admin UI — it is legacy and should not be extended without a deliberate decision.

## Admin frontend patterns

- **React-Admin v5** with `ra-data-simple-rest` data provider, customised in `frontend/src/dataProvider.ts`.
- **API endpoint**: `frontend/src/context/globalApi.ts` — `localhost:8080/api/v1` in dev, `window.location.origin/api/v1` in prod.
- **Auth**: JWT in `localStorage` (`accessToken`, `refreshToken`). Provider: `frontend/src/context/authProvider.ts`.
- **Resource components**: `frontend/src/components/{Resource}/` — each has `List`, `Create`, `Edit`. `auditlogs` has `List` + `Show` (read-only).
- **Multipart uploads**: `profiles`, `events`, `exhibitors` use `FormData` (detected by `rawFile` on `ImageInput` values). All other resources use standard JSON. To add a new file-upload resource, add it to the multipart list in `dataProvider.ts`.
- **Custom page**: `EventroSync` at `/eventrosync`, requires `eventrosync.access` permission — registered as a custom route in `App.tsx`.

## Environment variables

All vars loaded from `.env` (see `.env.example`). Key vars:

| Var                                             | Purpose                                           |
| ----------------------------------------------- | ------------------------------------------------- |
| `DB_HOST/PORT/USER/PASSWORD/NAME/SSLMODE`       | Postgres connection                               |
| `jwtsecret_laganda`                             | HMAC-SHA256 secret for JWT signing. **Required.** |
| `S3_BUCKET`, `AWS_REGION`                       | S3 file storage (region default: `eu-north-1`)    |
| `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`    | S3 credentials (optional if IAM role in use)      |
| `EVENTRO_API`, `EVENTRO_FAIR_ID`, `EVENTRO_ORG` | Eventro proxy integration                         |
| `AUDIT_LOG_RETENTION_DAYS`                      | Prune audit logs older than N days (default: 7)   |
| `PORT`                                          | Server port (default: 8080)                       |
| `DB_MAX_OPEN_CONNS`, `DB_MAX_IDLE_CONNS`        | Postgres connection pool tuning (optional)        |
| `INITIAL_ADMIN_USERNAME`                        | Username seeded on first startup (if DB empty)    |
| `INITIAL_ADMIN_PASSWORD`                        | Password for the seeded admin user (sensitive)    |

## Adding a new resource (checklist)

1. Create model in `models/` with GORM + camelCase JSON tags.
2. Register in `db.DB.AutoMigrate(...)` in `main.go`.
3. Create controller in `Controllers/` using `response_helpers.go` and **`audit_write_helpers.go`** for all writes.
4. Add routes in `main.go` — public GETs in `publicAPI`, write routes in `protectedAPI` with `auth.RequirePermission("resource.action", handler)`.
5. Create `List`, `Create`, `Edit` in `frontend/src/components/{Resource}/`.
6. Register `<Resource>` in `frontend/src/App.tsx`.
7. If file uploads: add to the multipart resource list in `frontend/src/dataProvider.ts`.
