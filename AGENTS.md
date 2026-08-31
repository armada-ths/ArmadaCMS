# ArmadaCMS agent guide

This file exists for cross-agent compatibility. The canonical project instructions live in `.github/copilot-instructions.md`.

Also consult:

- `README.md` for setup, scripts, project structure, and developer workflows
- `.github/instructions/terraform.instructions.md` when editing `infra/terraform/**`

## Project scope

`ArmadaCMS` is the backend/admin repo: a Go REST API plus a React-Admin frontend.

If a task changes the public website, also update the sibling `../armada.nu` repo and follow its instructions.

## Fast path

- Do not run time-consuming scripts such as builds, full test suites, linters, or type checks after every prompt. Run them only when the scope or risk of the changes creates a realistic chance that the scripts will fail and reveal an error; otherwise use targeted, lightweight checks or inspection.
- Keep write operations in controllers on the audit helper path: use `createWithAudit[T]`, `updateWithAudit[T]`, and `writeDeleteResponseWithAudit[T]` rather than calling `db.DB.Create/Save/Delete` directly from controllers.
- Register every new DB model in `db.DB.AutoMigrate(...)` in `main.go`.
- When a change affects a public-site resource, pass the correct revalidation tag through the audit helper so the Next.js site cache is purged.
- If a resource uploads files, add it to the multipart handling list in `frontend/src/dataProvider.ts`.
- When routes or Swagger annotations change, regenerate the docs with `swag init --generalInfo main.go --output docs --parseInternal` and commit the generated files in `docs/`.
- When the validation policy above warrants it, validate backend changes with `go test -race -count=1 ./...`; validate admin UI changes in `frontend/` with `pnpm run lint:check`, `pnpm run type-check`, and `pnpm run format:check`.
- Copy `.env.example` to `.env` for local setup if needed; do not commit secrets.

## Useful examples

- `Controllers/audit_write_helpers.go` for all write-path controller mutations
- `Controllers/update_normalization_helpers.go` for trimmed optional strings and partial update maps
- `frontend/src/dataProvider.ts` for multipart resource wiring and admin API behavior

Keep this file concise and keep `.github/copilot-instructions.md` as the source of truth.
