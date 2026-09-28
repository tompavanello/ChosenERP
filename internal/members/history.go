package members

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"chosenerp/internal/memberevents"
)

// HistoryEntry e um evento da vida eclesiastica do membro (requisito 1.8).
type HistoryEntry struct {
	ID            string    `json:"id"`
	MemberID      string    `json:"member_id"`
	OccurredAt    time.Time `json:"occurred_at"`
	Kind          string    `json:"kind"`
	EventKindID   *string   `json:"event_kind_id,omitempty"`
	EventName     *string   `json:"event_name,omitempty"`
	EventCategory *string   `json:"event_category,omitempty"`
	EventTone     *string   `json:"event_tone,omitempty"`
	Notes         *string   `json:"notes,omitempty"`
	CreatedBy     *string   `json:"created_by,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

// historyKindForStatus mapeia a nova situacao (e, quando inativo, o motivo) para
// o slug do evento do catalogo (migracao 000065). Os slugs abaixo sao os
// canonicos do seed.
func historyKindForStatus(status, exitReason string) string {
	switch status {
	case "active":
		return "reativacao"
	case "inactive":
		switch exitReason {
		case "transferencia":
			return "saida_transferencia"
		case "falecimento":
			return "saida_falecimento"
		case "ausencia", "abandono":
			return "abandono_atividades"
		default:
			return "baixa_rol"
		}
	default:
		return "status"
	}
}

// Rótulos legíveis usados no resumo do efeito gravado no historico.
var statusLabel = map[string]string{
	"active": "Ativo Professo", "member": "Ativo Nao Professo", "inactive": "Inativo",
}

var exitReasonLabel = map[string]string{
	"falecimento": "Falecimento", "desligamento": "Desligamento",
	"transferencia": "Transferencia", "abandono": "Abandono",
	"ausencia": "Ausencia", "outro": "Outro",
}

func nullStr(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// insertHistory grava um evento derivando tenant/branch do proprio membro (o
// INSERT..SELECT garante que o RLS WITH CHECK do historico passe e evita que o
// chamador precise carregar o contexto). Usado pelos fluxos automaticos
// (cadastro e mudanca de situacao), que NAO reaplicam efeitos - a coluna do
// membro ja foi alterada pelo fluxo. O event_kind_id e resolvido pelo slug.
func insertHistory(ctx context.Context, tx pgx.Tx, memberID, kind, notes, actorID string) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO member_history (tenant_id, branch_id, member_id, kind, event_kind_id, notes, created_by)
		SELECT m.tenant_id, m.branch_id, m.id, $2,
		       (SELECT id FROM member_event_kinds WHERE tenant_id = m.tenant_id AND slug = $2),
		       $3, $4::uuid
		FROM members m WHERE m.id = $1::uuid`,
		memberID, kind, nullStr(notes), nullStr(actorID))
	return err
}

// resolveEventKind localiza o tipo pelo id (preferencial) ou pelo slug.
func resolveEventKind(ctx context.Context, tx pgx.Tx, id, slug string) (*memberevents.EventKind, error) {
	repo := &memberevents.Repo{}
	if id != "" {
		return repo.GetByID(ctx, tx, id)
	}
	if slug != "" {
		return repo.GetBySlug(ctx, tx, slug)
	}
	return nil, errors.New("event_kind_id ou kind obrigatorio")
}

// applyEventEffects aplica no membro o que o evento declarou movimentar e
// devolve o resumo legivel do que mudou (para gravar no historico). Roda na
// mesma transacao do lancamento do historico.
func applyEventEffects(ctx context.Context, tx pgx.Tx, memberID string, k *memberevents.EventKind, occurredAt, baptismLocation string) ([]string, error) {
	var effects []string
	if k.ClearsExit {
		effects = append(effects, "saida reaberta (motivo/data limpos)")
	}
	if k.SetsStatus != nil && *k.SetsStatus != "" {
		if label, ok := statusLabel[*k.SetsStatus]; ok {
			effects = append(effects, "situacao -> "+label)
		} else {
			effects = append(effects, "situacao -> "+*k.SetsStatus)
		}
	}
	if k.SetsExitReason != nil && *k.SetsExitReason != "" {
		if label, ok := exitReasonLabel[*k.SetsExitReason]; ok {
			effects = append(effects, "motivo da inatividade -> "+label)
		} else {
			effects = append(effects, "motivo da inatividade -> "+*k.SetsExitReason)
		}
	}
	if k.SetsBaptism {
		effects = append(effects, "batismo registrado")
	}
	if k.SetsDateField == "joined_at" {
		effects = append(effects, "membro desde atualizado")
	}
	if k.SetsDateField == "marriage_date" {
		effects = append(effects, "data de casamento atualizada")
	}

	// Sem nenhum efeito relevante, evita um UPDATE inutil.
	if len(effects) == 0 && !k.SetsBaptism {
		return effects, nil
	}

	_, err := tx.Exec(ctx, `
		UPDATE members SET
			membership_status = COALESCE($2::text, membership_status),
			exit_reason = CASE
				WHEN btrim(COALESCE($4::text,'')) <> '' THEN btrim($4::text)
				WHEN $2::text = 'inactive' THEN COALESCE(exit_reason, 'outro')
				WHEN $2::text IN ('active','member') THEN NULL
				WHEN $3::boolean THEN NULL
				ELSE exit_reason END,
			exited_at = CASE
				WHEN $2::text = 'inactive' THEN COALESCE(exited_at, $5::timestamptz, now())::date
				WHEN $2::text IN ('active','member') THEN NULL
				WHEN $3::boolean THEN NULL
				WHEN btrim(COALESCE($4::text,'')) <> '' THEN COALESCE($5::timestamptz, now())::date
				ELSE exited_at END,
			baptism_date = CASE WHEN $6::boolean THEN COALESCE($5::timestamptz, now())::date ELSE baptism_date END,
			baptism_location = CASE WHEN $6::boolean AND btrim(COALESCE($7,'')) <> '' THEN btrim($7) ELSE baptism_location END,
			joined_at = CASE WHEN $8::text = 'joined_at' THEN COALESCE($5::timestamptz, now())::date ELSE joined_at END,
			marriage_date = CASE WHEN $8::text = 'marriage_date' THEN COALESCE($5::timestamptz, now())::date ELSE marriage_date END,
			updated_at = now()
		WHERE id = $1::uuid`,
		memberID, k.SetsStatus, k.ClearsExit, k.SetsExitReason,
		nullStr(occurredAt), k.SetsBaptism, baptismLocation, k.SetsDateField)
	if err != nil {
		return nil, err
	}
	return effects, nil
}

// AddHistory insere um evento manual (batismo, recepcao, ordenacao, baixa...)
// e aplica os efeitos declarados pelo tipo. `occurredAt` vazio usa agora.
func (r *Repo) AddHistory(ctx context.Context, tx pgx.Tx, memberID, eventKindID, kind, notes, occurredAt, baptismLocation, actorID string) (*HistoryEntry, error) {
	ev, err := resolveEventKind(ctx, tx, eventKindID, kind)
	if err != nil {
		return nil, err
	}
	effects, err := applyEventEffects(ctx, tx, memberID, ev, occurredAt, baptismLocation)
	if err != nil {
		return nil, err
	}
	if len(effects) > 0 {
		prefix := "Efeito: " + strings.Join(effects, "; ")
		if strings.TrimSpace(notes) == "" {
			notes = prefix
		} else {
			notes = notes + " (" + prefix + ")"
		}
	}

	var id string
	err = tx.QueryRow(ctx, `
		INSERT INTO member_history (tenant_id, branch_id, member_id, kind, event_kind_id, notes, occurred_at, created_by)
		SELECT m.tenant_id, m.branch_id, m.id, $2, $3::uuid, $4, COALESCE($5::timestamptz, now()), $6::uuid
		FROM members m WHERE m.id = $1::uuid
		RETURNING id::text`,
		memberID, ev.Slug, ev.ID, nullStr(notes), nullStr(occurredAt), nullStr(actorID)).Scan(&id)
	if err != nil {
		return nil, err
	}
	return r.getHistory(ctx, tx, id)
}

const historyCols = `h.id::text, h.member_id::text, h.occurred_at, h.kind, h.event_kind_id::text,
	mk.name, mk.category, mk.tone, h.notes, h.created_by::text, h.created_at`

func scanHistory(row pgx.Row) (*HistoryEntry, error) {
	var h HistoryEntry
	err := row.Scan(&h.ID, &h.MemberID, &h.OccurredAt, &h.Kind, &h.EventKindID,
		&h.EventName, &h.EventCategory, &h.EventTone, &h.Notes, &h.CreatedBy, &h.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &h, nil
}

func (r *Repo) getHistory(ctx context.Context, tx pgx.Tx, id string) (*HistoryEntry, error) {
	return scanHistory(tx.QueryRow(ctx, `
		SELECT `+historyCols+`
		FROM member_history h
		LEFT JOIN member_event_kinds mk ON mk.id = h.event_kind_id
		WHERE h.id = $1::uuid`, id))
}

// ListHistory devolve a linha do tempo do membro, do mais recente para o mais
// antigo.
func (r *Repo) ListHistory(ctx context.Context, tx pgx.Tx, memberID string) ([]HistoryEntry, error) {
	rows, err := tx.Query(ctx, `
		SELECT `+historyCols+`
		FROM member_history h
		LEFT JOIN member_event_kinds mk ON mk.id = h.event_kind_id
		WHERE h.member_id = $1::uuid
		ORDER BY h.occurred_at DESC, h.created_at DESC`, memberID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []HistoryEntry{}
	for rows.Next() {
		h, err := scanHistory(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *h)
	}
	return out, rows.Err()
}
