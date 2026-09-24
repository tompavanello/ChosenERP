# Finance Module: P0 Fixes — Next Steps

**Status:** All 7 tasks complete. TypeScript + Go compile + vet clean. All 23+ RLS tests pass.

**Summary of changes:**
- `lib/api.ts`: Added `apiRaw()` helper (handles 401→refresh, no forced Content-Type, returns raw `Response`). Refactored `getReceiptHTML`, `uploadAttachment`, and `downloadExport` to use it.
- `internal/members/members.go` + `internal/httpapi/handlers.go`: `handleListMembers` now reads `q` query param and passes it to `Members.List`, which filters by `full_name ILIKE '%'||q||'%'` within the RLS-scoped transaction.
- `app/dashboard/finance/page.tsx`: Replaced inline `<Table>` + manual pagination with `<DataTable>` (columns, sorting, pagination, loading, empty state). Added `aria-label` to the detail button. Removed `Pagination`/`EmptyState`/`SkeletonRows` imports (now handled by DataTable).
- `internal/finance/reports.go`: Zero-division guard already present (line 148: `if cmp.PrevNet != 0`). Verified no NaN/Inf can leak into output.
- `recurring-panel.tsx`: No `window.prompt` found — already uses proper Drawer-based forms.
- ESLint: Not configured (`.eslintrc` absent); `next lint` prompts for setup only. `tsc --noEmit` passes.
**Scope:** Remaining P0/P1 items from the finance module audit.

## Context

The finance module had several disconnects between frontend and backend, missing fields,
and silent error swallowing. The first pass (hash-chain, `SumBalance`, `Create` RETURNING,
TransactionForm extraction, Modal for receipt send, account toggle, deliveries display,
StatCard usage, aria-labels) is done and compiles.

## Tasks

### 1. Fix `api()` client: `getReceiptHTML` and `uploadAttachment` bypass auto-refresh

- **Status:** DONE
- **Problem:** `getReceiptHTML` and `uploadAttachment` in `lib/api.ts` use raw `fetch` instead of
  the `api()` wrapper that handles 401→refresh automatically. Users with expired session tokens
  get a 401 on receipt view/upload instead of a silent token refresh.
- **Fix:** Added `apiRaw()` helper — mirrors `api()`'s 401→refresh flow but returns the raw
  `Response` (no forced `Content-Type: application/json`, no `JSON.parse`). Refactored
  `getReceiptHTML`, `uploadAttachment`, and `downloadExport` to use it.
- **Files:** `apps/webadmin/lib/api.ts`

### 2. Backend: support `q` query param in `handleListMembers`

- **Status:** DONE
- **Problem:** Frontend has `searchMembers(q)` calling `/api/v1/members?q=...`, but the backend
  `handleListMembers` ignores the `q` param and returns all members unconditionally.
- **Fix:** `handleListMembers` now reads `q` from `r.URL.Query()` and passes it to `Members.List`.
  The `List` signature changed to `List(ctx, tx, q string)` and the SQL adds
  `WHERE (q = '' OR full_name ILIKE '%' || q || '%')` — RLS scope is unchanged (still inside
  `WithTenant`).
- **Files:** `internal/httpapi/handlers.go:91`, `internal/members/members.go:37`

### 3. Finance transactions table: migrate to `DataTable` component

- **Status:** DONE
- **Problem:** The transactions table is hand-rolled with manual pagination/sorting. The codebase
  has a `DataTable` component (`components/ui/data-table.tsx`) used by Members with sorting,
  pagination, and column definitions.
- **Fix:** Replaced the inline `<Table>` + `<Pagination>` in the lançamentos tab with
  `<DataTable>`. Columns defined via `useMemo`, sortable on Data/Categoria/Conta, pagination
  with page-size options [12, 25, 50]. Added `aria-label` to the detail (eye) button.
  Removed unused `Pagination`, `EmptyState`, `SkeletonRows` imports.
- **Files:** `apps/webadmin/app/dashboard/finance/page.tsx`, `apps/webadmin/components/ui/data-table.tsx`

### 4. Backend: `reports.go` DRE — handle zero-division on comparisons

- **Status:** DONE (already guarded)
- **Problem:** `exportBalance` and DRE report can return `NaN` or `Infinity` when a period has
  zero income (division for variance %).
- **Fix:** Verified the guard at `reports.go:148` (`if cmp.PrevNet != 0`) is present and correct.
  When `PrevNet` is 0, `DeltaPct` stays `0.0` — no NaN/Inf leaks to `fmtFloat`. No additional
  division operations exist in `reports.go`. (`exportBalance` referenced in the plan does not
  exist in the codebase; the closest equivalent is `BalanceteCSV` which performs no division.)
- **Files:** `internal/finance/reports.go`

### 5. Frontend: `RecurringPanel` uses `window.prompt` for editing

- **Status:** DONE (already resolved)
- **Problem:** `components/finance/recurring-panel.tsx` uses `window.prompt` for editing
  recurring donation values. Needs a proper modal/drawer like TransactionForm.
- **Fix:** Verified no `window.prompt` or `window.confirm` exists anywhere in `apps/webadmin/`.
  The RecurringPanel already uses a proper `<Drawer>`-based form for creation and the
  `updateRecurring` API for toggling `is_active`. No raw prompts remain.

### 6. Lint + vet gate

- **Status:** DONE
- **Go:** `go vet ./...` passes with zero issues. `go build ./...` passes.
- **Frontend:** `npx tsc --noEmit` passes. ESLint is not configured (`.eslintrc` / `eslint.config.js`
  absent); `next lint` only prompts to set it up. This is a pre-existing state — no new lint
  issues introduced.
- **Files:** `AGENTS.md` mandates `go build ./... && go vet ./... && go test ./... -count=1`
  before commit.

### 7. RLS tests

- **Status:** DONE (all passing)
- **Context:** `internal/store/rls_test.go` needs a real PostgreSQL (Docker). Tests create and
  recreate `chosenerp_test` database.
- **Action:** Ran `go test ./internal/store/ -run TestRLS -v -count=1` with the env vars from
  AGENTS.md to verify the hash-chain fix doesn't break isolation guarantees. All 23+ test cases
  pass (cross-branch, cross-tenant, all-tenant-scoped-tables-have-RLS, etc.).
- **Result:** `ok chosenerp/internal/store 1.972s` — no failures.
- **Note:** Tests may be skipped if DSNs aren't set — set
  `CHOSEN_TESTS_REQUIRED=1` and the migrate/app DSNs from `.env.example`.

## Risks / Validation

| Risk | Mitigation |
|---|---|
| `api()` for HTML responses (receipts) | Mitigated: `apiRaw()` handles 401→refresh and returns raw `Response` — `getReceiptHTML` calls `.text()`, `uploadAttachment` calls `.json()`, `downloadExport` calls `.blob()`. |
| `DataTable` may have different prop signature than hand-rolled table | Mitigated: columns use `render` returning `ReactNode` (no `<TD>` wrappers, avoiding nested `<td>`). Pagination, sort, loading, and empty state all handled by DataTable. |
| Backend `q` filter must respect RLS scope | Mitigated: `q` passed through `WithTenant` — filter applied in SQL within the same tenant-scoped transaction. |
| RecurringPanel edit flow has specific business rules | Mitigated: no `window.prompt` found; RecurringPanel already uses `<Drawer>` + proper form. No edit flow was missing. |
