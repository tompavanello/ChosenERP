# Plano - Identidade global, multi-igreja e subdominio (white-label)

Status: **implementado** - o numero de migracao final foi `000053_identity_memberships`
(e `000054_tenant_branding` para o white-label), pois `000052` ja existia. O plano
abaixo e mantido como referencia de arquitetura.
Escopo: separar **identidade** (pessoa) de **vinculo com a igreja** (membership),
permitir que **um mesmo e-mail acesse varias igrejas** e habilitar
**subdominio por igreja** com branding (central `app.<dominio>` + `*.<dominio>`).

## Decisoes ja tomadas (com o cliente)

1. **Senha unica** por identidade (modelo Slack/Notion): um login, N igrejas.
2. **Identidade global** + **memberships N:N** (`user_id  tenant_id`).
3. **Subdominio por igreja** (`igreja.dominio`) + **dominio central** (`app.dominio`).
4. Infra: **Cloudflare** com dominio proprio (DNS/TLS wildcard).
5. O **slug** da igreja e identificador publico (URL/branding) - nunca credencial.
6. Seguranca segue em **MFA (TOTP ja existe)**, rate limiting e SSO futuro.

---

## 1. Estado atual (pontos que serao tocados)

| Area | Arquivo | Observacao |
|---|---|---|
| Usuarios | `db/migrations/000003_users_rbac_audit.up.sql` | `users.email UNIQUE` **global**, mas `tenant_id/branch_id/role_id` no proprio usuario |
| Login (SECURITY DEFINER) | `db/migrations/000010_auth_functions.up.sql` | `auth_lookup_user(email)` resolve tenant/branch/role |
| Auth service | `internal/auth/service.go` | `Login`, `Me`, `Refresh`, MFA, senha |
| Claims | `internal/auth/jwt.go` | `uid`, `tid`, `bid`, `role`, `typ` |
| Bounds/RLS GUCs | `internal/store/db.go` | `app.tenant_id/branch_id/role/branch_scope` |
| Repo de usuarios | `internal/users/users.go` | CRUD por tenant (role_id/branch_id no usuario) |
| Middleware | `internal/httpapi/middleware.go` | Bearer + troca de filial via `X-Branch-Id` |
| Multi-tenant RLS | `000016`, `000036`, `000037`, `000045` | tenant + branch; Sede grava no tenant |
| Branding | - | **nao existe** (tenants so tem `name/slug/legal_name/cnpj/plan/locale/timezone`) |

Referencias de FK por `users.id` (nao mudam): `audit_log.actor_id`,
`documents.created_by`, `member_history.created_by`,
`financial_transactions.voided_by`, `recurring_donations`, etc.

---

## 2. Arquitetura alvo

```
Pessoa (users, GLOBAL)
    memberships (user_id, tenant_id, role_id, branch_id)   <- N:N
                                   
Tenant ativo (no JWT: tid/bid/role)  resolve do subdominio ou do seletor
```

- `users`: identidade (`email`, `password_hash`, `full_name`, `mfa_*`, `is_active`).
- `memberships`: papel/filial de uma pessoa dentro de uma igreja.
- `tenants`: branding (`slug` ja existe + `logo_url`, `brand_color`, `favicon_url`).
- JWT continua carregando **tenant ativo** (`tid`) - o RLS ja filtra por `app.tenant_id`.
- Resolucao do tenant ativo:
  1. **Subdominio** (`igreja.dominio`) -> tenant do slug, se houver membership ativa.
  2. **Central** (`app.dominio`) -> 1 membership entra direto; >1 abre **seletor de igreja**.

---

## 3. Fase 1 - Identidade + membership (independe de infra)

### 3.1 Migracao `000051_identity_memberships`

```sql
-- 1) Vinculo pessoa  igreja
CREATE TABLE memberships (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    uuid NOT NULL REFERENCES users(id)    ON DELETE CASCADE,
    tenant_id  uuid NOT NULL REFERENCES tenants(id)  ON DELETE CASCADE,
    role_id    uuid NOT NULL REFERENCES roles(id),
    branch_id  uuid REFERENCES branches(id) ON DELETE SET NULL, -- NULL => Sede
    is_active  boolean NOT NULL DEFAULT true,
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (user_id, tenant_id)
);
CREATE INDEX idx_memberships_user   ON memberships(user_id);
CREATE INDEX idx_memberships_tenant ON memberships(tenant_id, branch_id);

-- 2) Backfill: cada usuario atual vira 1 membership
INSERT INTO memberships (user_id, tenant_id, role_id, branch_id, is_active)
SELECT id, tenant_id, role_id, branch_id, is_active FROM users
ON CONFLICT (user_id, tenant_id) DO NOTHING;

-- 3) users passa a ser global
ALTER TABLE users
    DROP COLUMN tenant_id,
    DROP COLUMN branch_id,
    DROP COLUMN role_id;
-- (email UNIQUE global permanece)
```

> Observacao: a migracao e destrutiva das colunas; rodar em transacao (o runner ja
> faz) e validar o backfill antes do DROP. Alternativa segura: um release com as
> colunas antigas desativadas e um release seguinte removendo (ver 9).

### 3.2 GUC de identidade e helpers

```sql
CREATE OR REPLACE FUNCTION current_user_id() RETURNS uuid
LANGUAGE sql STABLE AS $$
    SELECT NULLIF(current_setting('app.user_id', true), '')::uuid
$$;
```

- `store.Bounds` ganha `UserID string`.
- `store.WithTenant` seta `app.user_id` (novo `set_config`).
- `store.WithSystem` seta `app.user_id = ''`.
- `boundsFromClaims` preenche `UserID: c.UserID`.

### 3.3 RLS

`memberships`:
```sql
ALTER TABLE memberships ENABLE ROW LEVEL SECURITY;
-- self ve todas as suas igrejas (para o seletor); senao, escopo do tenant
CREATE POLICY memberships_sel ON memberships FOR SELECT
  USING (is_system() OR user_id = current_user_id()
         OR rls_read(tenant_id, branch_id, false));
CREATE POLICY memberships_ins ON memberships FOR INSERT
  WITH CHECK (rls_write(tenant_id, branch_id, false));
CREATE POLICY memberships_upd ON memberships FOR UPDATE
  USING (rls_write(tenant_id, branch_id, false))
  WITH CHECK (rls_write(tenant_id, branch_id, false));
CREATE POLICY memberships_del ON memberships FOR DELETE
  USING (rls_write(tenant_id, branch_id, false));
```

`users` (global):
```sql
DROP POLICY IF EXISTS users_sel ON users;
CREATE POLICY users_sel ON users FOR SELECT USING (
    is_system() OR id = current_user_id() OR
    EXISTS (SELECT 1 FROM memberships me
            WHERE me.user_id = users.id AND me.tenant_id = current_tenant())
);
DROP POLICY IF EXISTS users_all ON users;
-- UPDATE: self (perfil/MFA/senha) ou quem e membro do tenant (autorizacao no handler)
CREATE POLICY users_upd ON users FOR UPDATE USING (
    is_system() OR id = current_user_id() OR
    EXISTS (SELECT 1 FROM memberships me
            WHERE me.user_id = users.id AND me.tenant_id = current_tenant())
) WITH CHECK (
    is_system() OR id = current_user_id() OR
    EXISTS (SELECT 1 FROM memberships me
            WHERE me.user_id = users.id AND me.tenant_id = current_tenant())
);
-- INSERT via funcao SECURITY DEFINER (ver 3.5) para anexar identidade existente
```

### 3.4 Auth (login / me / refresh / switch)

- **`auth_lookup_user(email)`** passa a devolver a **identidade** (sem tenant):
  `user_id, full_name, email, password_hash, mfa_enabled, mfa_secret, is_active`.
- Nova **`auth_memberships(user_id)`** (SECURITY DEFINER): lista
  `tenant_id, tenant_name, tenant_slug, role_key, branch_id, is_active`.

**Login (`Login(email, password, code, tenantSlug?)`)**
1. Valida identidade + senha (+ MFA).
2. Carrega memberships ativas.
3. Resolve tenant ativo:
   - `tenantSlug` informado (subdominio/central): precisa de membership ativa nele
     -> se nao houver, `ErrTenantForbidden` (redireciona ao central).
   - sem slug: **1** membership -> entra; **>1** -> retorna
     `requires_tenant_selection` + `selection_token` (JWT `typ='select'`, 5 min) + lista.
4. Emite access/refresh com `tid/bid/role` do membership (como hoje).

**`POST /auth/select-tenant`** `{ selection_token, tenant_id }`
-> valida o token `select` e a membership -> emite tokens.

**`POST /auth/switch-tenant`** (autenticado) `{ tenant_id }`
-> valida membership ativa -> novo par de tokens.

**`GET /me`** passa a devolver tambem `memberships: [{tenant_id, tenant_name,
tenant_slug, role, branch_id, is_active}]` e o tenant ativo.

`Refresh` mantem o tenant ativo ja embutido no refresh token.

### 3.5 Repo/administracao de usuarios

- `users.Create` (admin) -> **funcao SECURITY DEFINER** `user_attach_to_tenant(email,
  password, full_name, tenant_id, role_key, branch_id)`:
  - se o e-mail nao existe -> cria identidade + membership;
  - se existe -> **anexa** membership (nao altera a senha existente) - usado no
    "convidar usuario ja existente".
- `users.List/Get` passam a projetar por **membership** (join `memberships`).
- `users.Update` separa:
  - campos da **identidade** (`full_name`, `is_active`);
  - campos do **membership** (`role_id`, `branch_id`).
- `ResetPassword` continua por identidade (por `users.id`).
- `ListRoles` conta usuarios via `memberships` no tenant.

### 3.6 Middleware (tenant por requisicao)

- Estender o padrao do `X-Branch-Id`:
  - `X-Tenant-Id` (uuid) **ou** `X-Tenant-Slug` (derivado do Host no frontend).
  - So para quem tem a membership; valida `EXISTS(membership ativa)` e sobrescreve
    `claims.TenantID/branch/role` do membership correspondente.
- No login, o frontend envia o **slug do Host** (campo no corpo), nao header.

### 3.7 Frontend - seletor de igreja

- `lib/api.ts`: `login` aceita `tenant_slug`; guarda `memberships` no `User`;
  funcoes `selectTenant`/`switchTenant`; contexto de tenant em `localStorage`
  (mesmo padrao do `chosen_branch`).
- Tela de **selecionar igreja** quando `requires_tenant_selection`.
- **Switcher de igreja** na topbar (ao lado do switcher de filial), visivel quando
  o usuario tem >1 membership.

### 3.8 Contratos de API (Fase 1)

`POST /api/v1/auth/login`
```jsonc
// request
{ "email": "a@b.c", "password": "...", "code": "123456", "tenant_slug": "matriz" }
// 200 (entrou direto)
{ "tokens": { ... }, "user": { "id","email","full_name","tenant_id","branch_id","role","permissions":[...],
                             "memberships":[{"tenant_id","tenant_name","tenant_slug","role"}] } }
// 200 (precisa escolher)
{ "requires_tenant_selection": true,
  "selection_token": "...",
  "tenants": [{ "id","name","slug","role" }] }
```

`POST /api/v1/auth/select-tenant`
```jsonc
{ "selection_token": "...", "tenant_id": "uuid" }   // -> { tokens, user }
```

`POST /api/v1/auth/switch-tenant`
```jsonc
{ "tenant_id": "uuid" }                            // -> { tokens, user }
```

`GET /api/v1/me`
```jsonc
{ "id","email","full_name","tenant_id","branch_id","role","permissions":[...],
  "memberships":[{"tenant_id","tenant_name","tenant_slug","role","branch_id","is_active"}] }
```

---

## 4. Fase 2 - Subdominio + white-label

### 4.1 Dados
```sql
ALTER TABLE tenants
  ADD COLUMN logo_url      text,
  ADD COLUMN brand_color   text,
  ADD COLUMN favicon_url   text,
  ADD COLUMN custom_domain text;      -- opcional (dominio proprio da igreja)
```
(Alternativa: tabela `tenant_branding`, se quiser versionar assets.)

### 4.2 API publica (sem auth)
- `GET /api/v1/public/tenant/{slug}` -> `{ name, slug, logo_url, brand_color, favicon_url }`.
- Usada na **tela de login** (branding + validacao de que a igreja existe).
- Cuidado com enumeracao: resposta generica; nao listar igrejas.

### 4.3 Frontend
- Detectar o slug por `window.location.host` (subdominio) -> buscar branding.
- Aplicar nome/logo/cores no login e no shell; `tenant_slug` no corpo do login.
- No central, sem slug -> fluxo normal (seletor).

### 4.4 Infra
- **Cloudflare**: registro **wildcard** `*` -> tunel origem; certificado de borda
  wildcard (`*.dominio`) cobrindo os subdominios; `app` para o central.
- **nginx** (`infra/nginx`): `server_name ~^(?<slug>[a-z0-9-]+)\.dominio$ app.dominio;`
  repassando o `Host`; roteia `/api/*` para a API e o resto para o webadmin (como hoje).
- **CORS**: como front e API ficam na **mesma origem** por host, o CORS relativo
  continua valendo; se algum host cruzar chamadas, liberar `https://*.dominio`.
- Onboarding cria o `slug`; o subdominio nasce automaticamente pelo wildcard.

---

## 5. Fase 3 - Onboarding e gestao

- Tela "Configuracoes": branding (logo/cor/favicon), `slug` (readonly), `custom_domain`.
- Criar igreja (tenant) ja com `slug` e primeiro `super_admin`.
- Convidar usuario: por e-mail -> cria identidade + membership (ou anexa).
- `GET /api/v1/me/tenants` para o seletor.

---

## 6. Fluxos (sequencia)

**Login no subdominio** `igreja.dominio`
1. Front le o host -> `GET /public/tenant/igreja` (branding).
2. `POST /auth/login { email, password, tenant_slug: "igreja" }`.
3. Sem membership ali -> 403 generico + link para `app.dominio`.
4. Com membership -> tokens (tenant ativo = igreja).

**Login no central** `app.dominio`
1. `POST /auth/login { email, password }`.
2. 1 membership -> entra. >1 -> `selection_token` + lista.
3. `POST /auth/select-tenant` -> tokens.

**Troca de igreja** (autenticado)
- Switcher -> `POST /auth/switch-tenant { tenant_id }` -> novos tokens; front recarrega
  dados (mesmo padrao do reload do switcher de filial).

---

## 7. Plano de testes

- **Unidade (`internal/auth`)**: resolucao de tenant - 0/1/N memberships; slug
  invalido; membership inativa; token `select` expirado.
- **RLS (`internal/store/rls_test.go`)**: 
  - `memberships` visivel ao dono e ao tenant correto; isolada entre tenants.
  - `users` global: self, admin do tenant e sistema; outro tenant nao ve.
  - Novo caso: usuario com membership em X e Y nao ve dados de Z.
- **Handlers (`internal/httpapi`)**: contratos de login (direto e selecao),
  `switch-tenant`, `/me` com memberships.
- **Migracao**: backfill idempotente; cenario com e-mail ja existente ao convidar.
- **E2E manual**: login central com 2 igrejas (seletor), login por subdominio,
  troca de igreja, reset de senha afetando todas as igrejas.

---

## 8. Matriz de riscos

| Risco | Impacto | Mitigacao |
|---|---|---|
| DROP de colunas de `users` sem backfill perfeito | Alto | validar contagem antes do DROP; release em 2 passos |
| Politicas RLS de `users` regredirem isolamento | Alto | testes de RLS dedicados; revisao da 000016/000045 |
| Enumerar memberships/e-mails no login | Medio | respostas genericas; rate limit; nao revelar existencia |
| Senha unica vazada afeta todas as igrejas | Alto | MFA obrigatorio para admin; rate limit; alertas |
| Subdominio: cookies/CORS/TLS | Medio | wildcard TLS na Cloudflare; mesma origem |
| Reset de senha por identidade confundir | Medio | UI mostra "afeta todas as suas igrejas" |
| Slug colidir com reservados | Baixo | lista de slugs reservados (`app`, `www`, `api`, `admin`) |

---

## 9. Rollout sugerido (releases pequenos)

1. **R1 - Infra de dados**: `memberships` + backfill, **sem** remover colunas de
   `users`; codigo passa a ler membership com fallback nas colunas antigas.
2. **R2 - Auth/multi-igreja**: login com selecao, `switch-tenant`, `/me`, seletor
   de igreja; testes verdes.
3. **R3 - Limpeza**: remover colunas antigas de `users` + testes de RLS finais.
4. **R4 - Subdominio/branding**: migracao de branding + nginx/Cloudflare + telas.
5. **R5 - Onboarding**: criacao de igreja, convite, branding no admin.

---

## 10. Checklist por PR

**PR1 (dados)**
- [ ] `000051_identity_memberships.up/.down` (tabela + backfill + RLS).
- [ ] `current_user_id()` + GUC `app.user_id` em `store`.
- [ ] Testes de RLS de `memberships`.
- [ ] `auth_memberships()` + ajuste de `auth_lookup_user`.

**PR2 (auth/API)**
- [ ] `Login` com `tenant_slug`/selecao; `select-tenant`; `switch-tenant`.
- [ ] `/me` com `memberships`.
- [ ] Middleware `X-Tenant-Id`/`X-Tenant-Slug`.
- [ ] `internal/users` operando por membership + `user_attach_to_tenant`.

**PR3 (limpeza)**
- [ ] remover colunas de `users`; ajustar handlers/repo que as usavam.
- [ ] varredura de `SELECT/INSERT` em `users`.

**PR4 (subdominio)**
- [ ] branding em `tenants` + `GET /public/tenant/{slug}`.
- [ ] nginx wildcard + Cloudflare (documentar no AGENTS).
- [ ] frontend: branding + `tenant_slug` no login.

**PR5 (onboarding)**
- [ ] `GET /me/tenants`; tela de criacao de igreja e convite; branding no admin.

---

## 11. Itens em aberto (a decidir antes do PR4)

- Dominio base definitivo (ex.: `app.mgmconsultoria.com` + `*.mgmconsultoria.com`)?
- Branding: apenas logo+cor, ou tema completo (fonte, rodape, e-mails com marca)?
- E-mails transacionais (recibos/convites) saem com o branding da igreja do subdominio?
- `slug` reservado e politica de troca de slug depois de criado (link quebra).
- SSO/SAML por igreja entra no roadmap ou fica fora por ora?
