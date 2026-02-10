<!--
Guidance for AI coding agents working on ArmadaCMS.
Standalone version for when this repo is opened without armada.nu in the workspace.
-->

# ArmadaCMS — Copilot instructions

## Architecture

Go REST API + React-Admin frontend in one repo. The Go server (Gorilla Mux, GORM, Postgres) exposes:
- `/api/v1/*` — REST endpoints consumed by the public site (armada.nu) and the admin frontend.
- `/admin/` — serves the built React-Admin SPA from `frontend/dist`.
- `/health` — healthcheck endpoint.

Deployed via Docker on AWS. The public site (separate repo: `armada-ths/armada.nu`) fetches from this API.

## Developer workflows

- **Backend**: `go run main.go` (port 8080). Requires Postgres — configure via env vars or use `docker compose up`.
- **Admin frontend** (`frontend/`): npm-based Vite app. `npm run dev` for dev, `npm run build` outputs to `frontend/dist`.
- **Tests**: no test framework is configured. Verify manually via `go run main.go` and checking endpoints.
- **Docker**: `docker compose up` brings up the Go server with DB connection.

## Backend patterns

- **Routing** (`main.go`): `publicAPI` subrouter (no auth) and `protectedAPI` subrouter (Bearer JWT via `auth.Middleware`). All routes under `/api/v1`. CRUD follows the pattern: `GET /resource`, `GET /resource/{id}`, `POST /resource`, `PUT /resource/{id}`, `DELETE /resource/{id}`.
- **Controllers** (`Controllers/`): each controller reads from `db.DB` (GORM global), JSON-encodes response. List endpoints support react-admin pagination via `Content-Range` headers and `utils.ParseListParams`.
- **Models** (`models/`): GORM structs with JSON tags (camelCase). Many-to-many relations use GORM's `many2many` tag (e.g., `Exhibitor` ↔ `Industry`, `Program`, `Employment`).
- **Auto-migration**: models are registered in `db.DB.AutoMigrate(...)` in `main.go`. Add new models there.
- **File uploads**: controllers accepting files use `multipart/form-data`; uploads go to AWS S3 via `utils/aws_s3.go`.
- **Auth**: JWT-based. `Controllers/AuthController.go` handles login. `auth/middleware.go` validates Bearer tokens and injects `user_id` into context.

## Admin frontend patterns

- **React-Admin v5** with `ra-data-simple-rest` data provider, customized in `frontend/src/dataProvider.ts`.
- **API endpoint**: resolved in `frontend/src/context/globalApi.ts` — `localhost:8080/api/v1` in dev, `window.location.origin/api/v1` in prod.
- **Auth**: JWT stored in `localStorage` (`accessToken`, `refreshToken`). Provider in `frontend/src/context/authProvider.ts`.
- **Resource components**: each resource has `List`, `Create`, `Edit` components in `frontend/src/components/{Resource}/`.
- **Multipart uploads**: `profiles`, `events`, `exhibitors` use `FormData` upload via the custom data provider. Other resources use standard JSON.

## Environment variables

Backend: `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`, `DB_SSLMODE` (see `db/connect.go`). Loaded from `.env` via `godotenv` or from system env vars.

## Adding a new resource (checklist)

1. Create model in `models/` with GORM + JSON tags.
2. Register in `db.DB.AutoMigrate(...)` in `main.go`.
3. Create controller in `Controllers/` following existing CRUD pattern.
4. Add routes in `main.go` (public vs protected as appropriate).
5. Create `List`, `Create`, `Edit` components in `frontend/src/components/{Resource}/`.
6. Register `<Resource>` in `frontend/src/App.tsx`.
7. If the resource has file uploads, add it to the multipart list in `frontend/src/dataProvider.ts`.
