package store

import (
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
)

func insertFinTx(t *testing.T, tx pgx.Tx, id, date, description string) {
	t.Helper()
	if _, err := tx.Exec(testCtx, `
		INSERT INTO financial_transactions
			(id, tenant_id, branch_id, category_id, type, amount, currency,
			 account_id, description, occurred_at, entry_seq)
		VALUES ($1::uuid, $2::uuid, $3::uuid, $4::uuid, 'income', 100, 'BRL',
			$5::uuid, $6, $7::date, 1)`,
		id, fixTenantX, fixBranchA, fixCatA, fixAccountA, description, date); err != nil {
		t.Fatalf("insert lancamento %s: %v", id, err)
	}
}

func expectSavepointError(t *testing.T, tx pgx.Tx, sp, want string, fn func() error) {
	t.Helper()
	if _, err := tx.Exec(testCtx, "SAVEPOINT "+sp); err != nil {
		t.Fatalf("savepoint %s: %v", sp, err)
	}
	err := fn()
	if _, rbErr := tx.Exec(testCtx, "ROLLBACK TO SAVEPOINT "+sp); rbErr != nil {
		t.Fatalf("rollback savepoint %s: %v", sp, rbErr)
	}
	if _, relErr := tx.Exec(testCtx, "RELEASE SAVEPOINT "+sp); relErr != nil {
		t.Fatalf("release savepoint %s: %v", sp, relErr)
	}
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), want) {
		t.Fatalf("esperava erro contendo %q, veio: %v", want, err)
	}
}

// TestFinancialTransactions_ReconcileLocks cobre a conciliacao individual:
// o lancamento fica imutavel (inclusive paid_at/entry_seq, que nao passam pelo
// guard append-only) e a RPC libera a edicao quando desconcilia.
func TestFinancialTransactions_ReconcileLocks(t *testing.T) {
	const txID = "e0000000-0000-4000-8000-0000000000f1"

	err := inBounds(t, bounds(fixTenantX, fixBranchA, "tesoureiro"), func(tx pgx.Tx) error {
		insertFinTx(t, tx, txID, "2025-06-10", "conciliar")

		if _, err := tx.Exec(testCtx, `SELECT fin_tx_reconcile($1::uuid, true, '')`, txID); err != nil {
			return err
		}
		var reconciled *string
		if err := tx.QueryRow(testCtx,
			`SELECT to_char(reconciled_at,'YYYY-MM-DD') FROM financial_transactions WHERE id = $1::uuid`, txID).
			Scan(&reconciled); err != nil {
			return err
		}
		if reconciled == nil {
			t.Fatal("reconciled_at nao foi preenchido")
		}

		// paid_at nao esta no guard append-only; a trava de conciliacao precisa
		// bloquear mesmo assim.
		expectSavepointError(t, tx, "recon_paid", "conciliado", func() error {
			_, err := tx.Exec(testCtx,
				`UPDATE financial_transactions SET paid_at = now() WHERE id = $1::uuid`, txID)
			return err
		})
		expectSavepointError(t, tx, "recon_delete", "conciliado", func() error {
			_, err := tx.Exec(testCtx,
				`DELETE FROM financial_transactions WHERE id = $1::uuid`, txID)
			return err
		})

		if _, err := tx.Exec(testCtx, `SELECT fin_tx_reconcile($1::uuid, false, '')`, txID); err != nil {
			return err
		}
		if _, err := tx.Exec(testCtx,
			`UPDATE financial_transactions SET paid_at = now() WHERE id = $1::uuid`, txID); err != nil {
			t.Fatalf("apos desconciliar, paid_at deveria ser aceito: %v", err)
		}
		return errRollback
	})
	if err != nil {
		t.Fatalf("inBounds: %v", err)
	}
}

// TestFinancialTransactions_PeriodLock cobre o encerramento do periodo:
// INSERT dentro do intervalo, saida dele, entrada nele e DELETE sao bloqueados.
func TestFinancialTransactions_PeriodLock(t *testing.T) {
	const (
		outsideID = "e0000000-0000-4000-8000-0000000000f2"
		insideID  = "e0000000-0000-4000-8000-0000000000f3"
		newID     = "e0000000-0000-4000-8000-0000000000f4"
		reconID   = "e0000000-0000-4000-8000-0000000000f5"
	)

	err := inBounds(t, bounds(fixTenantX, fixBranchA, "tesoureiro"), func(tx pgx.Tx) error {
		insertFinTx(t, tx, outsideID, "2025-05-20", "fora")
		insertFinTx(t, tx, insideID, "2025-07-10", "dentro")

		if _, err := tx.Exec(testCtx, `
			INSERT INTO financial_reconciliations
				(id, tenant_id, branch_id, title, period_start, period_end, status)
			VALUES ($1::uuid, $2::uuid, $3::uuid, 'fechamento', '2025-07-01', '2025-07-31', 'conciliada')`,
			reconID, fixTenantX, fixBranchA); err != nil {
			return err
		}

		expectSavepointError(t, tx, "period_insert", "periodo encerrado", func() error {
			_, err := tx.Exec(testCtx, `
				INSERT INTO financial_transactions
					(tenant_id, branch_id, category_id, type, amount, currency,
					 account_id, description, occurred_at)
				VALUES ($1::uuid, $2::uuid, $3::uuid, 'income', 10, 'BRL',
					$4::uuid, 'novo no fechado', '2025-07-15')`,
				fixTenantX, fixBranchA, fixCatA, fixAccountA)
			return err
		})

		expectSavepointError(t, tx, "period_move_in", "periodo encerrado", func() error {
			_, err := tx.Exec(testCtx, `
				SELECT fin_tx_edit($1::uuid, $2, 'income', 100, 'pix', '', '', '', '', false,
					'mover para dentro', '2025-07-15', $3, '', '', false)`,
				outsideID, fixCatA, fixAccountA)
			return err
		})

		expectSavepointError(t, tx, "period_move_out", "periodo encerrado", func() error {
			_, err := tx.Exec(testCtx, `
				SELECT fin_tx_edit($1::uuid, $2, 'income', 100, 'pix', '', '', '', '', false,
					'mover para fora', '2025-08-01', $3, '', '', false)`,
				insideID, fixCatA, fixAccountA)
			return err
		})

		expectSavepointError(t, tx, "period_delete", "periodo encerrado", func() error {
			_, err := tx.Exec(testCtx,
				`DELETE FROM financial_transactions WHERE id = $1::uuid`, insideID)
			return err
		})

		expectSavepointError(t, tx, "period_reconcile", "periodo encerrado", func() error {
			_, err := tx.Exec(testCtx, `SELECT fin_tx_reconcile($1::uuid, true, '')`, insideID)
			return err
		})
		return errRollback
	})
	if err != nil {
		t.Fatalf("inBounds: %v", err)
	}
}

// TestFinancialTransactions_AuditClosedLocksPeriod garante que auditoria
// fechada tambem bloqueia INSERT/UPDATE/DELETE no periodo.
func TestFinancialTransactions_AuditClosedLocksPeriod(t *testing.T) {
	const (
		insideID = "e0000000-0000-4000-8000-0000000000f6"
		auditID  = "e0000000-0000-4000-8000-0000000000f7"
	)

	err := inBounds(t, bounds(fixTenantX, fixBranchA, "tesoureiro"), func(tx pgx.Tx) error {
		insertFinTx(t, tx, insideID, "2025-09-10", "auditado")

		if _, err := tx.Exec(testCtx, `
			INSERT INTO financial_audits
				(id, tenant_id, branch_id, title, period_start, period_end, status)
			VALUES ($1::uuid, $2::uuid, $3::uuid, 'auditoria fechada', '2025-09-01', '2025-09-15', 'fechada')`,
			auditID, fixTenantX, fixBranchA); err != nil {
			return err
		}

		expectSavepointError(t, tx, "audit_move_out", "periodo encerrado", func() error {
			_, err := tx.Exec(testCtx, `
				SELECT fin_tx_edit($1::uuid, $2, 'income', 100, 'pix', '', '', '', '', false,
					'mover para fora', '2025-10-01', $3, '', '', false)`,
				insideID, fixCatA, fixAccountA)
			return err
		})
		expectSavepointError(t, tx, "audit_delete", "periodo encerrado", func() error {
			_, err := tx.Exec(testCtx,
				`DELETE FROM financial_transactions WHERE id = $1::uuid`, insideID)
			return err
		})
		return errRollback
	})
	if err != nil {
		t.Fatalf("inBounds: %v", err)
	}
}

// TestFinancialPeriods_CrossCloseGuards impede fechar auditoria com conciliacao
// aberta sobreposta e impede abrir conciliacao sobre auditoria ja fechada.
func TestFinancialPeriods_CrossCloseGuards(t *testing.T) {
	err := inBounds(t, bounds(fixTenantX, fixBranchA, "tesoureiro"), func(tx pgx.Tx) error {
		if _, err := tx.Exec(testCtx, `
			INSERT INTO financial_reconciliations
				(id, tenant_id, branch_id, title, period_start, period_end, status)
			VALUES ('e0000000-0000-4000-8000-0000000000f8', $1::uuid, $2::uuid,
				'aberta', '2025-11-01', '2025-11-30', 'aberta')`, fixTenantX, fixBranchA); err != nil {
			return err
		}
		if _, err := tx.Exec(testCtx, `
			INSERT INTO financial_audits
				(id, tenant_id, branch_id, title, period_start, period_end, status)
			VALUES ('e0000000-0000-4000-8000-0000000000f9', $1::uuid, $2::uuid,
				'auditoria', '2025-11-05', '2025-11-10', 'aberta')`, fixTenantX, fixBranchA); err != nil {
			return err
		}

		expectSavepointError(t, tx, "audit_close_open_recon", "conciliacao aberta", func() error {
			_, err := tx.Exec(testCtx,
				`UPDATE financial_audits SET status = 'fechada' WHERE id = 'e0000000-0000-4000-8000-0000000000f9'`)
			return err
		})

		if _, err := tx.Exec(testCtx, `
			INSERT INTO financial_audits
				(id, tenant_id, branch_id, title, period_start, period_end, status)
			VALUES ('e0000000-0000-4000-8000-0000000000fa', $1::uuid, $2::uuid,
				'auditoria fechada', '2025-12-01', '2025-12-31', 'fechada')`, fixTenantX, fixBranchA); err != nil {
			return err
		}

		expectSavepointError(t, tx, "recon_over_closed_audit", "auditoria", func() error {
			_, err := tx.Exec(testCtx, `
				INSERT INTO financial_reconciliations
					(id, tenant_id, branch_id, title, period_start, period_end, status)
				VALUES ('e0000000-0000-4000-8000-0000000000fb', $1::uuid, $2::uuid,
					'conflito', '2025-12-05', '2025-12-20', 'aberta')`, fixTenantX, fixBranchA)
			return err
		})
		return errRollback
	})
	if err != nil {
		t.Fatalf("inBounds: %v", err)
	}
}
