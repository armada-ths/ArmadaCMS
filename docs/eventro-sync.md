# Eventro synchronization

Exhibitors, events, member profiles, fair dates, recruitment periods and recruitment roles share field ownership rules. Only fields mapped from Eventro are synchronized; CMS-only fields remain owned by the CMS.

For each record and mapped field, the backend stores the latest Eventro value in the private `eventro_sync_states` table. When the current CMS value equals that baseline, a new Eventro value is applied. When the CMS value differs, the CMS correction is retained. A manual editor save automatically acknowledges the changed field as CMS-owned and clears its current conflict. Review reads never manufacture conflicts from CMS-only differences. Only a subsequent sync with a changed Eventro value can reopen that conflict. Choosing **Keep CMS** explicitly retains CMS ownership and acknowledges the current Eventro version. Acknowledged differences disappear from review; a row remains only if it has other unresolved differences or data-quality issues. A later Eventro change requires review again. Choosing **Use Eventro** applies that value and resumes synchronization for the field.

Existing records have no trustworthy import baseline. Their non-empty differences are preserved and presented for review on the first sync. Missing fields can be filled. Explicit CMS clearing after a baseline exists is also a correction. Category relationships are compared by normalized names, independent of database IDs or list ordering. A logo and its thumbnail are reviewed together.

The dashboard's **Review sync and data quality** dialog selects a resource and shows CMS/Eventro values for unresolved differences, data-quality warnings and links to the normal editor. Exhibitors additionally report missing profile data and products without a package mapping. Decisions require both `eventrosync.access` and the resource's `.edit` permission. If either value has changed since the review loaded, the decision fails with HTTP 409; refresh before deciding again. Import metadata, decisions and content changes are audited transactionally. Metadata is never returned by public resource endpoints, and its table has no grants for Supabase client roles.

Paginated exports are fetched and validated completely before content import. Duplicate IDs, changing totals, missing lists and truncated pagination fail the fetch. A source fetch failure does not delete data. No sync deletes records that are absent from an export, including other fair-date configurations. When several fair-date configurations exist, the review highlights them; the public dates endpoint continues to use the highest ID. Remove obsolete configurations through the normal authorized CMS workflow when appropriate.

## Category normalization

Exhibitor industries, employment types and fields of study are normalized before field ownership comparison and category persistence using `internal/eventrocategories/mapping.csv`. The CSV is embedded in the backend binary. Matching ignores case and redundant whitespace; aliases that resolve to the same category are deduplicated within their category type. Unknown labels are retained. The `Review` column is retained as mapping metadata and does not change the specified normalized value. Locations are unaffected.

Mapping changes take effect on the next exhibitor sync. Existing field ownership and conflict review still apply: manual CMS corrections and untracked historical differences are preserved. Normalization does not delete existing category records or change unrelated CMS-managed relationships.

## Package mapping

Package tiers are defined in `Controllers/eventro_product_mapping.go`: `Startup Kit` maps to `Standard`, `Bronze Kit` to `Bronze`, `Silver Kit` to `Silver`, and `Gold Kit` to `Gold`. Matching is exact and case-sensitive on `displayName`, without falling back to `name` or `productId`. Missing, null or unknown display names remain unmapped and appear for review; product IDs are retained for diagnostics. The highest tier wins regardless of product order. An unknown tier preserves an existing CMS tier and leaves a new record unclassified. Exclude add-on products from the API export in Eventro's dashboard. Changes to the package mapping require a code change; the next exhibitor sync refreshes the saved report.

The website uses `logoThumbnail` for Eventro logos while retaining `logoFreesize` as the original. A manually supplied `logoSquared` remains preferred. Editing the original logo clears its imported thumbnail so the two cannot diverge. Locations are normalized and shown in the company details.

## Boundaries

- Eventro member export has no stable person ID. Existing name-derived matching remains in use; a name change can require manual reconciliation. When a member appears in several ordered security groups, the first occurrence determines imported rank.
- Optional archived event fields are omitted from comparison; explicit null values for active custom fields can clear an unmodified CMS value. Unknown field types are ignored.
- Missing recruitment team information is not interpreted as an instruction to clear a team.
- Record creation and existing publication behavior remain as before. Conflict review is not a publication approval queue, and synchronization is still manually initiated.

## Validation

Unit tests cover field ownership for every resource, legacy baselines, explicit clearing, deterministic product mapping, category comparison and complete pagination. The opt-in `TestEventroReviewDatabaseFlow` uses `EVENTRO_SYNC_TEST_DSN` and refuses any database other than a local `eventro_sync_test` database. It covers persistence, resource permissions, stale decisions, ownership reset and preservation of other fair configurations.
