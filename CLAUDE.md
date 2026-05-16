# ArmadaCMS Claude notes

This file is a compatibility wrapper for non-Copilot agents. The canonical instructions live in `.github/copilot-instructions.md`.

Also check:

- `README.md` for setup, scripts, project structure, and developer workflows
- `.github/instructions/terraform.instructions.md` for `infra/terraform/**`

## Must-follow rules

- This repo owns the Go API and React-Admin app; public-site changes belong in `../armada.nu` as well.
- In controllers, do not write directly with `db.DB.Create/Save/Delete`; use the audit helpers so mutations, audit logs, and cache revalidation stay consistent.
- Register new DB models in `main.go` auto-migration.
- If a resource handles uploads, update `frontend/src/dataProvider.ts` so the admin app sends `FormData`.
- Regenerate Swagger docs when routes or annotations change.
- Validate backend changes with `go test -race -count=1 ./...`, and admin UI changes in `frontend/` with `npm run lint:check`, `npm run type-check`, and `npm run format:check`.
- Use the documented validation commands before finishing work.

When in doubt, follow `.github/copilot-instructions.md` over this summary.
