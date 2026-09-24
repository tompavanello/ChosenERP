# Plano de Redesign UX — Módulo de Cadastro (Membros, Famílias, Visitantes, Benfeitores)

## Contexto Atual (do codebase)

### Estado atual de cada página

| Página | Linhas | Estado atual | Principais problemas |
|---|---|---|---|
| `/dashboard/members` | 485 | DataTable desktop + renderMobileCard duplicado | Filtragem client-side (LIMIT 100), duplicação mobile, sort fraco |
| `/dashboard/members/[id]` | 304 | Tabs com edição inline por aba | Editar em aba perde formulário (sem guard), RG usa `mask="phone"` (BUG: deveria ser `mask="rg"`), sem nome da filial |
| `/dashboard/families` | 134 | Tabela simples + 2 modais | Sem página de detalhe, lookup de `head_id` client-side |
| `/dashboard/visitors` | 219 | Table + Drawer + Modal de conversão | Sources hardcoded, stage avança só 1 passo (API aceita qualquer stage) |
| `/dashboard/benefactors` | 109 | Table simples + Drawer | Sem actions (só cria), sem paginação, sem histórico de doações |

### Padrões de design system atual

- **Tokens mortos**: `lib/tokens.ts` define 16 tons de brand, status tones, surface variants, spacing, typography, shadows, borderRadius, transitions, breakpoints, zIndex — com helper functions `getStatusClasses()`, `getPersonTypeClasses()`. **Nenhum componente usa.** O design real é Tailwind utility classes + CSS vars (`globals.css`).
- **CSS vars reais** (`globals.css`): `--ink`, `--paper`, `--card`, `--line`, `--muted`, `--brand`, `--brand-strong`, `--brand-soft`. Classes `.card`, `.btn-base`, `.btn-primary`, `.btn-ghost`, `.btn-outline`, `.input`, `.select`, `.textarea`, `.label`.
- **Componentes UI disponíveis**: Button (primary/ghost/outline), Card (Header/Title/Content/Body), Badge (10 tons × 3 variants), Table (simples), DataTable (sorting, pag, loading, empty, mobileCard), Modal (center), Drawer (right slide), Tabs (underline), Pagination, PageHeader, StatCard, Avatar (iniciais), Combobox, Skeleton/EmptyState, Field/Input/Select/Textarea/MaskedInput.

### Hooks SWR (não usados — com bug crítico)

`lib/swr-hooks.ts` define `useMembers`, `useVisitors`, `useFamilies`, `useBenefactors`, `usePeople`, `useCategories`, `useTransactions`, `useBalance`, `useBranches`, `useTransfers` e hooks de mutação (`useCreateMember`, `useUpdateMember`, etc.) — **não importados em nenhuma página**. Todas usam `useEffect` + estados locais.

**BUG CRÍTICO no `mutate` (linhas 137-140):**
```ts
function mutate(key: string) {
  const evt = new Event(`swr-mutate:${key}`);
  window.dispatchEvent(evt);
}
```
SWR's `useSWR` **não escuta** eventos `window` customizados. Isso não invalida o cache. Correção:
```ts
import { mutate } from "swr";
mutate(key); // invalida e refetcha automaticamente
```

### API verificada (`lib/api.ts`)

- **`listPeople(params: PeopleQuery)`**: Server-side paginated + filterable. Query suporta: `type` ("member"|"visitor"|"benefactor"|"all"), `status`, `q`, `age_min/max`, `gender`, `marital_status`, `branch_id`, `page`, `page_size`, `sort`, `order`. Retorna `{ people: Person[], total, page, page_size }`. **Fonte ideal para listagens** de membros/visitantes/benfeitores.
- **`Person` type** unificado: `id`, `type`, `full_name`, `first_name`, `last_name`, `email`, `phone`, `whatsapp`, `cpf`, `rg`, `birth_date`, `age`, `gender`, `marital_status`, `membership_status`, `office`, `profession`, `baptism_*`, `source`, `journey_stage`, `notes`, `branch_name`, `family_name`, `tags`, `created_at`, `updated_at`.
- **`createPerson(type, data)`** / **`updatePerson(type, id, data)`**: CRUD unificado via `/api/v1/people/{type}` e `/api/v1/people/{type}/{id}`.
- **`getMember(id)`**: funciona para membros. **NÃO existe** `getVisitor(id)` / `getBenefactor(id)` — detail pages usam `listPeople({ type: "..." })` + find by ID, ou backend novo endpoint.
- **`getMemberTree(id)`** → `/api/v1/members/{id}/tree` — existe para árvore genealógica.
- **`listFamilies()`** → `/api/v1/families` — retorna `Family[]` com `member_count`, `head_id`, `branch_id`. Não expõe lista de membros.
- **`convertVisitorToMember(visitorId, data)`** → `/api/v1/visitors/{id}/convert` POST.
- **`revertVisitorConversion(visitorId)`** → `/api/v1/visitors/{id}/revert` POST — **existe**. Visitors podem reverter.
- **`updateVisitorStage(id, stage)`** → `/api/v1/visitors/{id}/stage` PATCH — aceita **qualquer stage string** (dropdown direto, não só "avançar um passo").
- **`listTransactions(type)`** → `/api/v1/finance/transactions?type=...`. Retorna todas as transações. Filtrar client-side por `benefactor_id` para histórico de doações.
- **`addRelationship(memberId, relateMemberId, kind)`** → `/api/v1/members/{id}/relationships` POST. Não há DELETE de relacionamentos.
- **`addFamilyMember(familyId, memberId, relateId, relation)`** → POST. Não há remover membro da família.
- **Nenhum DELETE** em nenhuma API — nem membros, famílias, visitantes, benfeitores ou relacionamentos podem ser excluídos.

### Gaps críticos de dados

1. **Família detalhe**: `Family` tem `member_count` e `head_id`, mas não expõe lista de membros. `Person` não tem `family_id`. Para listar membros da família, é preciso **novo endpoint backend** (`GET /families/{id}/members`) — fora escopo frontend. Solução interina: usar `listPeople({ type: "member", q: familyName })` para find de membros pela família_name, ou carregar todos os membros e filtrar client-side por `family_name` no `Person` result.
2. **`head_id` resolução**: `listFamilies` retorna `head_id` (uuid). Para resolver o nome, carrega cache de membros (`useMembers` ou `listPeople({ type: "member" })`) e faz lookup client-side.
3. **Benefactor transaction history**: filter client-side de `listTransactions()` por `benefactor_id`.
4. **`MaskedInput`** suporta `mask="rg"` (input.tsx:44,70-74). Bug é só no uso: `member-detail.tsx:163` passa `mask="phone"` ao invés de `mask="rg"`. O prop `variant="rg"` é **ignorado** (detruit no componente mas nunca usado).
5. **`tokens.ts`** não é usado — mas existe. Opções: (a) migrar componentes para usar tokens, (b) deletar. Decisão: **deletar** (`tokens.ts`) e padronizar em Tailwind `dark:` variants consistentes. Menor risco, e evita confusão.

---

## Princípios do Redesign

1. **Consistência de padrões** — Unificar table vs DataTable; modal vs drawer; patterns de action.
2. **Mobile-first responsivo sem duplicação** — DataTable adaptável em vez de renderMobileCard manual.
3. **Server-side quando possível** — Usar `listPeople` (paginada) + SWR cache; mutações invalidam via `mutate()`.
4. **Ações contextuais** — Dropdown (3-dot) de ações por linha ao invés de botões espalhados.
5. **Formulários unificados** — `PersonForm` com seções condicionais por `entityType`.
6. **Cross-entity linking** — Conversão visitante→membro, vínculo familiar, ligação a benfeitor existente.
7. **Visualização rica** — Cards de perfil, timeline de jornada, árvore genealógica, histórico financeiro.

---

## Arquitetura Proposta

### Novo componente: `EntityDataTable`

Wrapper sobre `DataTable` que unifica acesso e padrões visuais:
- **Coluna de ações** padrão (dropdown "⋯" ou botões inline em mobile).
- **Empty state** integrado com call-to-action ("+ Novo").
- **Search + filter bar** integrada no header do card.
- **Stat cards** integrados (via prop opcional).
- **Mobile**: usa `renderMobileCard` do `DataTable` existente — cards verticais empilhados, **sem duplicação de código**.
- **Server-side pagination** via `page`, `page_size`, `total` props.

```
<EntityDataTable<Person>
  columns={columns}
  data={people}
  total={total}
  page={page}
  pageSize={pageSize}
  onPageChange={setPage}
  onSearch={setQuery}
  filters={<StatusFilter value={status} onChange={setStatus} />}
  statCards={[...]}
  mobileCard={(row) => <PersonCard person={row} />}
  rowActions={(row) => <RowActions />}
  searchPlaceholder="Buscar por nome, e-mail, CPF..."
  emptyTitle="Nenhum resultado"
  emptyDescription="..."
  emptyAction={<Button>Novo</Button>}
/>
```

### Novo padrão: `UnifiedEntityState` (state discriminado)

Um único state para todos os modais/drawers de uma página:
```ts
type EntityModal =
  | { kind: "create"; entityType: "member" | "visitor" | "benefactor" }
  | { kind: "edit"; entityType: "member" | "visitor" | "benefactor"; id: string; data: Person }
  | { kind: "convert-visitor"; visitor: Visitor }
  | { kind: "link-family"; family: Family }
  | { kind: "add-relationship"; member: Member }
  | null;
```

Implementado como hook: `const [modal, setModal] = useState<EntityModal>(null)`. Renderiza `<UnifiedDrawer modal={modal} onClose={() => setModal(null)} />`.

### Novo componente: `PersonForm`

Unifica `MemberForm`, forms de visitor e benefactor. Base de campos comuns + sections condicionais:

Props: `entityType: "member" | "visitor" | "benefactor"`, `initial?`, `visitor?`, `onSubmit`, `onCancel`, `saving`.

Sections:
1. **Identificação** — nome (first/last), apelido, nascimento, sexo, estado civil
2. **Documentos** — CPF (`mask="cpf"`), RG (`mask="rg"` — **BUG FIX**), profissão (members only)
3. **Contato** — email (full width), telefone, whatsapp
4. **Vida eclesiástica** — visível só se `entityType === "member"`: status (Combobox), cargo (Combobox), batismo (data + local), membro desde

```ts
// Reusable para visitor conversion, benefactor edit, member create/edit
<PersonForm
  entityType={type}
  initial={person}
  visitor={visitor ?? null}  // pré-preenche se convertendo de visitante
  onSubmit={async (data) => {
    await createPerson(type, data);    // API unificada
    mutate("people");                 // SWR invalidate (FIXED)
    setModal(null);
  }}
  onCancel={() => setModal(null)}
  saving={isSaving}
/>
```

### Novo componente: `JourneyTimeline`

Timeline horizontal para visitantes: dots clicáveis + label + cor por stage. Cada etapa clicável para saltar. Botões "Avançar" / "Retroceder" / "Converter".

### Novo componente: `PersonCard` (mobile)

Card vertical: avatar + nome + subtitle (cargo/filial/status) + actions icon row.

---

## Plano por Módulo

### 1. Membros

#### 1.1 `/dashboard/members` — Lista revisada

```
PageHeader + StatCards (Total, Ativos, Inativos, Visitantes)
EntityDataTable<Person>
  - Colunas: Membro (avatar + nome + cargo + filial), Contato (email/phone),
            Status (Badge), Última doação, Ações (3-dot dropdown)
  - Filtros: busca global + select status + select filial
  - Server-side pagination via listPeople({ type: "member" })
```

**Mudanças:**
- Usar `listPeople({ type: "member", page, page_size, q, status, branch_id, sort, order })` (server-side).
- `EntityDataTable` elimina `renderMobileCard` duplicado.
- Sort: numérico para idade/cpf, string para nomes (DataTable já suporta custom sort via `sortColumn`).
- Filtro por filial via `Person.branch_name` (API suporta `branch_id`).
- `Última doação`: merge de `listTransactions()` cache filtrado por `donor_member_id === person.id`.
- **Actions dropdown** por linha: Ver (→ detail), Editar (→ Drawer), Carteirinha (QR), Vínculos (→ tree), Mais (⋯) para ações secundárias.
- **Mutação via SWR**: `useCreateMember` / `useUpdateMember` invalidam `"people"` cache com `mutate()` (bug do `dispatchEvent` consertado).

#### 1.2 `/dashboard/members/[id]` — Perfil revisado

```
Header (voltar + avatar + nome + status badge + actions: Editar, Carteirinha, Mais⋯)
Tabs (Desktop) / Segmented (Mobile):
  1. Dados Pessoais    (read-only dl-grid)
  2. Vínculos & Família (relationships list + add button)
  3. Espiritual         (batismo, cargo, membro desde)
  4. Documentos         (carteirinha + QR + history)
  5. Histórico Financeiro (transactions da pessoa)
```

**Mudanças:**
- **Editar via Drawer global** — botão "Editar" abre Drawer com `PersonForm entityType="member"`. Não perde estado entre tabs. Ao salvar: `updatePerson("member", id, data)` + `mutate("people")` + `mutate("member:" + id)` + fecha Drawer.
- **RG mask bug FIX**: `mask="rg"` (não `mask="phone" variant="rg"`).
- **Filiação**: mostra `member.branch_name` (vem de `Person` ou `getMember`).
- **Vínculos**: lista `getMemberTree(id).relationships` com badges de tipo + botão remover (se API permitir; se não, botão "reportar" desabilitado). Botão "Adicionar" abre Drawer com Combobox de membro + select de kind.
- **Nova aba "Histórico Financeiro"**: `listTransactions()` filtrado por `donor_member_id`, DataTable com Data | Categoria | Valor | Tipo | Receita.
- **Documentos**: lista de documentos emitidos (query `/api/v1/documents/by-token/{token}` ou lista de `Person` docs). Status badge + botão de reemissão.

#### 1.3 `/dashboard/members/[id]/tree` — Árvore genealógica

```
Visualização de árvore vertical:
  [Membro (root)]
  ├── Cônjuge(s)   → cards clicáveis (link para perfil)
  ├── Filhos(as)   → cards clicáveis
  └── Discípulos   → cards clicáveis
Expandir/collapse cada nó. Badge de tipo de relação.
```

Tecnologia: component recursivo HTML/CSS (nested divs com border-left). Usa `getMemberTree()` existente.

---

### 2. Famílias

#### 2.1 `/dashboard/families` — Lista revisada

```
PageHeader + StatCards (Total, Membros agrupados, Famílias com responsável)
EntityDataTable<Family>
  - Colunas: Família | Responsável (resolved name) | Membros (badge count) | Filial | Criado | Ações
  - Ações dropdown: Ver detalhe, Vincular membro, Editar nome
```

**Mudanças:**
- `head_id` resolvido via cache de `listPeople({ type: "member" })` — lookup `people.find(p => p.id === family.head_id)`.
- Coluna "Filial" via `family.branch_id` + `useBranches` ou cache.
- Pagination server-side (se API suportar; se não, client-side via `listFamilies` + SWR).
- Actions dropdown.

#### 2.2 `/dashboard/families/[id]` — Detalhe (NOVA PÁGINA)

```
Header (voltar + nome da família + avatar do head + actions: Editar, Ver árvore)
Tabs:
  1. Membros (DataTable: avatar + nome + vínculo badge + status)
  2. Endereço (jsonb formatted)
  3. Histórico Financeiro (transações da família — filter by family_id ou member_ids)
```

**Ações:**
- "Vincular membro" → Drawer: `PersonForm` filtrado para membros não vinculados (ou `addFamilyMember`).
- "Editar endereço" → Drawer: form de endereço (street, number, complement, neighborhood, city, state, zip).
- "Ver árvore" → navega para `/members/{head_id}/tree`.

**Backend gap**: Sem `GET /families/{id}/members`, usar `listPeople({ type: "member", q: family.name })` + filter por `family_name === family.name`. Marcar como workaround + ticket backend.

---

### 3. Visitantes

#### 3.1 `/dashboard/visitors` — Lista revisada

```
PageHeader + StatCards (Total, Em trilha, Convertidos)
Tabs de filtro: Todos | Em acolhimento | Convertidos
EntityDataTable<Person>
  - Colunas: Visitante (avatar + nome + origem) | Contato | Trilha (JourneyTimeline compacto) | Entrada | Ações
  - Server-side pagination via listPeople({ type: "visitor" })
```

**Mudanças:**
- **Sources**: migrar do hardcode frontend (`SOURCES = [...]`) para constante em `lib/constants.ts` (`VISITOR_SOURCES` com label + value). Não há API backend para listar sources.
- **Stage como dropdown**: `updateVisitorStage(id, anyStage)` — usar Combobox com todas as etapas, não só "avançar". Permite avançar, retroceder, ou saltar.
- **Revert conversion**: botão "Desfazer conversão" → `revertVisitorConversion(id)` (existe na API).
- **Conversão para membro**: `PersonForm entityType="member" visitor={visitor}` pré-preenchido + `convertVisitorToMember`.
- **Actions**: Mais (⋯) → Avançar etapa, Converter, Editar, Desfazer (se convertido).

#### 3.2 `/dashboard/visitors/[id]` — Detalhe de visitante (NOVA PÁGINA)

```
Header + Tabs:
  1. Dados Pessoais (read-only)
  2. Jornada (timeline interativa JourneyTimeline + botões Avançar/Retroceder/Converter)
  3. Contato (editável inline via Drawer)
```

**JourneyTimeline interativa:**
```
[ welcome ] → [ coffee_pastor ] → [ course ] → [ cell ] → [ converted ]
   ✓            ↻              ✓            ↻            ✓
```
Cada etapa clicável para saltar. Botões de navegação. Botão "Converter para membro" (abre Drawer com PersonForm).

---

### 4. Benfeitores

#### 4.1 `/dashboard/benefactors` — Lista revisada

```
PageHeader + StatCards (Total, Com e-mail, Com telefone)
EntityDataTable<Person>
  - Colunas: Benfeitor (avatar + nome) | Contato | Observações | Total doado | Cadastrado | Ações
  - Server-side pagination via listPeople({ type: "benefactor" })
```

**Mudanças:**
- **Coluna "Total doado"**: merge de `listTransactions()` cache, filtrado por `benefactor_id === person.id`, soma de `amount`.
- **Actions**: Ver perfil, Editar (Drawer), Histórico de doações.
- **Pagination + server-side**.

#### 4.2 `/dashboard/benefactors/[id]` — Detalhe (NOVA PÁGINA)

```
Header + Tabs:
  1. Dados Pessoais (read-only + Editar Drawer)
  2. Histórico de Doações (DataTable: data | descrição | categoria | valor | status)
  3. Anotações (notes textarea editável)
```

**Backend gap**: Sem `GET /benefactors/{id}`, usar `listPeople({ type: "benefactor", q: id })` ou `listPeople({ type: "benefactor" })` + find by ID. Marcar workaround.

---

### 5. Componentes compartilhados

#### 5.1 `EntityDataTable`
Props: `columns`, `data`, `total`, `page`, `pageSize`, `onPageChange`, `onSearch`, `filters?`, `statCards?`, `mobileCard?`, `rowActions?`, `searchPlaceholder?`, `emptyTitle?`, `emptyDescription?`, `emptyAction?`.

#### 5.2 `PersonCard` (mobile)
Card vertical: avatar + nome + subtitle + status badge + actions icon row.

#### 5.3 `PersonForm` (unificado)
Props: `entityType`, `initial?`, `visitor?`, `onSubmit`, `onCancel`, `saving`. Sections condicionais. **Bug RG fix incluído.**

#### 5.4 `UnifiedDrawer`
Gerenciado pelo `EntityModal` state. Renderiza `PersonForm`, `PersonForm`-para-conversão, ou form de família dependendo do `kind`.

#### 5.5 `JourneyTimeline`
Horizontal timeline com dots clicáveis + labels + cor por stage. Modo compacto (para DataTable cell) e modo full (para página de detalhe).

#### 5.6 `RowActions` (dropdown)
3-dot dropdown: Ver, Editar, + entity-specific (Carteirinha, Converter, Histórico...). Usa `MoreHorizontal` icon do lucide.

---

## Plano de Implementação (Task List)

### Fase 1 — Infra + componentes base (4 tasks)
1. **Fix SWR `mutate`** (`swr-hooks.ts`): importar `mutate` do `"swr"`, remover `window.dispatchEvent` pattern. Exportar `mutate` para uso direto.
2. **Deletar `lib/tokens.ts`** — tokens mortos. Padronizar componentes em Tailwind `dark:` variants.
3. Criar `PersonForm` (unificado) — com `entityType` prop, RG mask correta, sections condicionais.
4. Criar `EntityDataTable` wrapper sobre `DataTable` — com pagination server-side, search, filters, statCards, mobileCard, rowActions props.
5. Criar `JourneyTimeline` + `PersonCard` + `RowActions` components.

### Fase 2 — Membros (3 tasks)
6. Redesign `/dashboard/members` — SWR `usePeople({ type: "member" })`, `EntityDataTable`, actions dropdown, filtro de filial, coluna última doação.
7. Redesign `/dashboard/members/[id]` — Drawer edit (global, não inline por aba), RG fix, aba Histórico Financeiro, vínculos listáveis, nova aba Docs.
8. Criar `/dashboard/members/[id]/tree` — árvore genealógica recursiva.

### Fase 3 — Famílias (2 tasks)
9. Redesign `/dashboard/families` — `EntityDataTable`, head_name lookup via cache, coluna filial.
10. Criar `/dashboard/families/[id]` — detalhe com tabs Membros/Endereço/Histórico. (Workaround para backend gap no member listing.)

### Fase 4 — Visitantes (2 tasks)
11. Redesign `/dashboard/visitors` — `EntityDataTable`, `JourneyTimeline` compacto, dropdown de stage (avançar/retroceder/saltar), revert conversion, sources constante.
12. Criar `/dashboard/visitors/[id]` — timeline interativa + detalhes + converter.

### Fase 5 — Benfeitores (2 tasks)
13. Redesign `/dashboard/benefactors` — `EntityDataTable`, coluna total doado (cache de transações), actions dropdown, edit Drawer.
14. Criar `/dashboard/benefactors/[id]` — detalhe com tabs Dados/Histórico de Doações/Anotações.

### Fase 6 — Integração cross-entity (1 task)
15. `UnifiedDrawer` + `EntityModal` state — unificar todos os modais/drawers. Cross-linking: visitante→membro (PersonForm pré-preenchido), vínculo familiar, benfeitor↔membro.

### Fase 7 — Validação (3 tasks)
16. `go build ./... && go vet ./...` (build Go).
17. `npm run lint && npx tsc --noEmit` (webadmin).
18. Testes manuais: desktop/mobile, dark mode, RG mask, conversão visitante→membro, árvore genealógica, histórico doador.

---

## Riscos e Considerações

### Riscos críticos
1. **SWR `mutate` quebrado**: o `window.dispatchEvent` não funciona. **Corrigir antes de qualquer página usar SWR hooks.** Tasks 6-14 dependem disso.
2. **API não expõe `GET /families/{id}/members`** — family detail page precisa client-side workaround (filter by `family_name`). Marcar como ticket backend.
3. **API não expõe `GET /benefactors/{id}`** — benefactor detail usa `listPeople` + find by ID. Marcar como ticket backend.
4. **`tokens.ts` deletado** — nenhum componente usa, mas se houver código referenciando, precisa remover imports. Verificar antes de deletar.
5. **Não há DELETE** — nenhum módulo permite exclusão. Fora de escopo.
6. **Nenhuma API para visitor sources** — sources continuam hardcode em `constants.ts`.
7. **`Person` não tem `family_id`** — não dá pra filtrar membros por família via API. Workaround: `family_name` match.

### Decisões de escopo
- **Mobile cards**: `EntityDataTable` usa `renderMobileCard` configurável do `DataTable` existente — elimina duplicação.
- **Server-side pagination**: `listPeople` suporta `page`/`page_size`. Implementar proper pagination.
- **Benefactor donation history**: filter client-side de `listTransactions()` por `benefactor_id`.
- **Dark mode**: padronizar em Tailwind `dark:` variants. Deletar `tokens.ts`.

---

## Validação

1. **Build**: `npm run build` (Next.js) e `go build ./...`.
2. **Lint/Typecheck**: `npm run lint && npx tsc --noEmit`.
3. **Manual**:
   - Navegar membros (desktop + mobile) — cards não duplicam.
   - Editar membro — Drawer abre/fecha sem perder estado entre tabs.
   - RG mask — aplica máscara de RG (`1.234.567-89`), não telefone.
   - Conversão visitante → membro — `PersonForm` pré-preenchido, cria membro + linka `converted_to_member_id`.
   - Reverter conversão visitante — botão funciona.
   - Árvore genealógica — navegação entre nós, expandir/collapse.
   - Benefactor detail — histórico de doações visível.
   - Filtrar membros por filial — dropdown funciona.
   - Stage dropdown de visitantes — avançar/retroceder/saltar.
4. **Dark mode**: alternar e verificar consistência em todos os novos componentes.

---

## Perguntas abertas (para time backend, se necessário)

1. A API suporta `GET /families/{id}/members`? (afeta family detail page — workaround client-side implementado)
2. A API suporta `GET /benefactors/{id}`? (afeta benefactor detail — workaround via `listPeople`)
3. DELETE em membros/visitantes/benfeitores/benefactor? (afeta ações — fora escopo, não implementado)
4. A API expõe `GET /api/v1/visitors/sources`? (sources continuam constantes em frontend)

> **Nota**: Todos os workarounds client-side estão implementados no plano. Perguntas 1 e 2 são otimizações de backend, não blockers para frontend.