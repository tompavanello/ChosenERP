package store

import (
	"testing"

	"github.com/jackc/pgx/v5"
)

// TestFinancialTransactions_Reversal cobre as garantias da migracao 000113 no
// banco: sinal/bank_direction, travamento do contra-lancamento e recusa de
// agrupamento. O fluxo de aplicacao (Reverse, limite parcial e recibo) usa as
// mesmas colunas/triggers validados aqui.
func TestFinancialTransactions_Reversal(t *testing.T) {
	const (
		original = "e0000000-0000-4000-8000-000000000201"
		reversal = "e0000000-0000-4000-8000-000000000202"
		group    = "e0000000-0000-4000-8000-000000000203"
		date     = "2031-01-15"
	)

	err := inBounds(t, bounds(fixTenantX, fixBranchA, "tesoureiro"), func(tx pgx.Tx) error {
		if _, err := tx.Exec(testCtx, `
			INSERT INTO financial_transactions
				(id, tenant_id, branch_id, category_id, type, amount, currency,
				 account_id, description, occurred_at)
			VALUES ($1::uuid, $2::uuid, $3::uuid, $4::uuid, 'income', 1000, 'BRL',
				$5::uuid, 'dizimo original', $6::date)`,
			original, fixTenantX, fixBranchA, fixCatA, fixAccountA, date); err != nil {
			return err
		}

		// Default: entrada normal.
		var sign int
		var bank string
		if err := tx.QueryRow(testCtx,
			`SELECT sign, bank_direction FROM financial_transactions WHERE id = $1::uuid`, original).
			Scan(&sign, &bank); err != nil {
			return err
		}
		if sign != 1 || bank != "income" {
			t.Fatalf("original: sign=%d bank_direction=%s", sign, bank)
		}

		// Contra-receita: sign=-1 => debito no banco, mesma conta de receita.
		if _, err := tx.Exec(testCtx, `
			INSERT INTO financial_transactions
				(id, tenant_id, branch_id, category_id, type, amount, currency,
				 account_id, description, occurred_at, sign, reversal_of_id, reversal_reason)
			VALUES ($1::uuid, $2::uuid, $3::uuid, $4::uuid, 'income', 300, 'BRL',
				$5::uuid, 'cheque devolvido', $6::date, -1, $7::uuid, 'cheque devolvido')`,
			reversal, fixTenantX, fixBranchA, fixCatA, fixAccountA, date, original); err != nil {
			return err
		}
		var revOf string
		if err := tx.QueryRow(testCtx,
			`SELECT sign, bank_direction, reversal_of_id::text FROM financial_transactions WHERE id = $1::uuid`, reversal).
			Scan(&sign, &bank, &revOf); err != nil {
			return err
		}
		if revOf != original {
			t.Fatalf("reversal_of_id = %q, queria %q", revOf, original)
		}
		if sign != -1 || bank != "expense" {
			t.Fatalf("estorno: sign=%d bank_direction=%s", sign, bank)
		}

		// Hash-chain recalculado com sign/reversal no corpo.
		if _, err := tx.Exec(testCtx, `SELECT fin_tx_rechain($1::uuid)`, fixTenantX); err != nil {
			return err
		}
		var hashes int
		if err := tx.QueryRow(testCtx, `
			SELECT count(*)::int FROM financial_transactions
			WHERE id = ANY(ARRAY[$1,$2]::uuid[]) AND length(hash) = 64`, original, reversal).
			Scan(&hashes); err != nil {
			return err
		}
		if hashes != 2 {
			t.Fatalf("esperava hash nos dois lancamentos, veio %d", hashes)
		}

		// sign/reversal sao imutaveis (guard append-only).
		expectSavepointError(t, tx, "reversal_sign_guard", "append-only", func() error {
			_, err := tx.Exec(testCtx,
				`UPDATE financial_transactions SET sign = 1 WHERE id = $1::uuid`, reversal)
			return err
		})

		// Devolucao/estorno nao entra em agrupamento.
		if _, err := tx.Exec(testCtx, `
			INSERT INTO financial_transaction_groups
				(tenant_id, branch_id, account_id, type, occurred_at, description)
			VALUES ($1::uuid, $2::uuid, $3::uuid, 'income', $4::date, 'grupo teste')`,
			fixTenantX, fixBranchA, fixAccountA, date); err != nil {
			return err
		}
		var grpID string
		if err := tx.QueryRow(testCtx, `
			SELECT id::text FROM financial_transaction_groups
			WHERE tenant_id = $1::uuid AND description = 'grupo teste'`, fixTenantX).Scan(&grpID); err != nil {
			return err
		}
		expectSavepointError(t, tx, "reversal_group_guard", "devolucao", func() error {
			_, err := tx.Exec(testCtx, `SELECT fin_group_attach($1::uuid, ARRAY[$2]::uuid[])`, grpID, reversal)
			return err
		})

		return nil
	})
	if err != nil {
		t.Fatalf("inBounds: %v", err)
	}
}
