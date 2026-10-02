package store

import (
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"chosenerp/internal/org"
)

// ---------------------------------------------------------------------------
// Platform admin + planos (migracao 000075)
// ---------------------------------------------------------------------------

// is_platform_admin() le a flag da propria identidade.
func TestIsPlatformAdmin(t *testing.T) {
	err := inBounds(t, boundsUser(fixTenantX, "", "super_admin", fixUserX), func(tx pgx.Tx) error {
		var before bool
		if err := tx.QueryRow(testCtx, `SELECT is_platform_admin()`).Scan(&before); err != nil {
			t.Fatalf("is_platform_admin: %v", err)
		}
		if before {
			t.Errorf("comecou true, esperava false")
		}
		if _, err := tx.Exec(testCtx,
			`UPDATE users SET is_platform_admin = true WHERE id = current_user_id()`); err != nil {
			t.Fatalf("marcar platform admin: %v", err)
		}
		var after bool
		if err := tx.QueryRow(testCtx, `SELECT is_platform_admin()`).Scan(&after); err != nil {
			t.Fatalf("is_platform_admin (depois): %v", err)
		}
		if !after {
			t.Errorf("esperava true depois de marcar a flag")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("WithTenant: %v", err)
	}
}

// O catalogo de planos so e visivel ao sistema/platform admin.
func TestPlansVisibility(t *testing.T) {
	// Sistema enxerga o catalogo semeado pela migracao.
	if err := inSystem(t, func(tx pgx.Tx) error {
		if got := count(t, tx, `SELECT count(*) FROM plans`); got < 3 {
			t.Errorf("system deveria ver >= 3 planos, viu %d", got)
		}
		return nil
	}); err != nil {
		t.Fatalf("system: %v", err)
	}

	// Usuario comum le APENAS o proprio plano (policy 000079), nao o catalogo.
	if err := inBounds(t, boundsUser(fixTenantX, fixBranchA, "secretario", fixUserX), func(tx pgx.Tx) error {
		if got := count(t, tx, `SELECT count(*) FROM plans`); got != 1 {
			t.Errorf("usuario comum deveria ver so o proprio plano, viu %d", got)
		}
		if got := count(t, tx, `SELECT count(*) FROM plans WHERE key = 'enterprise'`); got != 0 {
			t.Errorf("usuario comum nao deveria enxergar planos de outras igrejas")
		}
		return nil
	}); err != nil {
		t.Fatalf("common: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Acesso do membro (migracao 000076)
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// Entitlements por plano (migracao 000078)
// ---------------------------------------------------------------------------

// O plano Starter nao inclui finance, mas inclui events; o override da igreja
// pode liberar finance.
func TestEntitlementsResolve(t *testing.T) {
	err := inSystem(t, func(tx pgx.Tx) error {
		ent, e := org.EffectiveForTenant(testCtx, tx, fixTenantX)
		if e != nil {
			t.Fatalf("entitlements: %v", e)
		}
		if ent.Has("finance") {
			t.Errorf("starter nao deveria incluir finance")
		}
		if !ent.Has("events") {
			t.Errorf("starter deveria incluir events")
		}

		if _, e := tx.Exec(testCtx,
			`UPDATE tenants SET features = '{"finance":true}'::jsonb WHERE id = $1::uuid`,
			fixTenantX); e != nil {
			t.Fatalf("override: %v", e)
		}
		ent2, e := org.EffectiveForTenant(testCtx, tx, fixTenantX)
		if e != nil {
			t.Fatalf("entitlements2: %v", e)
		}
		if !ent2.Has("finance") {
			t.Errorf("override por igreja deveria liberar finance")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("WithSystem: %v", err)
	}
}

// auth_lookup_identity casa por e-mail E por telefone.
func TestAuthLookupIdentityByEmailAndPhone(t *testing.T) {
	err := inSystem(t, func(tx pgx.Tx) error {
		if _, err := tx.Exec(testCtx,
			`UPDATE users SET phone = '5548999990000' WHERE id = $1::uuid`, fixUserX); err != nil {
			t.Fatalf("set phone: %v", err)
		}
		var userID string
		if err := tx.QueryRow(testCtx,
			`SELECT user_id FROM auth_lookup_identity('user.x@rls.local')`).Scan(&userID); err != nil {
			t.Fatalf("por e-mail: %v", err)
		}
		if userID != fixUserX {
			t.Errorf("e-mail: veio %s, queria %s", userID, fixUserX)
		}
		// Telefone com mascara tambem deve casar (normalizado).
		if err := tx.QueryRow(testCtx,
			`SELECT user_id FROM auth_lookup_identity('+55 (48) 99999-0000')`).Scan(&userID); err != nil {
			t.Fatalf("por telefone: %v", err)
		}
		if userID != fixUserX {
			t.Errorf("telefone: veio %s, queria %s", userID, fixUserX)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("WithSystem: %v", err)
	}
}

// user_attach_member cria a identidade, o membership (papel membro) com
// member_id e recusa vincular o mesmo membro a outro acesso.
func TestUserAttachMember(t *testing.T) {
	err := inSystem(t, func(tx pgx.Tx) error {
		var userID string
		if err := tx.QueryRow(testCtx, `
			SELECT user_attach_member('novo.membro@rls.local', '5548988887777', 'hash',
			                          'Novo Membro', $1::uuid, $2::uuid)`,
			fixTenantX, fixMemberA2).Scan(&userID); err != nil {
			t.Fatalf("attach member: %v", err)
		}
		if got := count(t, tx,
			`SELECT count(*) FROM memberships WHERE user_id = $1::uuid AND member_id = $2::uuid`,
			userID, fixMemberA2); got != 1 {
			t.Errorf("membership com member_id nao criado (count=%d)", got)
		}
		if got := count(t, tx,
			`SELECT count(*) FROM memberships m JOIN roles r ON r.id = m.role_id
			 WHERE m.user_id = $1::uuid AND r.key = 'membro'`, userID); got != 1 {
			t.Errorf("papel membro nao atribuido (count=%d)", got)
		}

		// Mesmo membro em outra identidade => 23505 (uq_memberships_tenant_member).
		_, err := tx.Exec(testCtx, `
			SELECT user_attach_member('outro.membro@rls.local', '', 'hash',
			                          'Outro', $1::uuid, $2::uuid)`,
			fixTenantX, fixMemberA2)
		var pgErr *pgconn.PgError
		if !errors.As(err, &pgErr) || pgErr.Code != "23505" {
			t.Errorf("esperava 23505 ao vincular membro ja usado, veio %v", err)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("WithSystem: %v", err)
	}
}
