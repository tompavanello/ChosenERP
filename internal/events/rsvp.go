package events

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

// ErrInvalidRSVP sinaliza status de confirmacao de presenca invalido.
var ErrInvalidRSVP = errors.New("status de presenca invalido (use going, maybe ou declined)")

// RSVP e a confirmacao de presenca do membro a um evento.
type RSVP struct {
	EventID    string `json:"event_id"`
	MemberID   string `json:"member_id"`
	MemberName string `json:"member_name,omitempty"`
	Status     string `json:"status"`
	UpdatedAt  string `json:"updated_at,omitempty"`
}

// RSVPResult resume as respostas de um evento (contagens + lista).
type RSVPResult struct {
	Going    int    `json:"going"`
	Maybe    int    `json:"maybe"`
	Declined int    `json:"declined"`
	RSVPs    []RSVP `json:"rsvps"`
}

// UpsertRSVP grava/atualiza a resposta do membro ao evento.
func (r *Repo) UpsertRSVP(ctx context.Context, tx pgx.Tx, tenantID, branchID, eventID, memberID, status string) error {
	if status != "going" && status != "maybe" && status != "declined" {
		return ErrInvalidRSVP
	}
	_, err := tx.Exec(ctx, `
		INSERT INTO event_rsvps (tenant_id, branch_id, event_id, member_id, status)
		VALUES ($1, $2::uuid, $3::uuid, $4::uuid, $5)
		ON CONFLICT (event_id, member_id) DO UPDATE SET status = EXCLUDED.status, updated_at = now()`,
		tenantID, branchID, eventID, memberID, status)
	return err
}

// DeleteRSVP remove a resposta do membro (cancelar o "eu vou").
func (r *Repo) DeleteRSVP(ctx context.Context, tx pgx.Tx, eventID, memberID string) error {
	_, err := tx.Exec(ctx, `DELETE FROM event_rsvps WHERE event_id = $1::uuid AND member_id = $2::uuid`, eventID, memberID)
	return err
}

// MemberRSVPs devolve o status do membro por evento (para enriquecer /me/events).
func (r *Repo) MemberRSVPs(ctx context.Context, tx pgx.Tx, memberID string) (map[string]string, error) {
	rows, err := tx.Query(ctx, `SELECT event_id::text, status FROM event_rsvps WHERE member_id = $1::uuid`, memberID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]string{}
	for rows.Next() {
		var id, st string
		if err := rows.Scan(&id, &st); err != nil {
			return nil, err
		}
		out[id] = st
	}
	return out, rows.Err()
}

// RSVPsOfEvent devolve as respostas de um evento com as contagens por status.
func (r *Repo) RSVPsOfEvent(ctx context.Context, tx pgx.Tx, eventID string) (*RSVPResult, error) {
	res := &RSVPResult{RSVPs: []RSVP{}}
	rows, err := tx.Query(ctx, `
		SELECT rv.event_id::text, rv.member_id::text, mb.full_name, rv.status, rv.updated_at::text
		FROM event_rsvps rv
		JOIN members mb ON mb.id = rv.member_id
		WHERE rv.event_id = $1::uuid
		ORDER BY rv.status, mb.full_name`, eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var x RSVP
		if err := rows.Scan(&x.EventID, &x.MemberID, &x.MemberName, &x.Status, &x.UpdatedAt); err != nil {
			return nil, err
		}
		switch x.Status {
		case "going":
			res.Going++
		case "maybe":
			res.Maybe++
		case "declined":
			res.Declined++
		}
		res.RSVPs = append(res.RSVPs, x)
	}
	return res, rows.Err()
}

// SelfCheckIn marca a presenca do membro no evento (chamada nominal).
func (r *Repo) SelfCheckIn(ctx context.Context, tx pgx.Tx, tenantID, branchID, eventID, memberID string) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO event_attendance (tenant_id, branch_id, event_id, member_id, present)
		VALUES ($1, $2::uuid, $3::uuid, $4::uuid, true)
		ON CONFLICT (event_id, member_id) DO UPDATE SET present = true`,
		tenantID, branchID, eventID, memberID)
	return err
}
