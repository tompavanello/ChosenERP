# Chosen ERP

**SaaS eclesiastico multi-tenant** que unifica secretaria, financeiro, discipulado e
governanca institucional em uma unica plataforma.

Stack: **Go** (monolito modular) - **PostgreSQL 17** (RLS) - **Next.js 15** (Tailwind v4) -
**Docker Compose**.

---

## Inicio rapido

```powershell
.\start.ps1            # sobe tudo (.env, build Go, Docker postgres+api+webadmin+nginx)
.\start.ps1 -Full      # + redis, rabbitmq, prometheus e grafana
```

O script unico `start.ps1` cuida de toda a sequencia: cria o `.env`, compila o binario
Linux da API (imagem `FROM scratch`), sobe os containers - incluindo o webadmin
(Next.js standalone) - e aguarda o healthcheck. Os containers usam
`restart: unless-stopped`, entao sobem sozinhos quando o Docker inicia.

### Portas

| Servico | Porta |
|---|---|
| API (Go) | 38080 |
| PostgreSQL | 35432 |
| Webadmin (Next.js) | 33000 |
| Redis / RabbitMQ / Prometheus / Grafana | 36379 / 35672 / 39090 / 33001 (opcional) |

### Credenciais de desenvolvimento

Os logins existem no seed, mas as senhas **nao sao versionadas**: ficam no `.env`
(gitignorado), em `DEMO_ADMIN_PASSWORD` e `DEMO_NORTE_PASSWORD`. Para le-las:

```bash
grep '^DEMO_' .env
```

| Perfil | Login |
|---|---|
| Admin da Sede | `admin@demo.local` |
| Pastor da Filial Norte | `pastor.norte@demo.local` |

> A aplicacao e publicada em `chosenerp.mgmconsultoria.com` atras do Cloudflare
> Access. Trocar essas senhas de novo e `UPDATE users SET password_hash=...` com
> bcrypt (custo 10) - o formato esta em `internal/auth/password.go`.

---

## Estrutura

```
apps/webadmin/     Painel administrativo (Next.js 15, App Router)
cmd/api/           Entrypoint do servico Go
internal/          Dominios: auth, store, httpapi, members, finance, documents...
db/migrations/     Migracoes SQL versionadas (embutidas via go:embed)
infra/             docker-compose, Dockerfile da API, Prometheus
Docs/              PRD, blueprint e checkpoint
start.ps1          Inicializacao completa (arquivo unico)
```

## Principais funcionalidades (Fase 0 + Fase 1)

- **Multi-tenant com RLS:** isolamento por filial garantido pelo PostgreSQL.
- **Integridade:** `financial_transactions` e `audit_log` sao append-only com hash-chain.
- **Secretaria:** membros (perfil completo), familias, **arvore genealogica e de discipulado**,
  visitantes (trilha de acolhimento), benfeitores, carteirinha QR.
- **Financeiro:** plano de contas, lancamentos (dizimo/oferta/despesa), balancete.
- **Recibos:** geracao automatica, layout HTML para impressao e envio (e-mail/WhatsApp).
- **Relatorios:** balancete mensal e DRE por categoria com comparativo de periodo.

## Documentacao

- `Docs/Chosen_ERP_Documentacao_Completa.md` - PRD e especificacao completa.
- `Docs/01_Blueprint_Arquitetura.md` - decisoes de arquitetura.
- `Docs/00_Checkpoint.md` - historico consolidado do projeto.
- `AGENTS.md` - guia de desenvolvimento (endpoints, migracoes, build).

## Seguranca

- Nunca versione `.env` (ja esta no `.gitignore`).
- A aplicacao conecta como `chosenerp_app` (nao-superuser), garantindo que o RLS seja aplicado.
- Gere um `JWT_SECRET` forte em producao.
