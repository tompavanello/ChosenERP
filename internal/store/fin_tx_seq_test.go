package store

import (
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
)

// seqOrder devolve as descriptions do escopo (conta+data), na ordem da
// sequencia, para as linhas criadas por este teste (prefixo "seq").
func seqOrder(t *testing.T, tx pgx.Tx, accountID, date string) []string {
	t.Helper()
	rows, err := tx.Query(testCtx, `
		SELECT description FROM financial_transactions
		WHERE tenant_id = $1::uuid AND account_id = $2::uuid
		  AND (occurred_at AT TIME ZONE 'UTC')::date = $3::date
		  AND description LIKE 'seq%'
		ORDER BY entry_seq, created_at, id`, fixTenantX, accountID, date)
	if err != nil {
		t.Fatalf("consulta sequencia: %v", err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var d string
		if err := rows.Scan(&d); err != nil {
			t.Fatalf("scan: %v", err)
		}
		out = append(out, d)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows.Err: %v", err)
	}
	return out
}

func sameStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// TestFinancialTransactions_EntrySeq cobre a sequencia do lancamento (migracao
// 000072) no nivel do banco: auto-numeracao por escopo (filial+conta+data),
// insercao no meio (empurrando os demais via UPDATE de entry_seq) e reordenacao.
// Confirma tambem que o guard append-only permite alterar somente entry_seq e
// continua bloqueando os demais campos.
func TestFinancialTransactions_EntrySeq(t *testing.T) {
	const date = "2025-03-10"
	const date2 = "2025-03-11"
	repo := &seqRepo{t: t}

	err := inBounds(t, bounds(fixTenantX, fixBranchA, "tesoureiro"), func(tx pgx.Tx) error {
		// Escopo inicial: 1..3.
		repo.insert(tx, "seq A", fixAccountA, date, 1)
		repo.insert(tx, "seq B", fixAccountA, date, 2)
		repo.insert(tx, "seq C", fixAccountA, date, 3)

		// Insere na posicao 2: empurra B(2->3) e C(3->4).
		if _, err := tx.Exec(testCtx, `
			UPDATE financial_transactions
			SET entry_seq = entry_seq + 1
			WHERE tenant_id = $1::uuid AND branch_id = $2::uuid AND account_id = $3::uuid
			  AND (occurred_at AT TIME ZONE 'UTC')::date = $4::date
			  AND entry_seq >= 2`, fixTenantX, fixBranchA, fixAccountA, date); err != nil {
			return err
		}
		repo.insert(tx, "seq D", fixAccountA, date, 2)

		if got := seqOrder(t, tx, fixAccountA, date); !sameStrings(got, []string{"seq A", "seq D", "seq B", "seq C"}) {
			t.Errorf("apos insercao no meio: %v", got)
		}

		// Outro dia e outra conta tem sequencia independente.
		repo.insert(tx, "seq E", fixAccountB, date, 1)
		repo.insert(tx, "seq F", fixAccountA, date2, 1)
		if got := seqOrder(t, tx, fixAccountB, date); !sameStrings(got, []string{"seq E"}) {
			t.Errorf("outra conta: %v", got)
		}
		if got := seqOrder(t, tx, fixAccountA, date2); !sameStrings(got, []string{"seq F"}) {
			t.Errorf("outro dia: %v", got)
		}

		// Reordenacao do escopo A para C, A, B, D.
		ids := map[string]string{}
		rows, err := tx.Query(testCtx, `
			SELECT description, id::text FROM financial_transactions
			WHERE tenant_id = $1::uuid AND account_id = $2::uuid
			  AND (occurred_at AT TIME ZONE 'UTC')::date = $3::date AND description LIKE 'seq%'`,
			fixTenantX, fixAccountA, date)
		if err != nil {
			return err
		}
		for rows.Next() {
			var d, id string
			if err := rows.Scan(&d, &id); err != nil {
				rows.Close()
				return err
			}
			ids[d] = id
		}
		rows.Close()
		for i, d := range []string{"seq C", "seq A", "seq B", "seq D"} {
			if _, err := tx.Exec(testCtx,
				`UPDATE financial_transactions SET entry_seq = $2 WHERE id = $1::uuid`, ids[d], i+1); err != nil {
				return err
			}
		}
		if got := seqOrder(t, tx, fixAccountA, date); !sameStrings(got, []string{"seq C", "seq A", "seq B", "seq D"}) {
			t.Errorf("apos reorder: %v", got)
		}

		// O guard continua bloqueando alteracao de campo protegido (isolado por
		// SAVEPOINT para nao abortar a transacao do teste)...
		if _, err := tx.Exec(testCtx, `SAVEPOINT guard_test`); err != nil {
			return err
		}
		_, guardErr := tx.Exec(testCtx,
			`UPDATE financial_transactions SET description = 'hack' WHERE id = $1::uuid`, ids["seq A"])
		if _, err := tx.Exec(testCtx, `ROLLBACK TO SAVEPOINT guard_test`); err != nil {
			return err
		}
		if guardErr == nil || !strings.Contains(guardErr.Error(), "append-only") {
			t.Errorf("alteracao de description deveria ser bloqueada, veio: %v", guardErr)
		}
		// ...e permite alterar apenas entry_seq.
		if _, err := tx.Exec(testCtx,
			`UPDATE financial_transactions SET entry_seq = entry_seq + 0 WHERE id = $1::uuid`, ids["seq A"]); err != nil {
			t.Errorf("alteracao apenas de entry_seq deveria ser aceita: %v", err)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("WithTenant: %v", err)
	}
}

type seqRepo struct{ t *testing.T }

func (r *seqRepo) insert(tx pgx.Tx, desc, accountID, date string, seq int) {
	r.t.Helper()
	if _, err := tx.Exec(testCtx, `
		INSERT INTO financial_transactions
			(tenant_id, branch_id, category_id, type, amount, currency, account_id, description, occurred_at, entry_seq)
		VALUES ($1, $2, $3, 'income', 10, 'BRL', $4, $5, $6::date, $7)`,
		fixTenantX, fixBranchA, fixCatA, accountID, desc, date, seq); err != nil {
		r.t.Fatalf("insert %s: %v", desc, err)
	}
}
