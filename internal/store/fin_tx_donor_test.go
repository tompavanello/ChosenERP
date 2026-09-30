package store

import (
	"testing"

	"github.com/jackc/pgx/v5"
)

// TestFinancialTransactions_DonorNameChain cobre o doador avulso (migracao
// 000071): donor_name e persistido junto do vinculo com o membro e participa da
// hash-chain. Depois de um recadeamento, cada prev_hash deve bater com o hash do
// lancamento anterior (a versao anterior do rechain encadeava o hash antigo e
// quebraria a cadeia ao mudar a formula).
func TestFinancialTransactions_DonorNameChain(t *testing.T) {
	err := inBounds(t, bounds(fixTenantX, "", "super_admin"), func(tx pgx.Tx) error {
		var gotName, gotMember string
		if err := tx.QueryRow(testCtx, `
			INSERT INTO financial_transactions
				(tenant_id, branch_id, category_id, type, amount, currency, donor_member_id, donor_name)
			VALUES ($1, $2, $3, 'income', 25.00, 'BRL', $4, $5)
			RETURNING donor_name, donor_member_id::text`,
			fixTenantX, fixBranchA, fixCatA, fixMemberA1, "Visitante avulso").Scan(&gotName, &gotMember); err != nil {
			t.Fatalf("insert com doador avulso: %v", err)
		}
		if gotName != "Visitante avulso" || gotMember != fixMemberA1 {
			t.Errorf("doador nao persistiu: name=%q member=%q", gotName, gotMember)
		}

		if _, err := tx.Exec(testCtx, `SELECT fin_tx_rechain($1::uuid)`, fixTenantX); err != nil {
			t.Fatalf("fin_tx_rechain falhou: %v", err)
		}

		var broken int
		if err := tx.QueryRow(testCtx, `
			SELECT count(*) FROM (
				SELECT prev_hash, lag(hash) OVER (ORDER BY created_at, id) AS expected
				FROM financial_transactions
				WHERE tenant_id = $1::uuid
			) s
			WHERE prev_hash IS DISTINCT FROM expected`, fixTenantX).Scan(&broken); err != nil {
			return err
		}
		if broken != 0 {
			t.Errorf("hash-chain inconsistente apos recadeamento: %d elo(s) quebrado(s)", broken)
		}
		return errRollback
	})
	if err != nil {
		t.Fatalf("WithTenant: %v", err)
	}
}
