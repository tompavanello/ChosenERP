package store

import (
	"fmt"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
)

// ---------------------------------------------------------------------------
// LEITURA - isolamento por filial
// ---------------------------------------------------------------------------

func TestRLS_Members_ScopedToBranch(t *testing.T) {
	cases := []struct {
		name   string
		bounds Bounds
		want   []string
	}{
		{
			name:   "filial A ve apenas seus membros",
			bounds: bounds(fixTenantX, fixBranchA, "secretario"),
			want:   []string{fixMemberA1, fixMemberA2},
		},
		{
			name:   "filial B ve apenas seus membros",
			bounds: bounds(fixTenantX, fixBranchB, "secretario"),
			want:   []string{fixMemberB1, fixMemberB2},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := inBounds(t, tc.bounds, func(tx pgx.Tx) error {
				got := ids(t, tx, `SELECT id FROM members ORDER BY id`)
				if len(got) != len(tc.want) {
					t.Errorf("leu %d membros, queria %d", len(got), len(tc.want))
				}
				for _, w := range tc.want {
					if !contains(got, w) {
						t.Errorf("membro %s nao foi retornado", w)
					}
				}
				return nil
			})
			if err != nil {
				t.Fatalf("WithTenant: %v", err)
			}
		})
	}
}

func TestRLS_Members_CrossBranchByID(t *testing.T) {
	err := inBounds(t, bounds(fixTenantX, fixBranchA, "secretario"), func(tx pgx.Tx) error {
		got := count(t, tx, `SELECT count(*) FROM members WHERE id = $1`, fixMemberB1)
		if got != 0 {
			t.Errorf("vazamento entre filiais: filial A leu membro da filial B por id")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("WithTenant: %v", err)
	}
}

func TestRLS_Members_HeadquartersSeesAllBranchesOfTenant(t *testing.T) {
	err := inBounds(t, bounds(fixTenantX, "", "super_admin"), func(tx pgx.Tx) error {
		got := count(t, tx, `SELECT count(*) FROM members`)
		if got != 4 {
			t.Errorf("sede leu %d membros, queria 4 (filiais A e B do tenant X)", got)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("WithTenant: %v", err)
	}
}

func TestRLS_Members_CrossTenantBlocked(t *testing.T) {
	// Sede do tenant Y (branch vazio + papel super_admin).
	err := inBounds(t, bounds(fixTenantY, "", "super_admin"), func(tx pgx.Tx) error {
		got := count(t, tx, `SELECT count(*) FROM members`)
		if got != 1 {
			t.Errorf("vazamento entre tenants: sede de Y leu %d membros, queria 1 (apenas o de Y)", got)
		}
		if contains(ids(t, tx, `SELECT id FROM members`), fixMemberA1) {
			t.Errorf("vazamento entre tenants: sede de Y leu membro do tenant X")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("WithTenant: %v", err)
	}
}

// ---------------------------------------------------------------------------
// LEITURA - financeiro e relatorios
// ---------------------------------------------------------------------------

func TestRLS_Finance_ScopedToBranch(t *testing.T) {
	err := inBounds(t, bounds(fixTenantX, fixBranchA, "tesoureiro"), func(tx pgx.Tx) error {
		if got := count(t, tx, `SELECT count(*) FROM financial_transactions`); got != 1 {
			t.Errorf("filial A leu %d lancamentos, queria 1", got)
		}
		if got := count(t, tx, `SELECT count(*) FROM financial_transactions WHERE id = $1`, fixTxnB); got != 0 {
			t.Errorf("vazamento: filial A leu lancamento da filial B")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("WithTenant: %v", err)
	}
}

// Os relatorios agregam por periodo; a RLS precisa valer tambem para eles
// (o checkpoint validou isso manualmente para a filial Norte).
func TestRLS_ReportsAggregatesScopedToBranch(t *testing.T) {
	err := inBounds(t, bounds(fixTenantX, fixBranchB, "tesoureiro"), func(tx pgx.Tx) error {
		var total string
		q := `SELECT COALESCE(SUM(amount) FILTER (WHERE type <> 'expense'), 0)::text
		      FROM financial_transactions`
		if err := tx.QueryRow(testCtx, q).Scan(&total); err != nil {
			t.Fatalf("agregacao falhou: %v", err)
		}
		if !strings.HasPrefix(total, "250") {
			t.Errorf("soma da filial B = %s, queria 250.00 (sem o lancamento de A)", total)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("WithTenant: %v", err)
	}
}

func TestRLS_Documents_ScopedToBranch(t *testing.T) {
	err := inBounds(t, bounds(fixTenantX, fixBranchA, "secretario"), func(tx pgx.Tx) error {
		if got := count(t, tx, `SELECT count(*) FROM documents`); got != 1 {
			t.Errorf("filial A leu %d documentos, queria 1", got)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("WithTenant: %v", err)
	}

	err = inBounds(t, bounds(fixTenantX, fixBranchB, "secretario"), func(tx pgx.Tx) error {
		if got := count(t, tx, `SELECT count(*) FROM documents`); got != 0 {
			t.Errorf("vazamento: filial B leu %d documentos da filial A", got)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("WithTenant: %v", err)
	}
}

// ---------------------------------------------------------------------------
// LEITURA - audit_log (somente Sede/sistema)
// ---------------------------------------------------------------------------

func TestRLS_AuditLog_BranchRoleCannotRead(t *testing.T) {
	for _, role := range []string{"pastor_filial", "tesoureiro", "secretario"} {
		t.Run(role, func(t *testing.T) {
			err := inBounds(t, bounds(fixTenantX, fixBranchA, role), func(tx pgx.Tx) error {
				if got := count(t, tx, `SELECT count(*) FROM audit_log`); got != 0 {
					t.Errorf("perfil %s leu %d linhas do audit_log, queria 0", role, got)
				}
				return nil
			})
			if err != nil {
				t.Fatalf("WithTenant: %v", err)
			}
		})
	}
}

func TestRLS_AuditLog_HeadquartersSeesOnlyOwnTenant(t *testing.T) {
	err := inBounds(t, bounds(fixTenantY, "", "super_admin"), func(tx pgx.Tx) error {
		if got := count(t, tx, `SELECT count(*) FROM audit_log`); got != 1 {
			t.Errorf("vazamento entre tenants: sede de Y leu %d linhas do audit_log, queria 1", got)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("WithTenant: %v", err)
	}
}

// ---------------------------------------------------------------------------
// ESCRITA - cross-branch bloqueado
// ---------------------------------------------------------------------------

// expectRLSDenied roda stmt dentro do escopo dado e exige que o PostgreSQL
// recuse a operacao por violacao de politica de RLS.
func expectRLSDenied(t *testing.T, b Bounds, stmt string, args ...any) {
	t.Helper()
	var execErr error
	if err := inBounds(t, b, func(tx pgx.Tx) error {
		_, execErr = tx.Exec(testCtx, stmt, args...)
		// A transacao fica abortada depois do erro; apenas encerramos.
		return errRollback
	}); err != nil {
		t.Fatalf("WithTenant: %v", err)
	}
	if execErr == nil {
		t.Fatal("operacao cross-branch foi aceita; a politica WITH CHECK nao barrou")
	}
	if !strings.Contains(execErr.Error(), "row-level security") {
		t.Fatalf("erro inesperado (esperada violacao de RLS): %v", execErr)
	}
}

func TestRLS_Members_InsertCrossBranchBlocked(t *testing.T) {
	expectRLSDenied(t,
		bounds(fixTenantX, fixBranchA, "secretario"),
		`INSERT INTO members (tenant_id, branch_id, first_name, last_name, full_name)
		 VALUES ($1, $2, 'Invasor', 'X', 'Invasor X')`,
		fixTenantX, fixBranchB)
}

func TestRLS_Members_UpdateCrossBranchNoEffect(t *testing.T) {
	err := inBounds(t, bounds(fixTenantX, fixBranchA, "secretario"), func(tx pgx.Tx) error {
		tag, err := tx.Exec(testCtx,
			`UPDATE members SET full_name = 'Sequestrado' WHERE id = $1`, fixMemberB1)
		if err != nil {
			t.Fatalf("UPDATE cross-branch deveria ser silencioso: %v", err)
		}
		if tag.RowsAffected() != 0 {
			t.Errorf("UPDATE cross-branch afetou %d linha(s), queria 0", tag.RowsAffected())
		}
		return nil
	})
	if err != nil {
		t.Fatalf("WithTenant: %v", err)
	}

	// Confere, fora do RLS, que o registro da filial B continua intacto.
	withSuperuser(t, func(conn *pgx.Conn) error {
		var name string
		if err := conn.QueryRow(testCtx,
			`SELECT full_name FROM members WHERE id = $1`, fixMemberB1).Scan(&name); err != nil {
			t.Fatalf("ler membro B1: %v", err)
		}
		if name != "Bruno B" {
			t.Errorf("membro B1 foi alterado para %q", name)
		}
		return nil
	})
}

func TestRLS_Members_DeleteCrossBranchNoEffect(t *testing.T) {
	err := inBounds(t, bounds(fixTenantX, fixBranchA, "secretario"), func(tx pgx.Tx) error {
		tag, err := tx.Exec(testCtx, `DELETE FROM members WHERE id = $1`, fixMemberB1)
		if err != nil {
			t.Fatalf("DELETE cross-branch deveria ser silencioso: %v", err)
		}
		if tag.RowsAffected() != 0 {
			t.Errorf("DELETE cross-branch afetou %d linha(s), queria 0", tag.RowsAffected())
		}
		return nil
	})
	if err != nil {
		t.Fatalf("WithTenant: %v", err)
	}
}

func TestRLS_Finance_InsertCrossBranchBlocked(t *testing.T) {
	expectRLSDenied(t,
		bounds(fixTenantX, fixBranchA, "tesoureiro"),
		`INSERT INTO financial_transactions
		 (tenant_id, branch_id, type, amount, currency, payment_method)
		 VALUES ($1, $2, 'oferta', 999.99, 'BRL', 'pix')`,
		fixTenantX, fixBranchB)
}

func TestRLS_Transfers_InsertRequiresParticipation(t *testing.T) {
	// A filial C nao e origem nem destino do repasse A -> B.
	expectRLSDenied(t,
		bounds(fixTenantX, fixBranchC, "tesoureiro"),
		`INSERT INTO transfers (tenant_id, from_branch_id, to_branch_id, amount, rule_name)
		 VALUES ($1, $2, $3, 50.00, '10% sede')`,
		fixTenantX, fixBranchA, fixBranchB)
}

func TestRLS_Recurring_InsertCrossBranchBlocked(t *testing.T) {
	expectRLSDenied(t,
		bounds(fixTenantX, fixBranchA, "tesoureiro"),
		`INSERT INTO recurring_donations (tenant_id, branch_id, amount, subtype, frequency, next_run_at)
		 VALUES ($1, $2, 30.00, 'dizimo', 'monthly', now())`,
		fixTenantX, fixBranchB)
}

// ---------------------------------------------------------------------------
// Contas bancarias e classificacoes (tabelas novas da reestrutura 000018)
// ---------------------------------------------------------------------------

func TestRLS_FinancialAccounts_ScopedToBranch(t *testing.T) {
	err := inBounds(t, bounds(fixTenantX, fixBranchA, "tesoureiro"), func(tx pgx.Tx) error {
		if got := count(t, tx, `SELECT count(*) FROM financial_accounts`); got != 1 {
			t.Errorf("filial A leu %d contas, queria 1", got)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("WithTenant: %v", err)
	}
}

func TestRLS_FinancialAccounts_InsertCrossBranchBlocked(t *testing.T) {
	expectRLSDenied(t,
		bounds(fixTenantX, fixBranchA, "tesoureiro"),
		`INSERT INTO financial_accounts (tenant_id, branch_id, name, account_type)
		 VALUES ($1, $2, 'Conta Invadida', 'checking')`,
		fixTenantX, fixBranchB)
}

func TestRLS_Finance_TransactionsAssociatedWithAccount(t *testing.T) {
	// Branch A ve sua conta associada ao lancamento.
	err := inBounds(t, bounds(fixTenantX, fixBranchA, "tesoureiro"), func(tx pgx.Tx) error {
		var acctID string
		if err := tx.QueryRow(testCtx,
			`SELECT account_id::text FROM financial_transactions WHERE id = $1`, fixTxnA).Scan(&acctID); err != nil {
			t.Fatalf("ler account_id: %v", err)
		}
		if acctID != fixAccountA {
			t.Errorf("account_id do lancamento A = %s, queria %s", acctID, fixAccountA)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("WithTenant: %v", err)
	}
}

func TestRLS_FinancialAttachments_ScopedToBranch(t *testing.T) {
	// Branch A ve os anexos de seus lancamentos.
	err := inBounds(t, bounds(fixTenantX, fixBranchA, "tesoureiro"), func(tx pgx.Tx) error {
		if got := count(t, tx, `SELECT count(*) FROM financial_attachments`); got != 1 {
			t.Errorf("filial A leu %d anexos, queria 1", got)
		}
		// Nao ve anexos de lancamentos da filial B.
		if got := count(t, tx, `SELECT count(*) FROM financial_attachments fa JOIN financial_transactions t ON t.id = fa.transaction_id WHERE t.branch_id = $1`, fixBranchB); got != 0 {
			t.Errorf("vazamento: filial A leu %d anexos da filial B", got)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("WithTenant: %v", err)
	}
}

// TestFinancialTransactions_HashWithNullPaymentMethod cobre a regressao do
// trigger de hash-chain: payment_method e anulavel e, sem COALESCE, um lancamento
// sem forma de pagamento gerava hash NULL e quebrava o NOT NULL da coluna.
func TestFinancialTransactions_HashWithNullPaymentMethod(t *testing.T) {
	err := inBounds(t, bounds(fixTenantX, fixBranchA, "tesoureiro"), func(tx pgx.Tx) error {
		var hasHash bool
		if err := tx.QueryRow(testCtx, `
			INSERT INTO financial_transactions (tenant_id, branch_id, category_id, type, amount, currency)
			VALUES ($1, $2, $3, 'income', 1.00, 'BRL')
			RETURNING hash IS NOT NULL`, fixTenantX, fixBranchA, fixCatA).Scan(&hasHash); err != nil {
			t.Fatalf("insert sem forma de pagamento: %v", err)
		}
		if !hasHash {
			t.Errorf("hash ficou nulo quando payment_method e NULL")
		}
		return errRollback
	})
	if err != nil {
		t.Fatalf("WithTenant: %v", err)
	}
}

func TestAppendOnly_FinancialTransactions_UpdateBlocked(t *testing.T) {
	var execErr error
	if err := inBounds(t, bounds(fixTenantX, fixBranchA, "tesoureiro"), func(tx pgx.Tx) error {
		_, execErr = tx.Exec(testCtx,
			`UPDATE financial_transactions SET description = 'alterado' WHERE id = $1`, fixTxnA)
		return errRollback
	}); err != nil {
		t.Fatalf("WithTenant: %v", err)
	}
	if execErr == nil {
		t.Fatal("UPDATE em lancamento foi aceito; a tabela deve ser append-only")
	}
	if !strings.Contains(execErr.Error(), "append-only") {
		t.Fatalf("erro inesperado (esperado 'append-only'): %v", execErr)
	}
}

func TestAppendOnly_FinancialTransactions_DeleteBlocked(t *testing.T) {
	var execErr error
	if err := inBounds(t, bounds(fixTenantX, fixBranchA, "tesoureiro"), func(tx pgx.Tx) error {
		_, execErr = tx.Exec(testCtx,
			`DELETE FROM financial_transactions WHERE id = $1`, fixTxnA)
		return errRollback
	}); err != nil {
		t.Fatalf("WithTenant: %v", err)
	}
	if execErr == nil {
		t.Fatal("DELETE em lancamento foi aceito; a tabela deve ser append-only")
	}
	if !strings.Contains(execErr.Error(), "append-only") {
		t.Fatalf("erro inesperado (esperado 'append-only'): %v", execErr)
	}
}

func TestAppendOnly_AuditLog(t *testing.T) {
	// Pelo papel da aplicacao nao existe politica de UPDATE/DELETE: a RLS
	// simplesmente nao devolve linhas (nem mesmo para a Sede).
	err := inBounds(t, bounds(fixTenantX, "", "super_admin"), func(tx pgx.Tx) error {
		tag, err := tx.Exec(testCtx, `UPDATE audit_log SET action = 'hack' WHERE tenant_id = $1`, fixTenantX)
		if err != nil {
			t.Fatalf("UPDATE em audit_log deveria ser filtrado pela RLS: %v", err)
		}
		if tag.RowsAffected() != 0 {
			t.Errorf("UPDATE em audit_log afetou %d linha(s), queria 0", tag.RowsAffected())
		}
		return errRollback
	})
	if err != nil {
		t.Fatalf("WithTenant: %v", err)
	}

	// Mesmo burlando a RLS (superuser), o trigger de imutabilidade barra.
	withSuperuser(t, func(conn *pgx.Conn) error {
		if _, err := conn.Exec(testCtx, `UPDATE audit_log SET action = 'hack'`); err == nil {
			t.Errorf("superuser conseguiu dar UPDATE em audit_log; trigger nao barrou")
		}
		if _, err := conn.Exec(testCtx, `DELETE FROM audit_log`); err == nil {
			t.Errorf("superuser conseguiu dar DELETE em audit_log; trigger nao barrou")
		}
		return nil
	})
}

// ---------------------------------------------------------------------------
// ESCOPO DE SISTEMA (workers internos)
// ---------------------------------------------------------------------------

// WithSystem e usado pelos workers (outbox de envio, recorrencias). Ele
// enxerga todos os tenants de proposito - as politicas is_headquarters()
// incluem a role 'system'. Este teste fixa esse comportamento.
func TestRLS_SystemScopeSeesAllTenants(t *testing.T) {
	err := inSystem(t, func(tx pgx.Tx) error {
		if got := count(t, tx, `SELECT count(*) FROM members`); got != 5 {
			t.Errorf("escopo de sistema leu %d membros, queria 5 (todos os tenants)", got)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("WithSystem: %v", err)
	}
}

// ---------------------------------------------------------------------------
// REGRESSAO - toda tabela tenant-scoped precisa ter RLS
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// REGRESSAO - varredura de TODAS as tabelas tenant-scoped
// ---------------------------------------------------------------------------

// columnsLike devolve as tabelas do schema public que possuem a coluna dada.
func columnsLike(t *testing.T, column string) []string {
	t.Helper()
	var out []string
	withSuperuser(t, func(conn *pgx.Conn) error {
		rows, err := conn.Query(testCtx, `
			SELECT table_name FROM information_schema.columns
			WHERE table_schema = 'public' AND column_name = $1
			ORDER BY table_name`, column)
		if err != nil {
			t.Fatalf("information_schema: %v", err)
		}
		defer rows.Close()
		for rows.Next() {
			var n string
			if err := rows.Scan(&n); err != nil {
				t.Fatalf("scan: %v", err)
			}
			out = append(out, n)
		}
		return rows.Err()
	})
	return out
}

// TestRLS_NoCrossTenantReadForAnyTable e a rede de seguranca contra o vazamento
// corrigido pela migracao 000016: um usuario com escopo "Sede" de um tenant nao
// pode ler linhas de OUTRO tenant em tabela alguma (com ou sem branch_id).
func TestRLS_NoCrossTenantReadForAnyTable(t *testing.T) {
	for _, table := range columnsLike(t, "tenant_id") {
		t.Run(table, func(t *testing.T) {
			q := fmt.Sprintf(`SELECT count(*) FROM %s WHERE tenant_id = $1`, table)

			// Sede do tenant Y tentando ler dados do tenant X.
			err := inBounds(t, bounds(fixTenantY, "", "super_admin"), func(tx pgx.Tx) error {
				if got := count(t, tx, q, fixTenantX); got != 0 {
					t.Errorf("vazamento cross-tenant: sede de Y leu %d linha(s) de %s do tenant X", got, table)
				}
				return nil
			})
			if err != nil {
				t.Fatalf("WithTenant(sede Y): %v", err)
			}

			// Filial A tambem nao pode ler o tenant Y.
			err = inBounds(t, bounds(fixTenantX, fixBranchA, "secretario"), func(tx pgx.Tx) error {
				if got := count(t, tx, q, fixTenantY); got != 0 {
					t.Errorf("vazamento cross-tenant: filial A leu %d linha(s) de %s do tenant Y", got, table)
				}
				return nil
			})
			if err != nil {
				t.Fatalf("WithTenant(filial A): %v", err)
			}
		})
	}
}

// TestRLS_NoCrossBranchReadForAnyTable complementa o anterior no eixo filial:
// nenhuma tabela com branch_id devolve linhas de outra filial do mesmo tenant.
func TestRLS_NoCrossBranchReadForAnyTable(t *testing.T) {
	for _, table := range columnsLike(t, "branch_id") {
		t.Run(table, func(t *testing.T) {
			q := fmt.Sprintf(`SELECT count(*) FROM %s WHERE branch_id = $1`, table)

			// A filial A nao ve linhas da filial B...
			err := inBounds(t, bounds(fixTenantX, fixBranchA, "secretario"), func(tx pgx.Tx) error {
				if got := count(t, tx, q, fixBranchB); got != 0 {
					t.Errorf("vazamento entre filiais: filial A leu %d linha(s) de %s da filial B", got, table)
				}
				return nil
			})
			if err != nil {
				t.Fatalf("WithTenant(filial A): %v", err)
			}

			// ...e vice-versa.
			err = inBounds(t, bounds(fixTenantX, fixBranchB, "secretario"), func(tx pgx.Tx) error {
				if got := count(t, tx, q, fixBranchA); got != 0 {
					t.Errorf("vazamento entre filiais: filial B leu %d linha(s) de %s da filial A", got, table)
				}
				return nil
			})
			if err != nil {
				t.Fatalf("WithTenant(filial B): %v", err)
			}
		})
	}
}

// Tabelas de vinculo nao tem tenant_id proprio: herdam o escopo da tabela pai.
func TestRLS_RelationshipTablesInheritTenantScope(t *testing.T) {
	cases := []struct {
		table  string
		column string // coluna pela qual a tabela pai e referenciada
		// valor que aponta para a filial A e valor que aponta para a filial B
		idA, idB string
	}{
		{"member_relationships", "member_id", fixMemberA1, fixMemberB1},
		{"ministry_members", "ministry_id", fixMinistryA, fixMinistryB},
		{"member_cargos", "member_id", fixMemberA1, fixMemberB1},
	}

	for _, tc := range cases {
		t.Run(tc.table, func(t *testing.T) {
			q := fmt.Sprintf(`SELECT count(*) FROM %s WHERE %s = $1`, tc.table, tc.column)

			// A filial A nao ve vinculos da filial B.
			err := inBounds(t, bounds(fixTenantX, fixBranchA, "secretario"), func(tx pgx.Tx) error {
				if got := count(t, tx, q, tc.idB); got != 0 {
					t.Errorf("filial A leu %d vinculo(s) da filial B em %s", got, tc.table)
				}
				return nil
			})
			if err != nil {
				t.Fatalf("WithTenant: %v", err)
			}

			// A Sede de Y nao ve vinculos do tenant X.
			err = inBounds(t, bounds(fixTenantY, "", "super_admin"), func(tx pgx.Tx) error {
				if got := count(t, tx, q, tc.idA); got != 0 {
					t.Errorf("vazamento cross-tenant: sede de Y leu %d vinculo(s) de %s do tenant X", got, tc.table)
				}
				if got := count(t, tx, q, tc.idB); got != 0 {
					t.Errorf("vazamento cross-tenant: sede de Y leu %d vinculo(s) de %s do tenant X", got, tc.table)
				}
				return nil
			})
			if err != nil {
				t.Fatalf("WithTenant: %v", err)
			}
		})
	}
}

// TestRLS_AllTenantScopedTablesHaveRLS garante que nenhuma tabela nova com
// tenant_id/branch_id fique sem Row-Level Security habilitado.
func TestRLS_AllTenantScopedTablesHaveRLS(t *testing.T) {
	withSuperuser(t, func(conn *pgx.Conn) error {
		rows, err := conn.Query(testCtx, `
			SELECT c.relname
			FROM pg_class c
			JOIN pg_namespace n ON n.oid = c.relnamespace
			WHERE n.nspname = 'public'
			  AND c.relkind = 'r'
			  AND NOT c.relrowsecurity
			  AND EXISTS (
			        SELECT 1 FROM information_schema.columns col
			        WHERE col.table_schema = 'public'
			          AND col.table_name   = c.relname
			          AND col.column_name IN ('tenant_id', 'branch_id')
			  )
			ORDER BY c.relname`)
		if err != nil {
			t.Fatalf("consultar pg_class: %v", err)
		}
		defer rows.Close()

		var missing []string
		for rows.Next() {
			var name string
			if err := rows.Scan(&name); err != nil {
				t.Fatalf("scan: %v", err)
			}
			missing = append(missing, name)
		}
		if err := rows.Err(); err != nil {
			t.Fatalf("rows.Err: %v", err)
		}
		if len(missing) > 0 {
			t.Errorf("tabelas tenant-scoped SEM RLS habilitado: %s", strings.Join(missing, ", "))
		}
		return nil
	})
}

// ---------------------------------------------------------------------------
// Cargos (funcoes/ministerios) - migracao 000020
// ---------------------------------------------------------------------------

// TestRLS_CargoScope cobre o eixo que os testes genericos nao alcancam: o cargo
// GLOBAL do tenant (branch_id NULL) precisa ser visivel e gravavel pela filial -
// e o caso de uso real ("criar o cargo uma vez e usar em toda a igreja") - sem
// que isso abra a porteira para o cargo de OUTRA filial ou de outro tenant.
func TestRLS_CargoScope(t *testing.T) {
	// A filial A enxerga o cargo global do tenant e o proprio cargo...
	err := inBounds(t, bounds(fixTenantX, fixBranchA, "secretario"), func(tx pgx.Tx) error {
		visiveis := ids(t, tx, `SELECT id::text FROM cargos ORDER BY id`)
		if !contains(visiveis, fixCargoGlobal) {
			t.Errorf("filial A nao enxergou o cargo global do tenant")
		}
		if !contains(visiveis, fixCargoA) {
			t.Errorf("filial A nao enxergou o proprio cargo")
		}
		// ...e NAO enxerga o da filial B.
		if contains(visiveis, fixCargoB) {
			t.Errorf("vazamento entre filiais: filial A enxergou o cargo da filial B")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("WithTenant(filial A): %v", err)
	}

	// A filial B enxerga o global e o dela, nao o da A.
	err = inBounds(t, bounds(fixTenantX, fixBranchB, "secretario"), func(tx pgx.Tx) error {
		visiveis := ids(t, tx, `SELECT id::text FROM cargos ORDER BY id`)
		if !contains(visiveis, fixCargoGlobal) {
			t.Errorf("filial B nao enxergou o cargo global do tenant")
		}
		if contains(visiveis, fixCargoA) {
			t.Errorf("vazamento entre filiais: filial B enxergou o cargo da filial A")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("WithTenant(filial B): %v", err)
	}

	// A Sede do tenant X enxerga os tres cargos do seu tenant, nenhum de Y.
	err = inBounds(t, bounds(fixTenantX, "", "admin_sede"), func(tx pgx.Tx) error {
		visiveis := ids(t, tx, `SELECT id::text FROM cargos ORDER BY id`)
		for _, want := range []string{fixCargoGlobal, fixCargoA, fixCargoB} {
			if !contains(visiveis, want) {
				t.Errorf("sede de X nao enxergou o cargo %s do proprio tenant", want)
			}
		}
		if contains(visiveis, fixCargoY) {
			t.Errorf("vazamento cross-tenant: sede de X enxergou o cargo do tenant Y")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("WithTenant(sede X): %v", err)
	}
}

// TestRLS_MemberCargoCannotBindForeignCargo e o teste do WITH CHECK da politica
// member_cargos_all. As checagens de chave estrangeira rodam como dono da tabela
// e NAO passam por RLS, entao sem o EXISTS sobre `cargos` na politica seria
// possivel vincular um membro de um tenant a um cargo de outro.
func TestRLS_MemberCargoCannotBindForeignCargo(t *testing.T) {
	// Filial A tentando vincular o proprio membro a um cargo do tenant Y:
	// o RLS esconde o cargo Y, entao o EXISTS da politica e falso.
	err := inBounds(t, bounds(fixTenantX, fixBranchA, "secretario"), func(tx pgx.Tx) error {
		_, err := tx.Exec(testCtx, `
			INSERT INTO member_cargos (member_id, cargo_id, status)
			VALUES ($1::uuid, $2::uuid, 'ativo')`, fixMemberA2, fixCargoY)
		if err == nil {
			t.Errorf("filial A conseguiu vincular um membro do tenant X a um cargo do tenant Y")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("WithTenant(filial A): %v", err)
	}

	// Filial A tentando vincular um membro da filial B (mesmo tenant): escrever
	// fora da propria filial tambem e bloqueado.
	err = inBounds(t, bounds(fixTenantX, fixBranchA, "secretario"), func(tx pgx.Tx) error {
		_, err := tx.Exec(testCtx, `
			INSERT INTO member_cargos (member_id, cargo_id, status)
			VALUES ($1::uuid, $2::uuid, 'ativo')`, fixMemberB2, fixCargoA)
		if err == nil {
			t.Errorf("filial A conseguiu vincular um membro da filial B")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("WithTenant(filial A): %v", err)
	}

	// Contraprova: o vinculo legitimo (membro e cargo da propria filial) passa.
	err = inBounds(t, bounds(fixTenantX, fixBranchA, "secretario"), func(tx pgx.Tx) error {
		_, err := tx.Exec(testCtx, `
			INSERT INTO member_cargos (member_id, cargo_id, status)
			VALUES ($1::uuid, $2::uuid, 'ativo')`, fixMemberA2, fixCargoA)
		if err != nil {
			t.Errorf("filial A nao conseguiu vincular cargo da propria filial: %v", err)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("WithTenant(filial A): %v", err)
	}
}

// ---------------------------------------------------------------------------
// Historico eclesiastico do membro (migracao 000022)
// ---------------------------------------------------------------------------

// runInTx executa fn numa transacao propria (sempre rollback), para que um erro
// esperado nao aborte os cenarios seguintes do mesmo teste.
func runInTx(conn *pgx.Conn, fn func(tx pgx.Tx) error) error {
	tx, err := conn.Begin(testCtx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(testCtx) }()
	return fn(tx)
}

// seedHistoryMember cria um membro descartavel com uma entrada de historico e
// devolve o id. Roda transacionalmente e e sempre revertido.
func seedHistoryMember(t *testing.T, tx pgx.Tx) string {
	t.Helper()
	var id string
	if err := tx.QueryRow(testCtx, `
		INSERT INTO members (tenant_id, branch_id, first_name, last_name, full_name)
		VALUES ($1, $2, 'Hist', 'Teste', 'Hist Teste') RETURNING id::text`,
		fixTenantX, fixBranchA).Scan(&id); err != nil {
		t.Fatalf("inserir membro de teste: %v", err)
	}
	if _, err := tx.Exec(testCtx, `
		INSERT INTO member_history (tenant_id, branch_id, member_id, kind)
		VALUES ($1, $2, $3, 'cadastro')`, fixTenantX, fixBranchA, id); err != nil {
		t.Fatalf("inserir historico de teste: %v", err)
	}
	return id
}

func TestMemberHistory_AppendOnly(t *testing.T) {
	withSuperuser(t, func(conn *pgx.Conn) error {
		// UPDATE direto e barrado pelo trigger.
		if err := runInTx(conn, func(tx pgx.Tx) error {
			id := seedHistoryMember(t, tx)
			if _, err := tx.Exec(testCtx,
				`UPDATE member_history SET kind = 'hack' WHERE member_id = $1`, id); err == nil {
				t.Error("UPDATE em member_history foi aceito; deveria ser append-only")
			}
			return nil
		}); err != nil {
			t.Fatalf("cenario UPDATE: %v", err)
		}

		// DELETE direto tambem.
		if err := runInTx(conn, func(tx pgx.Tx) error {
			id := seedHistoryMember(t, tx)
			if _, err := tx.Exec(testCtx,
				`DELETE FROM member_history WHERE member_id = $1`, id); err == nil {
				t.Error("DELETE em member_history foi aceito; deveria ser append-only")
			}
			return nil
		}); err != nil {
			t.Fatalf("cenario DELETE: %v", err)
		}
		return nil
	})
}

// TestMemberHistory_CascadeDeleteWhenMemberRemoved garante que a imutabilidade
// NAO impede apagar um membro: o historico some junto (FK ON DELETE CASCADE) e
// o trigger libera a exclusao em cascata (pg_trigger_depth() > 1). Sem isso,
// remover um membro - ou reverter a conversao de um visitante - quebraria.
func TestMemberHistory_CascadeDeleteWhenMemberRemoved(t *testing.T) {
	withSuperuser(t, func(conn *pgx.Conn) error {
		return runInTx(conn, func(tx pgx.Tx) error {
			id := seedHistoryMember(t, tx)
			if _, err := tx.Exec(testCtx, `DELETE FROM members WHERE id = $1::uuid`, id); err != nil {
				t.Fatalf("excluir membro com historico deveria funcionar: %v", err)
			}
			var n int
			if err := tx.QueryRow(testCtx,
				`SELECT count(*) FROM member_history WHERE member_id = $1::uuid`, id).Scan(&n); err != nil {
				return err
			}
			if n != 0 {
				t.Errorf("historico do membro excluido continuou: %d linha(s)", n)
			}
			return nil
		})
	})
}

func TestRLS_MemberHistory_InsertCrossBranchBlocked(t *testing.T) {
	expectRLSDenied(t,
		bounds(fixTenantX, fixBranchA, "secretario"),
		`INSERT INTO member_history (tenant_id, branch_id, member_id, kind)
		 VALUES ($1, $2, $3, 'cadastro')`,
		fixTenantX, fixBranchB, fixMemberA1)
}

// ---------------------------------------------------------------------------
// Governanca (migracao 000033)
// ---------------------------------------------------------------------------

// TestRLS_Governance_ScopedToBranch confirma que atas, votacoes, opcoes, votos,
// documentos legais e assinaturas respeitam o escopo da filial.
func TestRLS_Governance_ScopedToBranch(t *testing.T) {
	cases := []struct {
		table string
		idA   string
	}{
		{"minutes", fixMinuteA},
		{"minute_signatures", fixSignatureA},
		{"votes", fixVoteA},
		{"vote_options", fixVoteOptA1},
		{"vote_ballots", fixBallotA},
		{"legal_documents", fixLegalA},
	}

	for _, tc := range cases {
		t.Run(tc.table, func(t *testing.T) {
			// A filial A enxerga o proprio registro.
			if err := inBounds(t, bounds(fixTenantX, fixBranchA, "secretario"), func(tx pgx.Tx) error {
				var n int
				if err := tx.QueryRow(testCtx, `SELECT count(*) FROM `+tc.table+` WHERE id = $1::uuid`, tc.idA).Scan(&n); err != nil {
					return err
				}
				if n != 1 {
					t.Errorf("filial A deveria ver 1 linha de %s, viu %d", tc.table, n)
				}
				return nil
			}); err != nil {
				t.Fatalf("WithTenant: %v", err)
			}

			// A filial B nao enxerga o registro da filial A.
			if err := inBounds(t, bounds(fixTenantX, fixBranchB, "secretario"), func(tx pgx.Tx) error {
				var n int
				if err := tx.QueryRow(testCtx, `SELECT count(*) FROM `+tc.table+` WHERE id = $1::uuid`, tc.idA).Scan(&n); err != nil {
					return err
				}
				if n != 0 {
					t.Errorf("vazamento entre filiais: filial B viu %d linha(s) de %s da filial A", n, tc.table)
				}
				return nil
			}); err != nil {
				t.Fatalf("WithTenant: %v", err)
			}
		})
	}
}

// TestRLS_Governance_InsertCrossBranchBlocked confirma o WITH CHECK da ata.
func TestRLS_Governance_InsertCrossBranchBlocked(t *testing.T) {
	expectRLSDenied(t,
		bounds(fixTenantX, fixBranchA, "secretario"),
		`INSERT INTO minutes (tenant_id, branch_id, title, meeting_at)
		 VALUES ($1, $2, 'Ata Vazada', now())`,
		fixTenantX, fixBranchB)
}

// TestGovernance_AppendOnly garante a imutabilidade das assinaturas de ata e
// dos votos (hash-chain): nem UPDATE nem DELETE passam pelo trigger.
func TestGovernance_AppendOnly(t *testing.T) {
	withSuperuser(t, func(conn *pgx.Conn) error {
		for _, q := range []string{
			`UPDATE minute_signatures SET signer_name = 'hack' WHERE id = '` + fixSignatureA + `'`,
			`DELETE FROM minute_signatures WHERE id = '` + fixSignatureA + `'`,
			`UPDATE vote_ballots SET option_id = option_id WHERE id = '` + fixBallotA + `'`,
			`DELETE FROM vote_ballots WHERE id = '` + fixBallotA + `'`,
		} {
			if _, err := conn.Exec(testCtx, q); err == nil {
				t.Errorf("operacao aceita em tabela append-only: %s", q)
			}
		}
		return nil
	})
}

// TestGovernance_CascadeDeleteWhenMinuteRemoved garante que a imutabilidade nao
// impede apagar a ata: as assinaturas somem por cascade.
func TestGovernance_CascadeDeleteWhenMinuteRemoved(t *testing.T) {
	withSuperuser(t, func(conn *pgx.Conn) error {
		return runInTx(conn, func(tx pgx.Tx) error {
			if _, err := tx.Exec(testCtx, `DELETE FROM minutes WHERE id = $1::uuid`, fixMinuteA); err != nil {
				t.Fatalf("excluir ata assinada deveria funcionar: %v", err)
			}
			var n int
			if err := tx.QueryRow(testCtx,
				`SELECT count(*) FROM minute_signatures WHERE minute_id = $1::uuid`, fixMinuteA).Scan(&n); err != nil {
				return err
			}
			if n != 0 {
				t.Errorf("assinaturas da ata excluida continuaram: %d", n)
			}
			return nil
		})
	})
}

// ---------------------------------------------------------------------------
// Hierarquia de filiais / sub-congregacoes (migracao 000036)
// ---------------------------------------------------------------------------

// TestRLS_BranchSeesSubCongregation prova a leitura hierarquica: a filial pai
// enxerga os dados da sub-congregacao, a filial irma nao, a Sede sim.
func TestRLS_BranchSeesSubCongregation(t *testing.T) {
	cases := []struct {
		name string
		b    Bounds
		want int
	}{
		{"filial A (pai) ve a sub-congregacao", bounds(fixTenantX, fixBranchA, "secretario"), 1},
		{"filial B (irma) nao ve a sub-congregacao", bounds(fixTenantX, fixBranchB, "secretario"), 0},
		{"sede ve a sub-congregacao", bounds(fixTenantX, "", "super_admin"), 1},
		{"outro tenant nao ve a sub-congregacao", bounds(fixTenantY, "", "super_admin"), 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := inBounds(t, tc.b, func(tx pgx.Tx) error {
				if got := count(t, tx, `SELECT count(*) FROM visitors WHERE id = $1`, fixVisitorSub); got != tc.want {
					t.Errorf("viu %d linha(s), queria %d", got, tc.want)
				}
				return nil
			}); err != nil {
				t.Fatalf("WithTenant: %v", err)
			}
		})
	}
}

// TestRLS_BranchWriteStaysExact fixa a decisao de arquitetura: a LEITURA e
// hierarquica, mas a GRAVACAO continua no branch exato do contexto. Assim uma
// congregacao nao altera o cadastro da sub-congregacao por engano.
func TestRLS_BranchWriteStaysExact(t *testing.T) {
	expectRLSDenied(t,
		bounds(fixTenantX, fixBranchA, "secretario"),
		`INSERT INTO visitors (tenant_id, branch_id, first_name, last_name)
		 VALUES ($1, $2, 'Invasor', 'Sub')`,
		fixTenantX, fixSubBranchA)
}

// TestRLS_HeadquartersWriteAllowed fixa a correcao da migracao 000045: a Sede
// do tenant (branch vazio + papel super_admin) mantem registros de qualquer
// filial do PROPRIO tenant - e o que destrava PATCH/DELETE/estorno que retornavam
// 404. O isolamento entre tenants continua valendo.
func TestRLS_HeadquartersWriteAllowed(t *testing.T) {
	// Sede do tenant X altera uma linha da filial B.
	err := inBounds(t, bounds(fixTenantX, "", "super_admin"), func(tx pgx.Tx) error {
		tag, err := tx.Exec(testCtx,
			`UPDATE members SET full_name = 'Sede Editou' WHERE id = $1`, fixMemberB1)
		if err != nil {
			t.Fatalf("UPDATE da Sede falhou: %v", err)
		}
		if tag.RowsAffected() != 1 {
			t.Errorf("UPDATE da Sede afetou %d linha(s), queria 1", tag.RowsAffected())
		}
		return nil
	})
	if err != nil {
		t.Fatalf("WithTenant: %v", err)
	}

	// A Sede de outro tenant continua sem gravar no tenant X.
	expectRLSDenied(t,
		bounds(fixTenantY, "", "super_admin"),
		`INSERT INTO visitors (tenant_id, branch_id, first_name, last_name)
		 VALUES ($1, $2, 'Intruso', 'Y')`,
		fixTenantX, fixBranchA)
}
