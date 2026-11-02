package events

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
)

// ListAgenda lista as ocorrencias do calendario com os filtros do scheduler:
// periodo, tipo, ministerio dono, origem e situacao.
func (r *Repo) ListAgenda(ctx context.Context, tx pgx.Tx, from, to, kindID, ministryID, origin, status string) ([]Event, error) {
	rows, err := tx.Query(ctx, `
		SELECT `+eventCols+`
		FROM church_events e
		LEFT JOIN event_kinds k ON k.id = e.kind_id
		LEFT JOIN ministries m ON m.id = e.owner_ministry_id
		WHERE ($1 = '' OR e.starts_at::date >= $1::date)
		  AND ($2 = '' OR e.starts_at::date <= $2::date)
		  AND ($3 = '' OR e.kind_id = $3::uuid)
		  AND ($4 = '' OR e.owner_ministry_id = NULLIF($4,'')::uuid)
		  AND ($5 = '' OR e.origin = $5)
		  AND ($6 = '' OR e.status = $6)
		ORDER BY e.starts_at`, from, to, kindID, ministryID, origin, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Event{}
	for rows.Next() {
		e, err := scanEvent(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *e)
	}
	return out, rows.Err()
}

// MoveOccurrence remarca uma ocorrencia preservando o calendario. Quando ela
// vem de uma serie (programacao), grava uma excecao 'move' para que publicar a
// grade de novo nao recrie a ocorrencia no horario antigo.
func (r *Repo) MoveOccurrence(ctx context.Context, tx pgx.Tx, id, startsAt string, endsAt *string, scope, actorID string) (*Event, error) {
	ev, err := r.GetEvent(ctx, tx, id)
	if err != nil {
		return nil, err
	}
	starts, err := parseTime(startsAt)
	if err != nil {
		return nil, err
	}
	var ends *time.Time
	if endsAt != nil && *endsAt != "" {
		t, err := parseTime(*endsAt)
		if err != nil {
			return nil, err
		}
		ends = &t
	}

	if ev.Origin == "programacao" && ev.OriginID != nil && scope != "all" {
		_, err = tx.Exec(ctx, `
			INSERT INTO programacao_exceptions
				(tenant_id, branch_id, programacao_id, occurrence_date, action,
				 new_starts_at, new_ends_at, created_by)
			SELECT e.tenant_id, e.branch_id, e.origin_id,
			       COALESCE(e.occurrence_date,
			                (e.starts_at AT TIME ZONE COALESCE(NULLIF(t.timezone,''),'America/Sao_Paulo'))::date),
			       'move', $2, $3, NULLIF($4,'')::uuid
			FROM church_events e
			JOIN tenants t ON t.id = e.tenant_id
			WHERE e.id = $1::uuid
			ON CONFLICT (programacao_id, occurrence_date) DO UPDATE SET
				action = 'move',
				new_starts_at = EXCLUDED.new_starts_at,
				new_ends_at = EXCLUDED.new_ends_at,
				created_by = EXCLUDED.created_by`, id, starts, ends, actorID)
		if err != nil {
			return nil, err
		}
	}

	if _, err := tx.Exec(ctx, `
		UPDATE church_events SET starts_at = $2, ends_at = $3, updated_at = now()
		WHERE id = $1::uuid`, id, starts, ends); err != nil {
		return nil, err
	}
	return r.GetEvent(ctx, tx, id)
}

// DeleteOccurrence exclui uma ocorrencia. Numa serie, scope='this' grava uma
// excecao 'cancel' (a grade nao a recria) e scope='all' desativa a serie e
// remove as ocorrencias futuras.
func (r *Repo) DeleteOccurrence(ctx context.Context, tx pgx.Tx, id, scope string) error {
	ev, err := r.GetEvent(ctx, tx, id)
	if err != nil {
		return err
	}
	if ev.Origin == "programacao" && ev.OriginID != nil {
		if scope == "all" {
			if _, err := tx.Exec(ctx, `
				UPDATE programacoes SET is_active = false, updated_at = now()
				WHERE id = $1::uuid`, *ev.OriginID); err != nil {
				return err
			}
			_, err := tx.Exec(ctx, `
				DELETE FROM church_events
				WHERE origin = 'programacao' AND origin_id = $1::uuid AND starts_at >= now()`,
				*ev.OriginID)
			return err
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO programacao_exceptions
				(tenant_id, branch_id, programacao_id, occurrence_date, action, created_by)
			SELECT e.tenant_id, e.branch_id, e.origin_id,
			       COALESCE(e.occurrence_date,
			                (e.starts_at AT TIME ZONE COALESCE(NULLIF(t.timezone,''),'America/Sao_Paulo'))::date),
			       'cancel', NULL
			FROM church_events e
			JOIN tenants t ON t.id = e.tenant_id
			WHERE e.id = $1::uuid
			ON CONFLICT (programacao_id, occurrence_date) DO UPDATE SET
				action = 'cancel', new_starts_at = NULL, new_ends_at = NULL`, id); err != nil {
			return err
		}
	}
	return r.DeleteEvent(ctx, tx, id)
}

// SetOccurrenceStatus atualiza a situacao da ocorrencia (planejado/confirmado/
// cancelado/concluido) sem mexer no restante da serie.
func (r *Repo) SetOccurrenceStatus(ctx context.Context, tx pgx.Tx, id, status string) (*Event, error) {
	if status != "planned" && status != "confirmed" && status != "cancelled" && status != "done" {
		status = "planned"
	}
	if _, err := tx.Exec(ctx, `
		UPDATE church_events SET status = $2, updated_at = now()
		WHERE id = $1::uuid`, id, status); err != nil {
		return nil, err
	}
	return r.GetEvent(ctx, tx, id)
}
