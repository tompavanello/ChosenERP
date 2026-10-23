package store

import (
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
)

// TestFinancialTransactions_RequireAccount garante a regra da migracao 000103:
// todo lancamento novo precisa de conta bancaria vinculada (o historico legado
// sem conta continua intacto porque a regra so vale no INSERT).
func TestFinancialTransactions_RequireAccount(t *testing.T) {
	var execErr error
	if err := inBounds(t, bounds(fixTenantX, fixBranchA, "tesoureiro"), func(tx pgx.Tx) error {
		_, execErr = tx.Exec(testCtx, `
			INSERT INTO financial_transactions (tenant_id, branch_id, category_id, type, amount, currency)
			VALUES ($1, $2, $3, 'income', 10.00, 'BRL')`, fixTenantX, fixBranchA, fixCatA)
		return errRollback
	}); err != nil {
		t.Fatalf("WithTenant: %v", err)
	}
	if execErr == nil {
		t.Fatal("lancamento sem conta bancaria foi aceito")
	}
	if !strings.Contains(execErr.Error(), "conta bancaria") {
		t.Fatalf("erro inesperado (esperado conta obrigatoria): %v", execErr)
	}
}

// TestFinancialTransactions_RequirePositiveAmount rejeita valor zerado ou
// negativo em qualquer origem (manual, lote, importacao, recorrencia, webhook).
func TestFinancialTransactions_RequirePositiveAmount(t *testing.T) {
	for _, amount := range []string{"0", "-1"} {
		var execErr error
		if err := inBounds(t, bounds(fixTenantX, fixBranchA, "tesoureiro"), func(tx pgx.Tx) error {
			_, execErr = tx.Exec(testCtx, `
				INSERT INTO financial_transactions
					(tenant_id, branch_id, category_id, type, amount, currency, account_id)
				VALUES ($1, $2, $3, 'income', $4::numeric, 'BRL', $5)`,
				fixTenantX, fixBranchA, fixCatA, amount, fixAccountA)
			return errRollback
		}); err != nil {
			t.Fatalf("WithTenant (amount=%s): %v", amount, err)
		}
		if execErr == nil {
			t.Fatalf("lancamento com valor %s foi aceito", amount)
		}
		if !strings.Contains(execErr.Error(), "maior que zero") {
			t.Fatalf("erro inesperado (amount=%s): %v", amount, execErr)
		}
	}
}
