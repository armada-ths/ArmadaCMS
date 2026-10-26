# Public content translations

The admin interface stays English. Original content fields remain the default/English source, with optional Swedish fields beside them. Public clients select each non-blank Swedish field independently and otherwise use the original. Do not translate personal/company names, IDs or grouping keys.

Every translated field must be connected through its model, additive SQL migration, create/update handler, admin form and public response. Regenerate Swagger when contracts change. Preserve audit transactions and the existing revalidation tags so both language routes see updated content.

## Update contract

An omitted field preserves its value. JSON null, an empty string or whitespace clears a translation to SQL NULL; non-empty input is trimmed. Controllers use the shared localization helpers. Multipart requests must include an empty field to clear a translation; the admin multipart serializer converts null translation values to empty fields and leaves omitted fields out.

Event descriptions are sanitized in both languages. Eventro synchronization uses original fields/IDs and leaves editorial Swedish fields intact. Keep explicit sync update-column lists; never replace them with full-model saves that could clear translations.

## Public response details

- Highlights: titleSv, subtitleSv, descriptionSv, linkTextSv.
- Blogs: titleSv, textSv; events: nameSv, descriptionSv, locationSv, openToSv, foodSv, feeSv.
- Exhibitors: aboutSv; employment, industry and program labels: nameSv.
- Organization: group nameSv and person roleSv (from profile titleSv). Rank values and group keys stay unchanged.
- Recruitment: period nameSv, role nameSv/descriptionSv and groupNamesSv keyed by original team name. Original group keys remain stable for business logic.
- Fair dates: fair.descriptionSv.
- History: entry titleSv/bodySv and eraTitleSv; era administration exposes titleSv.

Photo-event translations and language-specific link/QR generation are maintained in the Photos follow-up. These fields are additive so an older frontend/backend can run without deleting translations.

## Verification

The backend tests cover normalization, omitted/null/blank JSON updates and multipart clearing across public content models. Highlight tests inspect generated SQL for explicit NULL writes; recruitment response tests protect stable grouping and translated labels. Run the backend tests with the repository's documented libvips setup, and run the admin type/lint/format checks after form or serializer changes.
