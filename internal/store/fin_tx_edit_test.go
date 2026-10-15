package store

import (
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
)

// TestFinancialTransactions_EditInPlace cobre a edicao em vigor (migracao
// 000096): fin_tx_edit altera os campos do lancamento sem tocar em id/entry_seq,
// recalcula a hash-chain e mantem o guard append-only para UPDATE direto.
func TestFinancialTransactions_EditInPlace(t *testing.T) {
	const txID = "e0000000-0000-4000-8000-0000000000e1"
	const date = "2025-04-10"

	err := inBounds(t, bounds(fixTenantX, fixBranchA, "tesoureiro"), func(tx pgx.Tx) error {
		if _, err := tx.Exec(testCtx, `
			INSERT INTO financial_transactions
				(id, tenant_id, branch_id, category_id, type, amount, currency,
				 account_id, description, occurred_at, entry_seq)
			VALUES ($1::uuid, $2::uuid, $3::uuid, $4::uuid, 'income', 100, 'BRL',
				$5::uuid, 'antes', $6::date, 2)`,
			txID, fixTenantX, fixBranchA, fixCatA, fixAccountA, date); err != nil {
			return err
		}

		var hashBefore string
		if err := tx.QueryRow(testCtx,
			`SELECT hash FROM financial_transactions WHERE id = $1::uuid`, txID).Scan(&hashBefore); err != nil {
			return err
		}

		// Edita: valor 250, descricao "depois". Mesma conta/data.
		if _, err := tx.Exec(testCtx, `
			SELECT fin_tx_edit($1::uuid, $2, $3, $4::numeric, $5, $6, $7, $8, $9,
				$10, $11, $12::timestamptz, $13, $14, $15, $16)`,
			txID, fixCatA, "income", 250.0, "pix", "", "", "", "",
			false, "depois", date, fixAccountA, "", "", false); err != nil {
			return err
		}

		var idOut, desc, hashAfter string
		var amount float64
		var seq *int
		if err := tx.QueryRow(testCtx, `
			SELECT id::text, amount::float8, COALESCE(description,''), entry_seq, hash
			FROM financial_transactions WHERE id = $1::uuid`, txID).
			Scan(&idOut, &amount, &desc, &seq, &hashAfter); err != nil {
			return err
		}
		if idOut != txID {
			t.Errorf("id mudou: %s", idOut)
		}
		if amount != 250 {
			t.Errorf("amount = %v, queria 250", amount)
		}
		if desc != "depois" {
			t.Errorf("descricao = %q, queria \"depois\"", desc)
		}
		if seq == nil || *seq != 2 {
			t.Errorf("entry_seq mudou: %v, queria 2", seq)
		}
		if hashAfter == hashBefore {
			t.Errorf("hash nao foi recalculado")
		}

		// UPDATE direto do papel da aplicacao continua bloqueado.
		if _, err := tx.Exec(testCtx, `SAVEPOINT guard_edit`); err != nil {
			return err
		}
		_, guardErr := tx.Exec(testCtx,
			`UPDATE financial_transactions SET amount = 999 WHERE id = $1::uuid`, txID)
		if _, err := tx.Exec(testCtx, `ROLLBACK TO SAVEPOINT guard_edit`); err != nil {
			return err
		}
		if guardErr == nil || !strings.Contains(guardErr.Error(), "append-only") {
			t.Errorf("UPDATE direto deveria ser bloqueado pelo guard, veio: %v", guardErr)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("inBounds: %v", err)
	}
}
