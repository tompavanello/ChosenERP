package store

import (
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"

	"chosenerp/internal/members"
)

// TestMemberEventKind_AppliesEffects cobre o nucleo da migracao 000065: lancar
// um evento da vida eclesiastica movimenta o membro de acordo com o que o tipo
// declarou (situacao, motivo/data de saida) e a reativacao limpa a saida.
func TestMemberEventKind_AppliesEffects(t *testing.T) {
	// Sede do tenant X (super_admin): pode gravar o catalogo global e o membro.
	err := inBounds(t, bounds(fixTenantX, "", "super_admin"), func(tx pgx.Tx) error {
		if _, err := tx.Exec(testCtx, `
			INSERT INTO member_event_kinds
				(tenant_id, name, slug, category, sets_status, sets_exit_reason)
			VALUES ($1, 'Teste Baixa', 'teste_baixa', 'saida', 'inactive', 'desligamento'),
			       ($1, 'Teste Reativa', 'teste_reativa', 'retorno', 'active', NULL)`,
			fixTenantX); err != nil {
			return err
		}
		// Reativacao limpa a saida (efeito clears_exit).
		if _, err := tx.Exec(testCtx,
			`UPDATE member_event_kinds SET clears_exit = true WHERE slug = 'teste_reativa'`); err != nil {
			return err
		}

		repo := &members.Repo{}

		// 1) Baixa: inativo + motivo + data de saida.
		entry, err := repo.AddHistory(testCtx, tx, fixMemberA1, "", "teste_baixa",
			"Baixa de teste", "", "", "")
		if err != nil {
			t.Fatalf("AddHistory(baixa): %v", err)
		}
		if entry.EventName == nil || *entry.EventName != "Teste Baixa" {
			t.Errorf("event_name nao resolvido: %+v", entry.EventName)
		}
		if entry.Notes == nil || !strings.Contains(*entry.Notes, "Efeito:") {
			t.Errorf("notas nao registram o efeito: %+v", entry.Notes)
		}

		var status string
		var reason, exited *string
		if err := tx.QueryRow(testCtx,
			`SELECT membership_status, exit_reason, exited_at::text FROM members WHERE id = $1::uuid`,
			fixMemberA1).Scan(&status, &reason, &exited); err != nil {
			return err
		}
		if status != "inactive" {
			t.Errorf("situacao = %q, queria inactive", status)
		}
		if reason == nil || *reason != "desligamento" {
			t.Errorf("motivo da baixa = %v, queria desligamento", reason)
		}
		if exited == nil {
			t.Errorf("exited_at nao foi gravado")
		}

		// 2) Reativacao: active e limpa motivo/data.
		if _, err := repo.AddHistory(testCtx, tx, fixMemberA1, "", "teste_reativa",
			"Voltou", "", "", ""); err != nil {
			t.Fatalf("AddHistory(reativacao): %v", err)
		}
		if err := tx.QueryRow(testCtx,
			`SELECT membership_status, exit_reason, exited_at::text FROM members WHERE id = $1::uuid`,
			fixMemberA1).Scan(&status, &reason, &exited); err != nil {
			return err
		}
		if status != "active" {
			t.Errorf("situacao apos reativacao = %q, queria active", status)
		}
		if reason != nil || exited != nil {
			t.Errorf("reativacao nao limpou a saida: reason=%v exited=%v", reason, exited)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("WithTenant: %v", err)
	}
}
