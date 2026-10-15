package store

import (
	"context"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
)

// TestBackupRestoreRoundTrip valida as funcoes export_tenant/restore_tenant:
// o export captura o tenant inteiro; apos alterar o estado, o restore substitui
// o conteudo daquele tenant e NAO toca nas outras igrejas.
func TestBackupRestoreRoundTrip(t *testing.T) {
	if testStore == nil {
		t.Skip("sem banco de teste")
	}
	ctx := context.Background()

	// 1) Exporta o tenant X.
	var payload string
	if err := testStore.WithSystem(ctx, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `SELECT export_tenant($1::uuid)::text`, fixTenantX).Scan(&payload)
	}); err != nil {
		t.Fatalf("export_tenant: %v", err)
	}
	if !strings.Contains(payload, fixMemberA1) {
		t.Fatalf("payload nao contem os membros do tenant X")
	}

	// 2) Altera o estado: um membro a mais no tenant X (deve sumir no restore)
	//    e um membro intacto no tenant Y (deve permanecer). Remove tambem uma
	//    identidade global (fixUserX) para exercitar a reinsercao de users.
	withSuperuser(t, func(conn *pgx.Conn) error {
		if _, err := conn.Exec(ctx, `
			INSERT INTO members (id, tenant_id, branch_id, first_name, last_name, full_name, membership_status)
			VALUES (gen_random_uuid(), $1::uuid, $2::uuid, 'Extra', 'X', 'Extra X', 'active')`,
			fixTenantX, fixBranchA); err != nil {
			return err
		}
		if _, err := conn.Exec(ctx, `DELETE FROM users WHERE id = $1::uuid`, fixUserX); err != nil {
			return err
		}
		return nil
	})

	// 3) Restaura o tenant X a partir do payload.
	var result string
	if err := testStore.WithSystem(ctx, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `SELECT restore_tenant($1::uuid, $2::jsonb)::text`, fixTenantX, payload).Scan(&result)
	}); err != nil {
		t.Fatalf("restore_tenant: %v", err)
	}

	// 4) Confere: membro extra removido; fixtures do X presentes; Y intacto.
	withSuperuser(t, func(conn *pgx.Conn) error {
		var xCount, extraCount, yCount int
		if err := conn.QueryRow(ctx, `SELECT count(*) FROM members WHERE tenant_id = $1::uuid`, fixTenantX).Scan(&xCount); err != nil {
			return err
		}
		if err := conn.QueryRow(ctx, `SELECT count(*) FROM members WHERE tenant_id = $1::uuid AND first_name = 'Extra'`, fixTenantX).Scan(&extraCount); err != nil {
			return err
		}
		if err := conn.QueryRow(ctx, `SELECT count(*) FROM members WHERE tenant_id = $1::uuid`, fixTenantY).Scan(&yCount); err != nil {
			return err
		}
		if xCount != 4 {
			t.Errorf("tenant X deveria ter 4 membros apos restore, tem %d", xCount)
		}
		if extraCount != 0 {
			t.Errorf("o membro extra deveria ter sido removido pelo restore")
		}
		if yCount == 0 {
			t.Errorf("o tenant Y foi afetado pelo restore do tenant X")
		}
		// Identidade global removida deve ter sido reinserida.
		var userCount int
		if err := conn.QueryRow(ctx, `SELECT count(*) FROM users WHERE id = $1::uuid`, fixUserX).Scan(&userCount); err != nil {
			return err
		}
		if userCount != 1 {
			t.Errorf("a identidade fixUserX deveria ter sido restaurada")
		}
		var memCount int
		if err := conn.QueryRow(ctx, `SELECT count(*) FROM memberships WHERE id = $1::uuid`, fixMemX).Scan(&memCount); err != nil {
			return err
		}
		if memCount != 1 {
			t.Errorf("o membership fixMemX deveria ter sido restaurado")
		}
		// Append-only: os lancamentos do X voltaram.
		var txCount int
		if err := conn.QueryRow(ctx, `SELECT count(*) FROM financial_transactions WHERE tenant_id = $1::uuid`, fixTenantX).Scan(&txCount); err != nil {
			return err
		}
		if txCount == 0 {
			t.Errorf("os lancamentos financeiros do tenant X nao voltaram")
		}
		return nil
	})
}
