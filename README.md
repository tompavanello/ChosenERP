# Chosen ERP

**SaaS eclesiástico multi-tenant** que unifica secretaria, financeiro, discipulado e
governança institucional em uma única plataforma.

Stack: **Go** (monolito modular) · **PostgreSQL 17** (RLS) · **Next.js 15** (Tailwind v4) ·
**Docker Compose**.

---

## Início rápido

```powershell
.\start.ps1            # sobe tudo (.env, build Go, Docker postgres+api, webadmin)
.\start.ps1 -Full      # + redis, rabbitmq, prometheus e grafana
.\start.ps1 -NoWeb     # apenas infraestrutura e API
```

O script único `start.ps1` cuida de toda a sequência: cria o `.env`, compila o binário
Linux da API (imagem `FROM scratch`), sobe os containers, aguarda o healthcheck e inicia
o webadmin.

### Portas

| Serviço | Porta |
|---|---|
| API (Go) | 38080 |
| PostgreSQL | 35432 |
| Webadmin (Next.js) | 33000 |
| Redis / RabbitMQ / Prometheus / Grafana | 36379 / 35672 / 39090 / 33001 (opcional) |

### Credenciais de desenvolvimento

| Perfil | Login | Senha |
|---|---|---|
| Admin da Sede | `admin@demo.local` | `admin123` |
| Pastor da Filial Norte | `pastor.norte@demo.local` | `norte123` |

---

## Estrutura

```
apps/webadmin/     Painel administrativo (Next.js 15, App Router)
cmd/api/           Entrypoint do serviço Go
internal/          Domínios: auth, store, httpapi, members, finance, documents...
db/migrations/     Migrações SQL versionadas (embutidas via go:embed)
infra/             docker-compose, Dockerfile da API, Prometheus
Docs/              PRD, blueprint e checkpoint
start.ps1          Inicialização completa (arquivo único)
```

## Principais funcionalidades (Fase 0 + Fase 1)

- **Multi-tenant com RLS:** isolamento por filial garantido pelo PostgreSQL.
- **Integridade:** `financial_transactions` e `audit_log` são append-only com hash-chain.
- **Secretaria:** membros (perfil completo), famílias, **árvore genealógica e de discipulado**,
  visitantes (trilha de acolhimento), benfeitores, carteirinha QR.
- **Financeiro:** plano de contas, lançamentos (dízimo/oferta/despesa), balancete.
- **Recibos:** geração automática, layout HTML para impressão e envio (e-mail/WhatsApp).
- **Relatórios:** balancete mensal e DRE por categoria com comparativo de período.

## Documentação

- `Docs/Chosen_ERP_Documentacao_Completa.md` — PRD e especificação completa.
- `Docs/01_Blueprint_Arquitetura.md` — decisões de arquitetura.
- `Docs/00_Checkpoint.md` — histórico consolidado do projeto.
- `AGENTS.md` — guia de desenvolvimento (endpoints, migrações, build).

## Segurança

- Nunca versione `.env` (já está no `.gitignore`).
- A aplicação conecta como `chosenerp_app` (não-superuser), garantindo que o RLS seja aplicado.
- Gere um `JWT_SECRET` forte em produção.
