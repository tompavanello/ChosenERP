package store

import (
	"testing"

	"github.com/jackc/pgx/v5"
)

func insertGroupTxn(t *testing.T, tx pgx.Tx, id, branch, account, kind, date, description string, amount float64) {
	t.Helper()
	if _, err := tx.Exec(testCtx, `
		INSERT INTO financial_transactions
			(id, tenant_id, branch_id, category_id, type, amount, currency,
			 account_id, description, occurred_at)
		VALUES ($1::uuid, $2::uuid, $3::uuid, $4::uuid, $5, $6, 'BRL',
			$7::uuid, $8, $9::date)`,
		id, fixTenantX, branch, fixCatA, kind, amount, account, description, date); err != nil {
		t.Fatalf("insert lancamento %s: %v", id, err)
	}
}

func insertTxGroup(t *testing.T, tx pgx.Tx, id, branch, account, kind, date, description string) {
	t.Helper()
	if _, err := tx.Exec(testCtx, `
		INSERT INTO financial_transaction_groups
			(id, tenant_id, branch_id, account_id, type, occurred_at, description)
		VALUES ($1::uuid, $2::uuid, $3::uuid, $4::uuid, $5, $6::date, $7)`,
		id, fixTenantX, branch, account, kind, date, description); err != nil {
		t.Fatalf("insert grupo %s: %v", id, err)
	}
}

// TestFinancialTransactionGroups_Lifecycle cobre o ciclo completo: anexar,
// bloquear conciliacao individual, conciliar em bloco, travar o grupo e
// desagrupar. O rollback final mantem o banco de teste intacto.
func TestFinancialTransactionGroups_Lifecycle(t *testing.T) {
	const (
		tx1 = "e0000000-0000-4000-8000-000000000101"
		tx2 = "e0000000-0000-4000-8000-000000000102"
		grp = "e0000000-0000-4000-8000-000000000103"
	)

	err := inBounds(t, bounds(fixTenantX, fixBranchA, "tesoureiro"), func(tx pgx.Tx) error {
		insertGroupTxn(t, tx, tx1, fixBranchA, fixAccountA, "income", "2025-06-01", "dizimo 1", 400)
		insertGroupTxn(t, tx, tx2, fixBranchA, fixAccountA, "income", "2025-06-02", "dizimo 2", 600)
		insertTxGroup(t, tx, grp, fixBranchA, fixAccountA, "income", "2025-06-10", "deposito")

		if _, err := tx.Exec(testCtx,
			`SELECT fin_group_attach($1::uuid, ARRAY[$2,$3]::uuid[])`, grp, tx1, tx2); err != nil {
			return err
		}

		var linked int
		if err := tx.QueryRow(testCtx,
			`SELECT count(*)::int FROM financial_transactions WHERE group_id = $1::uuid`, grp).Scan(&linked); err != nil {
			return err
		}
		if linked != 2 {
			t.Fatalf("esperava 2 lancamentos no grupo, veio %d", linked)
		}

		// Conciliacao individual de membro de grupo e recusada.
		expectSavepointError(t, tx, "grp_individual", "agrupado", func() error {
			_, err := tx.Exec(testCtx, `SELECT fin_tx_reconcile($1::uuid, true, '')`, tx1)
			return err
		})

		// Conciliacao do grupo marca todos os membros.
		if _, err := tx.Exec(testCtx, `SELECT fin_group_reconcile($1::uuid, true, '')`, grp); err != nil {
			return err
		}
		var status string
		var reconciled int
		if err := tx.QueryRow(testCtx,
			`SELECT status FROM financial_transaction_groups WHERE id = $1::uuid`, grp).Scan(&status); err != nil {
			return err
		}
		if status != "conciliado" {
			t.Fatalf("status do grupo: esperava conciliado, veio %q", status)
		}
		if err := tx.QueryRow(testCtx, `
			SELECT count(*)::int FROM financial_transactions
			WHERE group_id = $1::uuid AND reconciled_at IS NOT NULL`, grp).Scan(&reconciled); err != nil {
			return err
		}
		if reconciled != 2 {
			t.Fatalf("esperava 2 membros conciliados, veio %d", reconciled)
		}

		// Grupo conciliado: imutavel e nao pode ser excluido/desagrupado.
		expectSavepointError(t, tx, "grp_delete_locked", "conciliado", func() error {
			_, err := tx.Exec(testCtx, `DELETE FROM financial_transaction_groups WHERE id = $1::uuid`, grp)
			return err
		})
		expectSavepointError(t, tx, "grp_update_locked", "conciliado", func() error {
			_, err := tx.Exec(testCtx,
				`UPDATE financial_transaction_groups SET description = 'outro' WHERE id = $1::uuid`, grp)
			return err
		})
		expectSavepointError(t, tx, "grp_detach_locked", "conciliado", func() error {
			_, err := tx.Exec(testCtx, `SELECT fin_group_detach($1::uuid, ARRAY[$2]::uuid[])`, grp, tx1)
			return err
		})

		// Desconciliar libera o grupo e os membros.
		if _, err := tx.Exec(testCtx, `SELECT fin_group_reconcile($1::uuid, false, '')`, grp); err != nil {
			return err
		}
		var pending int
		if err := tx.QueryRow(testCtx, `
			SELECT count(*)::int FROM financial_transactions
			WHERE group_id = $1::uuid AND reconciled_at IS NOT NULL`, grp).Scan(&pending); err != nil {
			return err
		}
		if pending != 0 {
			t.Fatalf("apos desconcliar, esperava 0 membros conciliados, veio %d", pending)
		}

		// Desanexar um membro deixa 1 lancamento: reconciliar deve recusar.
		if _, err := tx.Exec(testCtx, `SELECT fin_group_detach($1::uuid, ARRAY[$2]::uuid[])`, grp, tx1); err != nil {
			return err
		}
		expectSavepointError(t, tx, "grp_too_small", "pelo menos 2", func() error {
			_, err := tx.Exec(testCtx, `SELECT fin_group_reconcile($1::uuid, true, '')`, grp)
			return err
		})
		return errRollback
	})
	if err != nil {
		t.Fatalf("inBounds: %v", err)
	}
}

// TestFinancialTransactionGroups_Consistency cobre as regras de integridade:
// mesma filial/conta/tipo, lancamentos livres e escopo do tenant.
func TestFinancialTransactionGroups_Consistency(t *testing.T) {
	const (
		tx1    = "e0000000-0000-4000-8000-000000000111"
		tx2    = "e0000000-0000-4000-8000-000000000112"
		txB    = "e0000000-0000-4000-8000-000000000113"
		txDone = "e0000000-0000-4000-8000-000000000114"
		txVoid = "e0000000-0000-4000-8000-000000000115"
		grp    = "e0000000-0000-4000-8000-000000000116"
	)

	// Sede: enxerga e grava nas duas filiais, o que permite montar o cenario de
	// filial divergente. A consistencia continua sendo garantida pelo trigger.
	err := inBounds(t, bounds(fixTenantX, "", "super_admin"), func(tx pgx.Tx) error {
		insertGroupTxn(t, tx, tx1, fixBranchA, fixAccountA, "income", "2025-06-01", "a", 100)
		insertGroupTxn(t, tx, tx2, fixBranchA, fixAccountA, "income", "2025-06-02", "b", 200)
		insertGroupTxn(t, tx, txB, fixBranchB, fixAccountB, "income", "2025-06-03", "outra filial", 300)
		insertGroupTxn(t, tx, txDone, fixBranchA, fixAccountA, "income", "2025-06-04", "conciliado", 400)
		insertGroupTxn(t, tx, txVoid, fixBranchA, fixAccountA, "income", "2025-06-05", "estornado", 500)
		insertTxGroup(t, tx, grp, fixBranchA, fixAccountA, "income", "2025-06-10", "deposito")

		if _, err := tx.Exec(testCtx,
			`SELECT fin_group_attach($1::uuid, ARRAY[$2,$3]::uuid[])`, grp, tx1, tx2); err != nil {
			return err
		}

		// Lancamento de outra filial nao entra no grupo (trigger de consistencia).
		expectSavepointError(t, tx, "grp_other_branch", "incompativel", func() error {
			_, err := tx.Exec(testCtx, `SELECT fin_group_attach($1::uuid, ARRAY[$2]::uuid[])`, grp, txB)
			return err
		})

		// Lancamento estornado ou ja conciliado nao pode ser agrupado.
		if _, err := tx.Exec(testCtx,
			`UPDATE financial_transactions SET voided_at = now() WHERE id = $1::uuid`, txVoid); err != nil {
			return err
		}
		expectSavepointError(t, tx, "grp_voided", "estornado", func() error {
			_, err := tx.Exec(testCtx, `SELECT fin_group_attach($1::uuid, ARRAY[$2]::uuid[])`, grp, txVoid)
			return err
		})

		if _, err := tx.Exec(testCtx, `SELECT fin_tx_reconcile($1::uuid, true, '')`, txDone); err != nil {
			return err
		}
		expectSavepointError(t, tx, "grp_reconciled", "conciliado", func() error {
			_, err := tx.Exec(testCtx, `SELECT fin_group_attach($1::uuid, ARRAY[$2]::uuid[])`, grp, txDone)
			return err
		})

		// Chamar a RPC com outro tenant no contexto falha; o grupo continua no X.
		if _, err := tx.Exec(testCtx, `SELECT set_config('app.tenant_id', $1, true)`, fixTenantY); err != nil {
			return err
		}
		expectSavepointError(t, tx, "grp_other_tenant", "escopo", func() error {
			_, err := tx.Exec(testCtx, `SELECT fin_group_reconcile($1::uuid, true, '')`, grp)
			return err
		})
		return errRollback
	})
	if err != nil {
		t.Fatalf("inBounds: %v", err)
	}
}
