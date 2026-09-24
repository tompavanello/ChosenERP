package store

import (
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// ---------------------------------------------------------------------------
// Identidade global + memberships (migração 000053)
// ---------------------------------------------------------------------------

// A própria pessoa enxerga todos os seus vínculos, mesmo de tenants diferentes
// — é o que alimenta o seletor de igreja.
func TestRLS_MembershipsSelfAcrossTenants(t *testing.T) {
	err := inBounds(t, boundsUser(fixTenantX, "", "super_admin", fixUserBoth), func(tx pgx.Tx) error {
		got := count(t, tx, `SELECT count(*) FROM memberships WHERE user_id = $1::uuid`, fixUserBoth)
		if got != 2 {
			t.Errorf("self deveria ver 2 vínculos, viu %d", got)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("WithTenant: %v", err)
	}
}

// O vínculo de um tenant não vaza para outro.
func TestRLS_MembershipsIsolatedBetweenTenants(t *testing.T) {
	err := inBounds(t, boundsUser(fixTenantX, fixBranchA, "secretario", fixUserX), func(tx pgx.Tx) error {
		if got := count(t, tx, `SELECT count(*) FROM memberships WHERE tenant_id = $1::uuid`, fixTenantY); got != 0 {
			t.Errorf("vazamento cross-tenant: user X viu %d vínculo(s) do tenant Y", got)
		}
		if got := count(t, tx, `SELECT count(*) FROM memberships WHERE tenant_id = $1::uuid`, fixTenantX); got == 0 {
			t.Errorf("user X deveria ver o próprio vínculo no tenant X")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("WithTenant: %v", err)
	}
}

// users é global: self enxerga a si; quem está no tenant enxerga as identidades
// do próprio tenant; outro tenant não enxerga.
func TestRLS_UsersGlobalScope(t *testing.T) {
	// Self.
	if err := inBounds(t, boundsUser(fixTenantX, fixBranchA, "secretario", fixUserX), func(tx pgx.Tx) error {
		if got := count(t, tx, `SELECT count(*) FROM users WHERE id = $1::uuid`, fixUserX); got != 1 {
			t.Errorf("self não enxergou a própria identidade (%d)", got)
		}
		return nil
	}); err != nil {
		t.Fatalf("self: %v", err)
	}

	// Sede do tenant X enxerga identidades com vínculo no tenant, mas não do Y.
	if err := inBounds(t, boundsUser(fixTenantX, "", "super_admin", fixUserX), func(tx pgx.Tx) error {
		if got := count(t, tx, `SELECT count(*) FROM users WHERE id = $1::uuid`, fixUserBoth); got != 1 {
			t.Errorf("sede X deveria enxergar a identidade com vínculo no tenant X")
		}
		if got := count(t, tx, `SELECT count(*) FROM users WHERE id = $1::uuid`, fixUserY); got != 0 {
			t.Errorf("vazamento: sede X enxergou identidade só do tenant Y")
		}
		return nil
	}); err != nil {
		t.Fatalf("sede X: %v", err)
	}

	// Sede do tenant Y não enxerga identidade do X.
	if err := inBounds(t, boundsUser(fixTenantY, "", "super_admin", fixUserY), func(tx pgx.Tx) error {
		if got := count(t, tx, `SELECT count(*) FROM users WHERE id = $1::uuid`, fixUserX); got != 0 {
			t.Errorf("vazamento: sede Y enxergou identidade do tenant X")
		}
		return nil
	}); err != nil {
		t.Fatalf("sede Y: %v", err)
	}
}

// user_attach_to_tenant cria identidade + vínculo e recusa vínculo duplicado.
func TestRLS_UserAttachToTenant(t *testing.T) {
	err := inBounds(t, boundsUser(fixTenantX, "", "super_admin", fixUserX), func(tx pgx.Tx) error {
		var newID string
		if err := tx.QueryRow(testCtx, `
			SELECT user_attach_to_tenant('novo.user@rls.local', 'hash', 'Novo User',
			                             $1::uuid, 'secretario', NULL, true)`, fixTenantX).Scan(&newID); err != nil {
			t.Fatalf("attach: %v", err)
		}
		if got := count(t, tx, `SELECT count(*) FROM users WHERE id = $1::uuid`, newID); got != 1 {
			t.Errorf("identidade criada não visível (count=%d)", got)
		}
		if got := count(t, tx, `SELECT count(*) FROM memberships WHERE user_id = $1::uuid AND tenant_id = $2::uuid`, newID, fixTenantX); got != 1 {
			t.Errorf("membership não criado (count=%d)", got)
		}

		// Segundo vínculo da mesma identidade na mesma igreja => 23505.
		_, err := tx.Exec(testCtx, `
			SELECT user_attach_to_tenant('novo.user@rls.local', 'hash', 'Novo User',
			                             $1::uuid, 'secretario', NULL, true)`, fixTenantX)
		var pgErr *pgconn.PgError
		if !errors.As(err, &pgErr) || pgErr.Code != "23505" {
			t.Errorf("esperava 23505 no vínculo duplicado, veio %v", err)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("WithTenant: %v", err)
	}
}

// public_tenant() (000054) devolve o branding pelo slug, fora do RLS.
func TestPublicTenantFunction(t *testing.T) {
	err := inBounds(t, bounds("", "", "system"), func(tx pgx.Tx) error {
		var name string
		if err := tx.QueryRow(testCtx,
			`SELECT name FROM public_tenant('rls-tenant-x')`).Scan(&name); err != nil {
			t.Fatalf("public_tenant: %v", err)
		}
		if name != "Tenant X" {
			t.Errorf("public_tenant devolveu %q, queria 'Tenant X'", name)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("WithTenant: %v", err)
	}
}
