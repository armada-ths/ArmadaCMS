---
description: "Scaffold a new CMS resource end-to-end: Go model, controller, routes, React-Admin UI, and optionally an armada.nu data hook with cache revalidation"
agent: "agent"
argument-hint: "Resource name and fields, e.g. 'Sponsor with name:string, logoUrl:string, tier:string'"
---

# Add a CMS resource

Scaffold a new resource across ArmadaCMS (backend + admin UI) and optionally the public site (armada.nu).

Follow the "Adding a new resource" checklist in [ArmadaCMS copilot-instructions.md](../../.github/copilot-instructions.md) for conventions and patterns. Reference existing resources (e.g. `HighlightCard`, `Blogpost`) as implementation examples.

## Gathering input

Before generating any code, confirm the following with the user:

1. **Resource name** — singular PascalCase (e.g. `Sponsor`)
2. **Fields** — name and Go type for each (e.g. `name:string`, `logoUrl:*string`, `tier:string`)
3. **File uploads** — does the resource have an image or file field?
4. **Public reads** — should unauthenticated users be able to list/get this resource, or is it admin-only?
5. **Public site display** — is it displayed on armada.nu? (if yes, needs a data hook + cache revalidation tag)

If the user provides only a resource name or explanation of the resource, **suggest reasonable defaults** for fields, visibility, and file uploads based on the user's input, then ask the user to confirm or adjust before proceeding.

## Implementation

Work through the checklist steps sequentially. For each step, follow the exact patterns documented in copilot-instructions.md and match the style of existing resources.

Key decisions that vary per resource:

- **Route visibility**: if public reads, register GET routes on `publicAPI`. If admin-only, register all routes (including reads) on `protectedAPI` with appropriate permissions.
- **File uploads**: if present, use `multipart/form-data` parsing in the controller and add the resource to the multipart list in `frontend/src/dataProvider.ts`.
- **Cache revalidation**: if displayed on the public site, pass a tag string to the audit helpers' `revalidateTags` argument and create the corresponding data hook in armada.nu. Add the new tag to the tag inventory in `armada.nu/.github/copilot-instructions.md`.

## After completion

- Run `go vet ./...` in ArmadaCMS to check for compile errors.
- Run `pnpm type-check` in armada.nu if a data hook was created.
- Regenerate Swagger docs: `swag init --generalInfo main.go --output docs --parseInternal`
