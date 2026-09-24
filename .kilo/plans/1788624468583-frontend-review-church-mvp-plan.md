# Chosen ERP — Frontend Review & MVP Assessment (Members + Finance)

**Role:** Frontend Specialist + Church Management Specialist
**Date:** 2026-09-22
**Scope:** Review of `apps/webadmin`, documentation assessment, attention points, and a concrete MVP plan for member & financial management with a modern full-screen layout.

---

## 1. Executive Summary

The frontend is **more mature than the documentation backlog suggests**. All MVP-scoped pages (members, visitors, benefactors, families, finance, transfers, ministries, reports, dashboard) are implemented and functional against the existing Go API. The codebase has a thoughtful component library, proper auth/RBAC, dark mode, and responsive patterns.

However, there is a **critical layer desync**: the frontend was partially refactored toward a "unified People API" abstraction (`/api/v1/people`, `Person` type, SWR hooks, `EntityDataTable`, `PersonCard`, `UnifiedDrawer`) that has **no backend implementation** in `router.go`. This means ~30% of the frontend code is dead/unused — it cannot work at runtime.

**Bottom line:** The *functional* frontend is a solid, shipping MVP. The *architectural* frontend has a half-finished refactor that must be resolved (complete or revert) before it can be considered production-ready.

---

## 2. Documentation Assessment

### 2.1 Strengths
- **`Docs/00_Checkpoint.md`** is excellent — consolidates the entire project history, decisions, status per phase, bugs found and fixed (cross-tenant leak, `/me` endpoint), and known debt. A rare level of honesty and engineering maturity.
- **`Docs/01_Blueprint_Arquitetura.md`** gives concrete technical decisions (monolito modular Go, RLS by branch_id, Go:embed migrations, `FROM scratch` images, ports >30000).
- **`Docs/02_Backlog.md`** is prioritized (P0-P4) with sequencing notes and blocking relationships.
- **`AGENTS.md`** is comprehensive: full endpoint table, how-to-run, credentials, test instructions.
- The **PRD** (`Chosen_ERP_Documentacao_Completa.md`) is thorough — 12 modules, RBAC, security, KPIs, risks, roadmap with weekly estimates.

### 2.2 Issues in Documentation
| Problem | Location | Impact |
|---|---|---|
| `pastor.norte@demo.local` documented as dev credential | `README.md`, `AGENTS.md` | **No backend migration or seed creates this user or the "Norte" branch.** The seed (000009) only creates `demo` tenant, "Sede Matriz" branch, and `admin@demo.local`. This misleads anyone trying to test multi-branch isolation. Backlog #40 confirms this is unresolved. |
| `lib/tokens.ts` referenced in plan but doesn't exist | Plan file mentions it as "dead code to delete" | The plan file was aspirational — it references a file that was never created. Indicates planning artifacts may not be fully aligned with the working tree. |
| No frontend test strategy documented | `AGENTS.md` covers Go build/vet/test only | Backlog #41 (frontend tests) exists as a line item but no framework, command, or tooling is specified. |
| `app/member/[token]` vs PRD "app do membro" | Code only has a public web page | The README/Checkpoint say "App do membro v1" is done as a public page. The PRD envisioned a full native app (Expo). Manage expectations with potential users. |

---

## 3. Frontend Architecture Review

### 3.1 What Works Well
- **Component library** (`components/ui/*`) is well-factored: Button (3 variants × 3 sizes), Card (header/title/content), Badge (10 tones × 3 variants), DataTable (sorting/pagination/loading/empty/selection), Modal + Drawer, Tabs, Pagination, Avatar (initials), StatCard, Combobox (keyboard nav, search), MaskedInput (CPF/RG/phone/CNPJ), Toast, Breadcrumbs.
- **Auth** (`auth-provider.tsx` + `lib/api.ts`): JWT access/refresh, **auto-refresh on 401**, `useAuth()` context with `hasPerm()`, menu filtering by permission, `registerSessionLost` callback. The `api()` core fetcher handles token refresh transparently.
- **`lib/api.ts`** is comprehensive: typed interfaces for all entities (Member, Transaction, Balance, DRE, Family, Visitor, Branch, Transfer, RecurringDonation, Ministry, SmallGroup, Announcement, PublicCardData), `apiRaw` for HTML/blob/multipart responses, auto-refresh preserved across all fetch paths.
- **Dark mode** toggle with localStorage persistence + system preference detection.
- **Responsive**: DataTable hides mobile cards on `lg:`, pages show native mobile card layouts on `lg:hidden`.

### 3.2 The "People API" Layer Desync (Critical)

The following frontend components/functions reference `/api/v1/people` endpoints that **do not exist** in `router.go`:

| Frontend Asset | References | Backend Route |
|---|---|---|
| `lib/api.ts` → `listPeople`, `createPerson`, `updatePerson` | `/api/v1/people` and `/api/v1/people/{type}/{id}` | **Does not exist** in `router.go` |
| `lib/swr-hooks.ts` → `usePeople`, `useCreatePerson`, `useUpdatePerson` | Uses `listPeople` / `createPerson` / `updatePerson` | Dead — API not implemented |
| `lib/swr-hooks.ts` → `useMembers`, `useCreateMember`, `useUpdateMember`, etc. | Uses `listMembers` / `createMember` / `updateMember` (these DO exist) | Works, but **never imported** by any page |
| `components/people/entity-data-table.tsx` → `EntityDataTable` | Uses `DataTable` wrapper, designed for `listPeople` | **Not used** by any page |
| `components/people/person-card.tsx` → `PersonCard`, `personStatusTone`, `personStatusLabel` | Uses `Person` type | **Not used** by any page |
| `components/people/unified-drawer.tsx` → `UnifiedDrawer` | Uses `createPerson`/`updatePerson`/`convertVisitorToMember` | **Not used** by any page |
| `components/people/journey-timeline.tsx` → `JourneyTimeline`, `JourneyStageBadge` | Uses `JOURNEY_ORDER` | **Not used** by any page |

**Conclusion:** The frontend contains a complete, parallel "next-generation" component layer (`components/people/*`, `lib/swr-hooks.ts`, the unified `Person` type) that was built but never wired into the actual pages. All live pages (`dashboard/members/page.tsx`, `visitors/page.tsx`, etc.) use the entity-specific APIs (`/api/v1/members`, `/api/v1/visitors`, etc.) and `useEffect` + local state — which works. The "people" layer is speculative refactoring that doesn't run.

**Recommendation:** Either (A) implement `/api/v1/people` on the backend and migrate pages to use the new components, or (B) delete the dead `people` components, `swr-hooks.ts`, and the `/people` API functions in `lib/api.ts`. For the MVP scope, option B + a focused incremental adoption is lower-risk.

### 3.3 Existing Plan Files
Two plan files exist in `.kilo/plans/`:
- `1788624468583-ux-layout-registration-suite.md` — A 424-line redesign plan for the registration suite (members/families/visitors/benefactors). It correctly identifies: duplicated mobile cards, broken SWR `mutate` (using `window.dispatchEvent`), the RG mask bug, and missing detail pages. **However, it is partially stale**: it references deleting `lib/tokens.ts` (doesn't exist), and claims the SWR `mutate` bug exists (it's actually already fixed — `swr-hooks.ts` uses `swrMutate()` correctly). Much of what it proposes is already done or is dead code.
- `1788617554483-finance-p0-next-steps.md` — Documents 7 completed finance P0 fixes. Status: "complete."

---

## 4. Attention Points (Bugs & Risks)

### 4.1 Confirmed Bugs

| # | Bug | Location | Severity |
|---|---|---|---|
| 1 | **RG mask bug** — member detail edit form uses `mask="phone" variant="rg"` for the RG field. The `MaskedInput` component supports `mask="rg"` natively but `variant` prop is ignored. | `app/dashboard/members/[id]/page.tsx:163` | Medium — RG is formatted as a phone number during edit |
| 2 | **Members page layout** is constrained to `max-w-6xl` — does not occupy full screen width as required. | `app/dashboard/members/page.tsx:370` and nearly all dashboard pages | UX — wastes horizontal space on wide screens |
| 3 | **`renderMobileCard` duplicated** — members page has its own mobile card implementation (`renderMobileCard` at line 317) alongside the `DataTable` component which also supports `renderMobileCard`. The page hides DataTable on mobile (`lg:hidden`) and shows its own card list. This is redundant and will diverge from the shared `DataTable` mobile implementation. | `app/dashboard/members/page.tsx:444-482` vs `components/ui/data-table.tsx:266-275` | Medium — maintainability |
| 4 | **Dashboard overview "Últimos lançamentos"** is a placeholder with hardcoded "Sem lançamentos recentes." — never fetches real data. | `app/dashboard/page.tsx:149-157` | Low-Medium — missing feature |
| 5 | **No `next.config.js`** — the project uses `next.config.ts` but there's no `tailwind.config.js` — Tailwind v4 uses `@tailwindcss/postcss` ( PostCSS) configured in `postcss.config.mjs`. There's no explicit Tailwind config file. This works but limits customization of the design system (tokens, themes, custom breakpoints). | `apps/webadmin/` | Low — but prevents design-system customization |

### 4.2 Functional Gaps (vs PRD MVP scope)

| Gap | Current State | Impact |
|---|---|---|
| **Members list pagination** — fetches ALL members via `listMembers()` (no server-side pagination). The API supports `q` param (added per backlog plan) but no `page`/`page_size`. | `app/dashboard/members/page.tsx:40` — `listMembers()` returns all. Client-side filter/sort/paginate. | Performance: degrades with >200 members |
| **Benefactor list** — raw `<Table>` with no pagination, no row actions, no detail page. | `app/dashboard/benefactors/page.tsx` | Cannot edit benefactor; large lists overflow |
| **Family list** — raw `<Table>`, no pagination, no detail page, `head_id` lookup is client-side. | `app/dashboard/families/page.tsx` | No family detail view; head name resolution is a workaround |
| **Member detail page** — inline per-tab editing (state lost when switching tabs). The plan correctly flagged this. | `app/dashboard/members/[id]/page.tsx` | Poor UX — form data lost on tab switch |
| **No family detail page** (`/dashboard/families/[id]`) | Missing route | Cannot view family members or edit family |
| **No visitor detail page** (`/dashboard/visitors/[id]`) | Missing route | Cannot convert via detail view; conversion only from list |
| **No benefactor detail page** (`/dashboard/benefactors/[id]`) | Missing route | Cannot view benefactor donation history |
| **No member financial history** on detail page | Member detail has no finance tab | Dízimo/oferta tracking requires leaving the page |
| **No family member listing on detail** — `GET /api/v1/families/{id}/members` exists in router but frontend `listFamilies()` doesn't fetch members. | Backend has `handleFamilyMembers`, frontend doesn't use it | Family detail page can't show members |
| **No benefactor-specific transactions** — `/api/v1/benefactors/{id}` doesn't exist on backend; no way to GET a single benefactor. | `router.go` only has GET/POST benefactors (no GET/{id}) | Individual benefactor view impossible |

### 4.3 Architectural Debt

| Issue | Detail |
|---|---|
| **CSS-in-base** — `globals.css` defines raw CSS classes (`.card`, `.btn-base`, `.btn-primary`, `.input`, `.label`, etc.) **in addition to Tailwind utility classes** used by components. This creates two styling systems: semantic CSS classes AND Tailwind utilities. Components like `Button` mix `className="btn-base"` (CSS class) with Tailwind `cn()` calls. This is inconsistent and hard to maintain. |
| **No design tokens** — `globals.css` uses raw CSS custom properties (`--ink`, `--paper`, `--brand`). The AGENTS.md mentions shadcn/ui but the project built its own lightweight system. No `tailwind.config.js` means no theme extension, custom colors, or consistent spacing scale. |
| **Inline styles in chart components** — Recharts `Tooltip`, `XAxis` etc. use hardcoded hex colors (`#9ca3af`, `#e5e7eb`) instead of CSS variables — dark mode doesn't propagate to charts. |
| **`distDir` collision** — `next.config.ts` has a comment about dev/build sharing the same `.next` folder and corruption. This is a real issue if running `npm run dev` and `npm run build` concurrently. |

### 4.4 Security Observations

| Issue | Detail |
|---|---|
| **CORS is wildcard** — `withCORS` sets `Access-Control-Allow-Origin: *`. Acceptable for dev but dangerous for production. | `router.go:158-168` |
| **JWT secret is dev-insecure** — `JWT_SECRET=dev-insecure-secret-change-me` in `.env.example`. Documented but should have a stronger default or generation hint. | `.env.example:33` |

### 4.5 Multi-Tenant / Church Management Context

From an institutional church management perspective:

| Concern | Assessment |
|---|---|
| **Branch scoping** — The frontend shows `user.branch_id || "Sede"` in the header. The API correctly enforces RLS per branch. The frontend trusts the backend for scoping — this is correct. | Good |
| **No tenant switching** — The frontend has no way to switch tenants at runtime. Only one tenant context per login. For a multi-tenant SaaS where a Super Admin might manage multiple churches, this is a gap (but expected for the MVP). | Acceptable for MVP |
| **Role-based menu** — Menu items are filtered by `hasPerm`. Only `overview`, `members`, `families`, `visitors`, `benefactors`, `finance`, `transfers`, `ministries`, `reports` exist. No "Users/Access" management, no "Governance" or "Settings" sections. | Matches Fase 1 scope |

---

## 5. Layout & Design Assessment

### Current State
The layout is **functional but not "full-screen professional"** as requested:

- **Dashboard pages**: Almost all use `mx-auto max-w-6xl` (1280px max), which on a 4K monitor leaves ~40% of the screen empty. The user's requirement is "ocupando toda a tela."
- **Sidebar**: 256px (`w-64`) — reasonable.
- **Typography**: Uses system font (`font-sans`), sizes are reasonable (text-sm, text-xs for tables).
- **Lines/borders**: Uses `border-zinc-200 dark:border-zinc-800` consistently for cards and tables. Good definition.
- **Color palette**: Violet-to-fuchsia gradient for brand, status-based color tones for badges. Church-appropriate (not too corporate).
- **Dark mode**: Well-implemented with proper CSS variable switching.

### What's Missing for "Modern Professional Full-Screen"
1. **No `tailwind.config.js`** — can't configure containers, custom breakpoints, font families, or extend the color palette.
2. **`max-w-6xl` constraint** on all pages — should use `w-full max-w-none` or a configurable container.
3. **Font system**: No `@font-face` or Google Fonts import. The design is purely utility-based. For a church context, a warm, readable font (like `Inter` or `Poppins`) would feel more professional.
4. **Chart dark mode**: Recharts charts use hardcoded colors — dark mode doesn't apply proper colors.
5. **No CSS reset/normalize** beyond Tailwind's base — some browser inconsistencies may appear.

---

## 6. MVP Definition: Members + Finance (Entradas/Saídas)

### 6.1 What Already Exists (Shipping)

**Members Module:**
- ✅ List page: DataTable, search, status filter, role filter, stats cards, mobile cards
- ✅ Detail page: tabs (Dados, Contato, Vínculos, Espiritual, Documentos), edit via form, relationships
- ✅ Create/Edit: `MemberForm` / `PersonForm` with 16 fields, sections, masked inputs
- ✅ Carteirinha QR emission (`issueCard`)
- ✅ Árvore genealógica: `getMemberTree()` + relationships display (not a visual tree yet)
- ✅ Visitor → Member conversion

**Finance Module:**
- ✅ Lançamentos: list (DataTable), create (TransactionForm), detail (Drawer), filters, sorting
- ✅ Bar chart: Entradas × Saídas (mensal)
- ✅ Stat cards: Entradas, Saídas, Saldo
- ✅ Contas bancárias: list, create, toggle active
- ✅ Plano de contas: list, create categories
- ✅ Tipos de classificação: list, create
- ✅ Recibos: HTML rendering, email/WhatsApp send, delivery history
- ✅ Anexos: upload, list
- ✅ Doações recorrentes: schedule, toggle, auto-generate
- ✅ Relatórios: DRE, balancete mensal, CSV/PDF export
- ✅ Repasses entre filiais: list, create

### 6.2 MVP Gaps to Close

| # | Gap | Priority | Effort |
|---|---|---|---|
| G1 | Remove full-width constraint (`max-w-6xl` → `w-full`) on all dashboard pages | P1 | Small (search/replace across 7 page files) |
| G2 | Standardize on `DataTable` for all list pages (replace raw `<Table>` in Benefactors, Families, Ministries, Visitors) | P1 | Medium |
| G3 | Fix RG mask in member detail | P1 | Small (1-line fix) |
| G4 | Add server-side pagination to Members list (`listMembers` should support `page`/`page_size`) | P1 | Medium (backend + frontend) |
| G5 | Add "Últimos lançamentos" to dashboard overview (wire up real data) | P2 | Small |
| G6 | Fix chart dark-mode colors (use CSS variables instead of hardcoded hex) | P2 | Medium |
| 7 | Resolve the dead "People API" layer — delete unused `components/people/*` (entity-data-table, person-card, row-actions, journey-timeline, unified-drawer) + `swr-hooks.ts` + `listPeople`/`createPerson`/`updatePerson` in `lib/api.ts`. **Keep** `person-form.tsx` (used via `member-form.tsx`). | P0 | Large (decision required) |

### 6.3 Recommended MVP Scope (Members + Finance)

For a **basic, shippable MVP** focused on the user's request:

**Must-have (P0):**
1. Full-screen layout — remove `max-w-6xl` constraints, use `w-full`
2. Members list with proper DataTable (already done — standardize width)
3. Members detail with working edit form (fix RG mask bug)
4. Finance: transaction create/list with Entradas/Saídas/DRE (already done)
5. Finance: StatCard KPIs (already done)
6. Dark mode toggle (already done)

**Should-have (P1):**
1. Server-side pagination for Members (backend change)
2. Family detail page (uses existing `GET /families/{id}/members`)
3. Member financial history tab (uses existing `listTransactions`)
4. Dashboard "Últimos lançamentos" (real data)

**Could-have (P2):**
1. `tailwind.config.js` for design system customization
2. Custom Google Font (e.g., Inter)
3. Proper dark-mode chart colors
4. Member `[id]/tree` visual genealogy page
5. Visitor/benefactor detail pages

---

## 7. Implementation Plan (Task List)

### Phase A — Layout Foundation (Full-Screen, Typography, Design System)

**Task A1: Add `tailwind.config.js`**
- Create `apps/webadmin/tailwind.config.js` (ESM format for Tailwind v4 compat)
- Extend theme with church-appropriate font family (Inter), custom containers, and semantic colors
- Add `@import url(...)` in `globals.css` for Inter font
- Configure `maxWidth` for `7xl` (already in Tailwind by default) and a `screen-7xl` if needed

**Task A2: Remove full-width constraints**
- Replace `mx-auto max-w-6xl` → `mx-auto w-full max-w-none` (or just `w-full`) in:
  - `app/dashboard/members/page.tsx:370`
  - `app/dashboard/finance/page.tsx:314`
  - `app/dashboard/families/page.tsx:61`
  - `app/dashboard/visitors/page.tsx:103`
  - `app/dashboard/benefactors/page.tsx:47`
  - `app/dashboard/reports/page.tsx:78`
  - `app/dashboard/transfers/page.tsx:43`
  - `app/dashboard/ministries/page.tsx:138`
  - `app/dashboard/page.tsx:80`
- Replace `max-w-4xl` → `w-full max-w-none` in:
  - `app/dashboard/members/[id]/page.tsx:108,102`

**Task A3: Fix chart dark-mode colors**
- Replace hardcoded hex colors in Recharts components with CSS variable references or conditional dark-mode classes
- Apply to: `dashboard/page.tsx` (AreaChart), `dashboard/finance/page.tsx` (BarChart), `dashboard/reports/page.tsx` (AreaChart, BarChart)

**Task A4: Fix RG mask bug**
- `app/dashboard/members/[id]/page.tsx:163`: change `mask="phone" variant="rg"` → `mask="rg"`

### Phase B — Members Module MVP

**Task B1: Standardize Members list on DataTable**
- The members page already uses `DataTable` on desktop with custom `renderMobileCard`. Ensure the mobile card uses `DataTable`'s built-in `renderMobileCard` prop instead of the custom hidden-on-lg block, to eliminate duplication.

**Task B2: Add server-side pagination to Members API**
- Backend: extend `handleListMembers` to accept `page`, `page_size`, `sort`, `order` query params
- Frontend: `lib/api.ts` → `listMembers()` accepts `PeopleQuery`-like options; pass to DataTable server-side pagination
- Update `app/dashboard/members/page.tsx` to use server-side pagination

**Task B3: Add financial history tab to Member detail**
- New tab "Financeiro" on `/dashboard/members/[id]`
- Fetch `listTransactions()`, filter by `donor_member_id === id`
- Display DataTable: Data | Categoria | Valor | Tipo | Recibo

### Phase C — Finance Module MVP (Validation)

The Finance module is already the most complete. The user specifically mentions "entradas e saídas" (entries and exits). Validate:

**Task C1: Ensure balance/receipt flow works end-to-end**
- Transaction → auto-receipt → send via email/WhatsApp → delivery tracking
- Already implemented; verify TypeScript/lint passes

**Task C2: Ensure DRE shows Entradas/Saídas comparison**
- Already implemented in `dashboard/reports/page.tsx`
- Verify CSV/PDF export works

### Phase D — Dead Code Resolution

**Task D1: Decision & cleanup**
- **Option A (recommended for MVP):** Delete the unused dead-code layer: `components/people/entity-data-table.tsx`, `components/people/person-card.tsx`, `components/people/row-actions.tsx`, `components/people/journey-timeline.tsx`, `components/people/unified-drawer.tsx`, and `lib/swr-hooks.ts`. Also remove the `listPeople`, `createPerson`, `updatePerson` functions from `lib/api.ts` (they call `/api/v1/people` routes that don't exist in `router.go`).
- **Note:** `components/people/person-form.tsx` is **NOT dead** — it is used via `components/members/member-form.tsx` → `PersonForm` by the members and visitors pages. Keep it. Only the 5 files listed above are truly unused.
- **Option B:** Implement `/api/v1/people` backend route and migrate all pages to use the new components. (Full Phase 2 refactor — larger scope.)
- **Decision:** **Option A for the MVP.** Document Option B as a future refactor. The unused code creates confusion and false expectations.

### Phase E — Polish

**Task E1: Dashboard "Últimos lançamentos"**
- Wire up `listTransactions()` on the dashboard overview page, show last 5 transactions in a compact table

**Task E2: Add `tailwind.config.js`**
- Already in Task A1 — enables consistent design system

**Task E3: Font**
- Import Inter from Google Fonts (or local if offline) in `globals.css`

---

## 8. Validation Plan

| Step | Command | Expected |
|---|---|---|
| Lint/TS | `cd apps/webadmin && npx tsc --noEmit` | Zero errors |
| Build | `cd apps/webadmin && npm run build` | Compiles, no errors |
| Layout | Visually inspect 37" monitor — content should reach screen edges (minus sidebar) | max-w removed |
| RG mask | Edit member → RG field — typing `123456789` should produce `1.234.567-89` | Correct mask |
| RTL consistency | All list pages should use the same table pattern | Standardized |
| Dark mode | Toggle theme — charts should adapt colors | CSS variables in charts |

---

## 9. Risk Register

| Risk | Mitigation |
|---|---|
| Removing `max-w-6xl` causes tables to overflow on narrow screens | Use `overflow-x-auto` wrappers (already present on raw `<Table>`) |
| Deleting `components/people/*` removes future-ready code | The 5 dead files (`entity-data-table`, `person-card`, `row-actions`, `journey-timeline`, `unified-drawer`) reference a nonexistent `/api/v1/people` backend route — they cannot work. `person-form.tsx` is kept (it IS used via `member-form.tsx`). |
| Backend pagination changes break existing API contract | Add new query params as optional with backward-compatible defaults |
| Tailwind config breaks existing utility classes | Start minimal — only add `fontFamily` and `container` config; preserve all existing Tailwind utilities |
| Charts with CSS variables need careful color mapping | Use Tailwind's `currentColor` or CSS vars (`--tw-colors`) passed as props to Recharts |

---

## 10. Recommendations Summary

1. **Immediate (for the user's "basic MVP"):** Tasks A1-A4, B1, C1-C2, E1. These deliver a full-screen, professional, working members + finance module in <1 day of work.
2. **Short-term (next iteration):** Tasks B2, B3. Server-side pagination and member financial history.
3. **Future refactor (post-MVP):** Task D1-Option B — implement `/api/v1/people` backend route and migrate pages to the unified People components. This is a nice-to-have, not an MVP blocker.
4. **Do not touch:** The existing working pages are solid. Don't rewrite what works; enhance incrementally.
