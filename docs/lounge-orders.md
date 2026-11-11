# Lounge orders

## Catalogue and opening hours

Under **Fair & events > Lounge catalogue**, manage localized categories and items, set per-order quantity limits, and mark items sold out. Drag items or use the keyboard to reorder them or move them between categories. Deleting a category removes its catalogue items. Saved orders retain their original item and category details.

Under **Fair dates**, set opening and closing times for each fair day in Stockholm time. The ordering cutoff is the number of minutes before each day closes. A cutoff of 0 keeps ordering open until the fair closes.

## Permissions

| Permission                                                                    | Access                                           |
| ----------------------------------------------------------------------------- | ------------------------------------------------ |
| `lounge-catalogue.view`                                                       | Show the catalogue in CMS                        |
| `lounge-catalogue.create`, `lounge-catalogue.edit`, `lounge-catalogue.delete` | Manage categories and items                      |
| `lounge-orders.view`                                                          | View orders, details, and totals                 |
| `lounge-orders.edit`                                                          | Update fulfilment status and queue Slack retries |

The Order app reads the public catalogue. Order details and totals require CMS permissions.

## Order handling

**Lounge orders** shows saved orders with the company, items, and categories recorded at submission. Filter by company, category, item, fulfilment status, or real/test orders. Real orders are shown by default. Quantity totals cover all matching order lines across pages.

CMS saves an order and its pending Slack notification in the same transaction. Cloud Scheduler dispatches pending notifications every minute. From the order detail, staff can queue a retry after a failure or an unknown outcome. Retrying an unknown outcome can duplicate the Slack message while preserving a single order record.

## Local development and tests

Use the lounge variables in [the CMS environment example](../.env.example). Match the submission secrets to the Order app and use a dedicated `SLACK_ORDER_TEST_HOOK_URL` to enable local test notifications.

From `frontend/`, run `pnpm exec vitest run --config vitest.lounge.config.ts` for the catalogue and order browser tests. Run `go test -race -count=1 ./Controllers -run '^TestLounge'` from the repository root for backend tests. The integration tests require Go/libvips and `LOUNGE_TEST_DSN` for a dedicated, migrated PostgreSQL database named `lounge_plan_test`. They truncate fixture tables in that database.
