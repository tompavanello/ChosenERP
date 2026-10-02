# Plano - Fluxo de caixa, contas a pagar/receber e baixa em lote

Status: **planejado** (nao implementado). Guardado para execucao futura.
Escopo: criar a camada de **previsao** (contas a pagar/receber) separada do
**realizado** (`financial_transactions`), com parcelas, baixa em lote para uma
conta corrente e relatorio de **fluxo de caixa diario/mensal**.

## Decisoes ja tomadas (com o cliente)

1. **AP/AR unico** com `direction` (`payable`/`receivable`) - nao dois modulos.
2. O **realizado continua** em `financial_transactions`; a baixa e a ponte que
   cria o lancamento efetivo.
3. **Parcelas** com datas/valores proprios; geracao **igual ou customizada**.
4. **Baixa parcial** com historico (varias baixas por parcela).
5. Baixa **em lote**, com modo `single` (um lancamento) ou `per_title` (um por
   titulo), direcionando para **uma conta corrente**.
6. Modo `single`: exigir mesma **direcao + categoria + pagador** (senao `422`);
   emite **recibo unico** do lote.
7. Excluir baixa/lote = **hard delete** (remove o lancamento e recalcula a
   hash-chain via `fin_tx_rechain`), **sem status de estorno** - alinhado a
   decisao da migracao `000061`.
8. Excluir uma baixa individual de um lote `single` e **bloqueado** (so exclui o
   lote inteiro) - `409`.
9. Relatorio de fluxo de caixa **por conta bancaria**, comparando
   **Previsto x Realizado**; granularidade **diaria ou mensal**.
10. Recorrencias atuais **migram para titulos** (`auto_settle = true` para
    preservar o comportamento atual).

---

## 1. Estado atual (pontos tocados)

| Area | Arquivo | Observacao |
|---|---|---|
| Ledger (realizado) | `db/migrations/000006_finance.up.sql`, `000018` | `financial_transactions`, append-only, hash-chain |
| Hard delete | `db/migrations/000061_fin_tx_hard_delete.up.sql` | DELETE permitido; `fin_tx_rechain` recalcula a cadeia |
| Conciliacao | `db/migrations/000062_financial_reconciliation.up.sql` | periodo conciliado trava INSERT/UPDATE/DELETE |
| Auditoria | `db/migrations/000050`, `000052`, `000063` | auditoria fechada bloqueia void/delete |
| Relatorios | `internal/finance/reports.go`, `statement.go` | regime de caixa por `occurred_at`/`voided_at` |
| Extrato | `internal/finance/finance.go:512` (`List`) | sem `origin`/`batch_id` |
| Recorrencias | `internal/finance/recurring.go` | so receita; escreve direto no ledger |
| Rotas | `internal/httpapi/router.go` | bloco Financeiro/Relatorios |
| Reset | `db/migrations/000058` (`reset_operational_data`) | precisa truncar as novas tabelas |
| RLS tests | `internal/store/rls_test.go` | varredura de todas as tabelas |

**Lacuna:** nao existe `due_date`, status nem previsao (backlog #14). Reusar
`financial_transactions` como "previsto" duplicaria no DRE/balanco, quebraria o
append-only e esbarraria na trava de conciliacao.

---

## 2. Modelo de dados (migration `000065_financial_titles.{up,down}.sql`)

### `financial_titles` (cabecalho)

`id`, `tenant_id`, `branch_id`, `direction` (`payable`|`receivable`),
`category_id`, `account_id` (previsao), `supplier_id`, `member_id`,
`benefactor_id`, `description`, `document_ref`, `notes`, `total_amount`,
`status` (`open`|`partial`|`settled`|`cancelled`), `origin`
(`manual`|`recurring`), `created_by`, `created_at`, `updated_at`.

### `financial_installments` (parcelas)

`id`, `tenant_id`, `branch_id` (desnormalizado p/ RLS), `title_id` (cascade),
`seq`, `due_date`, `amount`, `paid_amount` (default 0), `status`
(`open`|`partial`|`paid`|`cancelled`), `last_settled_at`, timestamps.
`UNIQUE (title_id, seq)`.

### `financial_settlement_batches` (lote de baixa)

`id`, `tenant_id`, `branch_id`, `account_id` (conta corrente de destino),
`settled_at` (date), `payment_method`, `mode` (`single`|`per_title`),
`total_amount`, `notes`, `created_by`, `created_at`.

### `financial_installment_settlements` (baixas)

`id`, `tenant_id`, `branch_id`, `batch_id` (FK), `installment_id` (FK),
`transaction_id` (FK `financial_transactions`), `amount`, `settled_at`,
`payment_method`, `created_by`, `created_at`.

### Regras

- Triggers: recalcular `paid_amount`/status da parcela e do titulo a partir das
  baixas; impedir `paid_amount > amount`.
- RLS `rls_read`/`rls_write` por filial nas 4 tabelas.
- Indices: `(tenant_id, branch_id, due_date, status)` em parcelas;
  `settlements(batch_id)`, `settlements(installment_id)`,
  `settlements(transaction_id)`.
- Permissao nova `finance.settle` (Sede, tesoureiro, conciliador).
- O `account_id` do lote e o destino real da baixa (pode diferir da previsao do
  titulo).

---

## 3. Fluxos e endpoints da API

### Titulos / parcelas

- `GET/POST /api/v1/finance/titles`
- `GET/PATCH/DELETE /api/v1/finance/titles/{id}`
- `POST /api/v1/finance/titles/{id}/installments` (adiciona parcela)
- `PATCH/DELETE /api/v1/finance/installments/{id}` (reagendar/editar/cancelar
  parcela em aberto)

### Baixa em lote

`POST /api/v1/finance/settlements`

```json
{
  "account_id": "...",
  "settled_at": "2026-09-25",
  "payment_method": "pix",
  "mode": "single",
  "items": [{ "installment_id": "...", "amount": 123.45 }],
  "notes": "..."
}
```

- `per_title`: cada item cria seu `financial_transactions` via `Repo.Create`
  (categoria/pagador/recibo corretos).
- `single`: valida mesma direcao + categoria + pagador (senao `422`); cria **um**
  lancamento somando o lote e **um** recibo; todas as baixas apontam para o
  mesmo `transaction_id`.
- Tudo atomico dentro de `WithTenant`.

### Consultas do vinculo

- `GET /api/v1/finance/settlement-batches` e `/settlement-batches/{id}` (lote +
  baixas + titulos + lancamentos).
- `GET /api/v1/finance/installments/{id}/settlements` (historico por parcela).
- `GET /api/v1/finance/transactions/{id}/settlements` (origem do lancamento).
- Extrato de lancamentos: incluir `origin` (baixa de titulo / lote) e `batch_id`
  via join em `financial_installment_settlements`.

### Exclusao (hard delete)

- `DELETE /api/v1/finance/settlement-batches/{id}`: remove os lancamentos
  (`Repo.Delete` + `fin_tx_rechain`), depois baixas e lote.
- `DELETE /api/v1/finance/settlements/{id}`: bloqueado (`409`) se o lote for
  `single`.
- Periodo conciliado ou auditoria fechada => `409`.

---

## 4. Relatorio de Fluxo de Caixa

`GET /api/v1/reports/cash-flow?from=&to=&granularity=day|month&basis=realized|forecast|both&account_id=`

- `realized`: `financial_transactions` (`voided_at IS NULL`) por dia/mes
  (entradas, saidas, liquido, saldo acumulado).
- `forecast`: parcelas `open`/`partial` por `due_date` (a pagar/a receber).
- `both`: colunas realizado + previsto + saldo projetado.
- Saldo inicial = liquido realizado antes de `from` (filtravel por conta).
- Export CSV/XLSX/PDF no padrao de `monthly-statement/export`
  (`internal/finance/cashflow.go`).

---

## 5. Recorrencias (`internal/finance/recurring.go`)

- Novas colunas `auto_settle` (default `true`) e horizonte de projecao.
- Worker deixa de escrever direto no ledger: gera titulo a receber + parcelas
  futuras; se `auto_settle`, efetiva pelo mesmo caminho de `SettleBatch` na data.
- Sem `auto_settle`, a parcela fica aberta para baixa manual.

---

## 6. Frontend

- `Financeiro -> Contas a Pagar/Receber`: lista com filtros (direcao/status/
  periodo), criacao (titulo + N parcelas iguais ou customizadas), detalhe com
  historico de baixas, modal de **baixa em lote** (selecao de titulos, conta,
  data, modo `single`/`per_title`), exclusao definitiva.
- `Relatorios -> Fluxo de Caixa`: switch **Diario | Mensal**, periodo, filtro de
  conta, alternancia **Realizado / Previsto / Consolidado**; tabela + grafico
  `recharts`.
- `lib/api.ts` (funcoes/tipos), `lib/constants.ts` (rotulos de status/origem),
  entradas em `apps/webadmin/app/dashboard/layout.tsx` filtradas por permissao.

---

## 7. Qualidade / infra

- `reset_operational_data()` (`000058`) truncando as novas tabelas (ordem
  correta: settlements -> batches -> installments -> titles).
- Incluir as 4 tabelas na varredura de RLS em `internal/store/rls_test.go`.
- Testes: baixa parcial/status, `single` x `per_title`, validacao `422`,
  hard delete + rechain, agregacao do fluxo.
- Atualizar `AGENTS.md` (tabela de endpoints) e `Docs/02_Backlog.md` (#14).

---

## 8. Fases de execucao

1. Migration `000065` + reset + RLS tests.
2. Repo de titulos/parcelas + settlements (lote) + handlers/rotas + permissao
   `finance.settle`.
3. Migracao do worker de recorrencia.
4. Tela Contas a Pagar/Receber.
5. Tela Fluxo de Caixa.
6. Testes, `go build/vet/test`, build do webadmin, docs.

---

## 9. Pontos em aberto / assumidos

- Modo `single` exige mesma categoria e pagador; se a operacao real precisar
  agrupar categorias, avaliar no futuro uma regra de "agrupamento por
  categoria/pagador" (hoje descartada).
- O endpoint legado `POST /finance/transactions/{id}/void` permanece para
  lancamentos avulsos; o fluxo de baixa usa exclusao definitiva.
- Recorrencias nascem com `auto_settle = true` para nao quebrar o comportamento
  atual (gera previsto e efetiva na data).
