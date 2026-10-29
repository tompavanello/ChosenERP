package store

import (
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
)

// TestAccountTransfers_Guards valida no banco os invariantes das transferencias
// entre contas (000108): tipo de conta, protecao da perna contra edicao e
// proibicao de agrupar. A logica de criacao/saldo fica em internal/finance.
func TestAccountTransfers_Guards(t *testing.T) {
	const date = "2025-07-01"
	const transferID = "f0000000-0000-4000-8000-000000000001"
	const legOut = "f0000000-0000-4000-8000-000000000002"
	const legIn = "f0000000-0000-4000-8000-000000000003"
	const groupID = "f0000000-0000-4000-8000-000000000004"

	err := inBounds(t, bounds(fixTenantX, fixBranchA, "tesoureiro"), func(tx pgx.Tx) error {
		// Tipo de conta invalido e recusado pelo CHECK.
		if _, err := tx.Exec(testCtx, `SAVEPOINT bad_type`); err != nil {
			return err
		}
		_, typeErr := tx.Exec(testCtx,
			`UPDATE financial_accounts SET account_type = 'bogus' WHERE id = $1::uuid`, fixAccountA)
		if _, err := tx.Exec(testCtx, `ROLLBACK TO SAVEPOINT bad_type`); err != nil {
			return err
		}
		if typeErr == nil {
			t.Errorf("account_type invalido deveria ser recusado")
		}

		if _, err := tx.Exec(testCtx, `
			INSERT INTO account_transfers
				(id, tenant_id, branch_id, from_account_id, to_account_id, kind, amount, occurred_at, description)
			VALUES ($1::uuid, $2::uuid, $3::uuid, $4::uuid, $5::uuid, 'deposit', 100, $6::date, 'teste')`,
			transferID, fixTenantX, fixBranchA, fixAccountA, fixAccountB, date); err != nil {
			return err
		}
		if _, err := tx.Exec(testCtx, `
			INSERT INTO financial_transactions
				(id, tenant_id, branch_id, type, amount, currency, account_id, description, occurred_at, account_transfer_id)
			VALUES ($1::uuid, $2::uuid, $3::uuid, 'expense', 100, 'BRL', $4::uuid, 'saida', $5::date, $6::uuid),
			       ($7::uuid, $2::uuid, $3::uuid, 'income', 100, 'BRL', $8::uuid, 'entrada', $5::date, $6::uuid)`,
			legOut, fixTenantX, fixBranchA, fixAccountA, date, transferID, legIn, fixAccountB); err != nil {
			return err
		}

		// account_transfer_id e protegido pelo fin_tx_guard.
		if _, err := tx.Exec(testCtx, `SAVEPOINT guard_transfer`); err != nil {
			return err
		}
		_, guardErr := tx.Exec(testCtx,
			`UPDATE financial_transactions SET account_transfer_id = NULL WHERE id = $1::uuid`, legOut)
		if _, err := tx.Exec(testCtx, `ROLLBACK TO SAVEPOINT guard_transfer`); err != nil {
			return err
		}
		if guardErr == nil || !strings.Contains(guardErr.Error(), "append-only") {
			t.Errorf("alteracao de account_transfer_id deveria ser bloqueada, veio: %v", guardErr)
		}

		// Perna de transferencia nao pode ser agrupada.
		if _, err := tx.Exec(testCtx, `
			INSERT INTO financial_transaction_groups
				(id, tenant_id, branch_id, account_id, type, occurred_at, description)
			VALUES ($1::uuid, $2::uuid, $3::uuid, $4::uuid, 'income', $5::date, 'grupo')`,
			groupID, fixTenantX, fixBranchA, fixAccountB, date); err != nil {
			return err
		}
		if _, err := tx.Exec(testCtx, `SAVEPOINT group_transfer`); err != nil {
			return err
		}
		_, groupErr := tx.Exec(testCtx,
			`SELECT fin_group_attach($1::uuid, ARRAY[$2]::uuid[])`, groupID, legIn)
		if _, err := tx.Exec(testCtx, `ROLLBACK TO SAVEPOINT group_transfer`); err != nil {
			return err
		}
		if groupErr == nil || !strings.Contains(groupErr.Error(), "transferencia entre contas nao pode ser agrupada") {
			t.Errorf("agrupar perna de transferencia deveria falhar, veio: %v", groupErr)
		}

		// Sanidade: as duas pernas existem e tem o mesmo vinculo.
		var n int
		if err := tx.QueryRow(testCtx,
			`SELECT count(*)::int FROM financial_transactions WHERE account_transfer_id = $1::uuid`, transferID).
			Scan(&n); err != nil {
			return err
		}
		if n != 2 {
			t.Errorf("esperava 2 pernas, veio %d", n)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("WithTenant: %v", err)
	}
}
