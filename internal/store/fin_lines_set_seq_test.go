package store

import (
	"testing"

	"github.com/jackc/pgx/v5"
)

// TestFinLinesSetSeq_ReconciledAndGroup confirma que a RPC de reordenacao
// (migracao 000107) altera somente o entry_seq e funciona tambem para linhas
// ja conciliadas (lancamento avulso e grupo), preservando a conciliacao.
func TestFinLinesSetSeq_ReconciledAndGroup(t *testing.T) {
	const date = "2025-04-15"

	err := inBounds(t, bounds(fixTenantX, fixBranchA, "tesoureiro"), func(tx pgx.Tx) error {
		insert := func(desc string, seq int) string {
			t.Helper()
			var id string
			if err := tx.QueryRow(testCtx, `
				INSERT INTO financial_transactions
					(tenant_id, branch_id, category_id, type, amount, currency, account_id, description, occurred_at, entry_seq)
				VALUES ($1, $2, $3, 'income', 10, 'BRL', $4, $5, $6::date, $7)
				RETURNING id::text`,
				fixTenantX, fixBranchA, fixCatA, fixAccountA, desc, date, seq).Scan(&id); err != nil {
				t.Fatalf("insert %s: %v", desc, err)
			}
			return id
		}

		// Lancamento conciliado pode ser reordenado (apenas entry_seq muda).
		idA := insert("seq A", 1)
		idB := insert("seq B", 2)
		if _, err := tx.Exec(testCtx, `SELECT fin_tx_reconcile($1::uuid, true, '')`, idA); err != nil {
			t.Fatalf("conciliar A: %v", err)
		}
		if _, err := tx.Exec(testCtx, `SELECT fin_lines_set_seq('transaction', $1::uuid, 2)`, idA); err != nil {
			t.Fatalf("reordenar lancamento conciliado: %v", err)
		}
		var seq int
		var reconciled bool
		if err := tx.QueryRow(testCtx, `
			SELECT entry_seq, reconciled_at IS NOT NULL
			FROM financial_transactions WHERE id = $1::uuid`, idA).Scan(&seq, &reconciled); err != nil {
			t.Fatalf("ler A: %v", err)
		}
		if seq != 2 || !reconciled {
			t.Errorf("A: seq=%d reconciled=%v", seq, reconciled)
		}

		// Grupo conciliado tambem pode ser reordenado.
		idC := insert("seq C", 3)
		var groupID string
		if err := tx.QueryRow(testCtx, `
			INSERT INTO financial_transaction_groups
				(tenant_id, branch_id, account_id, type, occurred_at, description, entry_seq)
			VALUES ($1, $2, $3, 'income', $4::date, 'grp', 5)
			RETURNING id::text`,
			fixTenantX, fixBranchA, fixAccountA, date).Scan(&groupID); err != nil {
			t.Fatalf("insert grupo: %v", err)
		}
		if _, err := tx.Exec(testCtx, `SELECT fin_group_attach($1::uuid, $2::uuid[])`, groupID, []string{idB, idC}); err != nil {
			t.Fatalf("anexar grupo: %v", err)
		}
		if _, err := tx.Exec(testCtx, `SELECT fin_group_reconcile($1::uuid, true, '')`, groupID); err != nil {
			t.Fatalf("conciliar grupo: %v", err)
		}
		if _, err := tx.Exec(testCtx, `SELECT fin_lines_set_seq('group', $1::uuid, 1)`, groupID); err != nil {
			t.Fatalf("reordenar grupo conciliado: %v", err)
		}
		var gseq *int
		var status string
		if err := tx.QueryRow(testCtx, `
			SELECT entry_seq, status FROM financial_transaction_groups WHERE id = $1::uuid`, groupID).
			Scan(&gseq, &status); err != nil {
			t.Fatalf("ler grupo: %v", err)
		}
		if gseq == nil || *gseq != 1 || status != "conciliado" {
			t.Errorf("grupo: seq=%v status=%s", gseq, status)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("WithTenant: %v", err)
	}
}
