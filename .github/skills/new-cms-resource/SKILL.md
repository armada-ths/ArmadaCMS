---
name: new-cms-resource
description: 'Create or update an ArmadaCMS resource end to end. Use for new Go REST resources, admin CRUD screens, React-Admin resource registration, AutoMigrate wiring, audit-helper write paths, multipart upload resources, Swagger updates, and cache revalidation tag setup when the resource feeds armada.nu.'
argument-hint: '[resource name] [public/private] [has uploads?] [revalidates armada.nu?]'
---

# New CMS Resource

Use this skill when adding a new ArmadaCMS resource or extending an existing one across the backend API and React-Admin frontend.

## Before You Start

- Read `README.md` and `.github/copilot-instructions.md` for the canonical architecture and validation rules.
- Use examples from:
  - `Controllers/audit_write_helpers.go`
  - `Controllers/update_normalization_helpers.go`
  - `frontend/src/dataProvider.ts`
  - `main.go`
- If the new resource is also consumed by `../armada.nu`, plan the matching frontend hook/tag work there instead of inventing API-only fields in isolation, and coordinate the cross-repo changes when practical in the same task.

## Gather Decisions Up Front

Confirm these details before writing code:

1. Resource name in singular and plural forms.
2. Whether reads are public (`publicAPI`) or admin-only (`protectedAPI`).
3. Permission names to use (`resource.list`, `resource.show`, `resource.create`, `resource.edit`, `resource.delete`).
4. Whether the model needs file uploads.
5. Whether writes should trigger armada.nu cache revalidation, and if so which tag string.
6. What persisted schema changes are required, so you can add the matching checked-in Supabase migration immediately.

## Procedure

### 1. Model and schema

1. Add the GORM model in `models/`.
2. Use camelCase JSON tags to match existing API responses.
3. If the resource introduces persistent schema changes, register it in `db.DB.AutoMigrate(...)` in `main.go`.
4. If the resource introduces persistent schema changes, generate or update the checked-in migration under `supabase/migrations/` in the same task.

### 2. Controller implementation

1. Add a controller file in `Controllers/` or extend an existing one.
2. For list endpoints, use `utils.ParseListParams` and set `Content-Range` for React-Admin pagination.
3. Use `writeJSONResponse`, `writeCreatedJSONResponse`, and the other helpers from `Controllers/response_helpers.go`.
4. For write paths, do **not** call `db.DB.Create`, `Save`, or `Delete` directly.
5. Use the audit helpers instead:
   - `createWithAudit[T]`
   - `updateWithAudit[T]`
   - `writeDeleteResponseWithAudit[T]`
6. If updates include optional strings or patch-style partial updates, use the normalization helpers in `Controllers/update_normalization_helpers.go`.
7. If the resource is visible on `armada.nu`, pass the matching revalidation tag string to the audit helper call.

### 3. Routing and permissions

1. Add routes in `main.go` under `/api/v1`.
2. Put public read endpoints on `publicAPI`.
3. Put writes and admin-only reads on `protectedAPI` with `auth.RequirePermission(...)`.
4. Follow the existing permission naming pattern: `resource.action`.
5. Keep the route shape consistent with existing resources: collection `GET/POST`, item `GET/PUT/DELETE`.

### 4. Admin frontend wiring

1. Create `List`, `Create`, and `Edit` components in `frontend/src/components/{Resource}/`.
2. Register the resource in `frontend/src/App.tsx`.
3. If the resource accepts uploaded files, add it to the multipart resource list in `frontend/src/dataProvider.ts`.
4. Match the surrounding React-Admin and MUI patterns instead of introducing a new form abstraction.

### 5. Swagger and docs

1. If routes or Swagger annotations changed, regenerate docs with `swag init --generalInfo main.go --output docs --parseInternal`.
2. Commit the generated `docs/` changes together with the controller and route changes.

### 6. Cross-repo follow-through

If the resource feeds the public site:

1. Update `../armada.nu` in the same task if practical, following the instructions in `.github/skills/new-public-page-route/SKILL.md`.
2. Add or update the API hook in `src/components/shared/hooks/api/` using the exact same cache tag string.
3. If the resource powers a new public page, also update `src/app/sitemap.ts` and any relevant feature flags there.

## Branching Rules

### If the resource has uploads

- Reuse the `FormData` flow in `frontend/src/dataProvider.ts`.
- Validate the backend controller's multipart handling and S3-compatible upload path.
- Confirm the resource name is added to the multipart upload list, or the admin UI will quietly send JSON instead.

### If the resource is read-only in the admin UI

- You may omit `Create`/`Edit`, but still register the resource appropriately in `frontend/src/App.tsx`.
- Keep permission checks aligned with the routes you actually expose.

### If the resource is not shown on armada.nu

- Skip revalidation tags.
- Do not add speculative frontend hooks in the sibling repo.

### If the resource changes the remote schema

- Do not stop at AutoMigrate.
- Ensure the checked-in Supabase migration exists so staging and production stay aligned.

## Completion Checklist

Do not consider the resource done until you have checked all relevant items:

- Model exists in `models/`.
- Model is registered in `db.DB.AutoMigrate(...)` when needed.
- Checked-in Supabase migration exists for every persisted schema change.
- Controller uses response helpers.
- Every write path uses audit helpers.
- Routes are wired in `main.go` with correct auth and permission names.
- Admin resource is registered in `frontend/src/App.tsx`.
- Multipart upload wiring is updated if needed.
- Matching revalidation tags are passed when the resource feeds `armada.nu`.
- Swagger docs regenerated and committed when routes/annotations changed.
- Validation run for both affected layers.

## Validation

Run the checks that match the scope of the change:

- Backend: `go test -race -count=1 ./...`
- Admin frontend in `frontend/`: `npm run lint:check`, `npm run type-check`, `npm run format:check`
- If Docker wiring or local environment behavior changed, verify with `docker compose -f docker-compose.dev.yml up --build`

## Common Pitfalls

- Forgetting `db.DB.AutoMigrate(...)` after adding a model.
- Adding or changing persistent schema locally without a checked-in migration under `supabase/migrations/`.
- Writing directly with GORM and bypassing audit logs and revalidation.
- Adding an upload field but forgetting the multipart resource list in `frontend/src/dataProvider.ts`.
- Adding a public-site resource without passing the revalidation tag.
- Updating routes but forgetting to regenerate `docs/`.
- Treating local AutoMigrate as sufficient for staging/production schema changes.
