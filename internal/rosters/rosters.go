// Package rosters gerencia escalas de voluntários: criação da escala, convocação
// de membros (com função), confirmação/recusa de presença, detecção de conflito
// de agenda e sugestão de voluntários. O isolamento é do RLS.
package rosters

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

// ErrInvalidStatus sinaliza status de resposta inválido.
var ErrInvalidStatus = errors.New("status inválido (use confirmado ou recusado)")

// Roster é uma escala.
type Roster struct {
	ID              string       `json:"id"`
	BranchID        string       `json:"branch_id"`
	MinistryID      *string      `json:"ministry_id,omitempty"`
	MinistryName    *string      `json:"ministry_name,omitempty"`
	EventID         *string      `json:"event_id,omitempty"`
	EventName       *string      `json:"event_name,omitempty"`
	EventKindID     *string      `json:"event_kind_id,omitempty"`
	EventKindName   *string      `json:"event_kind_name,omitempty"`
	GeneratedEvent  bool         `json:"generated_event"`
	Title           string       `json:"title"`
	StartsAt        time.Time    `json:"starts_at"`
	EndsAt          *time.Time   `json:"ends_at,omitempty"`
	Location        *string      `json:"location,omitempty"`
	Notes           *string      `json:"notes,omitempty"`
	Status          string       `json:"status"`
	AssignmentCount int          `json:"assignment_count"`
	ConfirmedCount  int          `json:"confirmed_count"`
	Assignments     []Assignment `json:"assignments,omitempty"`
	CreatedAt       time.Time    `json:"created_at"`
}

// Assignment é um voluntário escalado.
type Assignment struct {
	ID          string     `json:"id"`
	RosterID    string     `json:"roster_id"`
	MemberID    string     `json:"member_id"`
	MemberName  string     `json:"member_name"`
	Role        *string    `json:"role,omitempty"`
	Status      string     `json:"status"`
	RespondedAt *time.Time `json:"responded_at,omitempty"`
	Notes       *string    `json:"notes,omitempty"`
}

// CreateInput é o corpo de criação da escala.
type CreateInput struct {
	MinistryID  *string           `json:"ministry_id"`
	EventID     *string           `json:"event_id"`
	EventKindID *string           `json:"event_kind_id"`
	CreateEvent bool              `json:"create_event"`
	Title       string            `json:"title"`
	StartsAt    string            `json:"starts_at"`
	EndsAt      *string           `json:"ends_at"`
	Location    *string           `json:"location"`
	Notes       *string           `json:"notes"`
	Status      *string           `json:"status"`
	Assignments []AssignmentInput `json:"assignments"`
}

// UpdateInput é o corpo de edição da escala.
type UpdateInput struct {
	MinistryID  *string `json:"ministry_id"`
	EventID     *string `json:"event_id"`
	EventKindID *string `json:"event_kind_id"`
	Title       *string `json:"title"`
	StartsAt    *string `json:"starts_at"`
	EndsAt      *string `json:"ends_at"`
	Location    *string `json:"location"`
	Notes       *string `json:"notes"`
	Status      *string `json:"status"`
}

// AssignmentInput identifica um membro a escalar.
type AssignmentInput struct {
	MemberID string  `json:"member_id"`
	Role     *string `json:"role"`
}

// Conflict é um choque de agenda de um membro escalado.
type Conflict struct {
	MemberID    string    `json:"member_id"`
	MemberName  string    `json:"member_name"`
	OtherID     string    `json:"other_roster_id"`
	OtherTitle  string    `json:"other_roster_title"`
	OtherStarts time.Time `json:"other_starts_at"`
}

// Suggestion é um candidato a entrar na escala.
type Suggestion struct {
	MemberID   string  `json:"member_id"`
	MemberName string  `json:"member_name"`
	Frequency  *string `json:"frequency,omitempty"`
	Busy       bool    `json:"busy"`
}

type Repo struct{}

const rosterCols = `r.id::text, r.branch_id::text, r.ministry_id::text, mi.name, r.event_id::text, ek.name,
	r.event_kind_id::text, ek2.name, r.generated_event,
	r.title, r.starts_at, r.ends_at, r.location, r.notes, r.status,
	(SELECT count(*) FROM roster_assignments a WHERE a.roster_id = r.id)::int,
	(SELECT count(*) FROM roster_assignments a WHERE a.roster_id = r.id AND a.status = 'confirmado')::int,
	r.created_at`

func scanRoster(row pgx.Row) (*Roster, error) {
	var r Roster
	err := row.Scan(&r.ID, &r.BranchID, &r.MinistryID, &r.MinistryName, &r.EventID, &r.EventName,
		&r.EventKindID, &r.EventKindName, &r.GeneratedEvent,
		&r.Title, &r.StartsAt, &r.EndsAt, &r.Location, &r.Notes, &r.Status,
		&r.AssignmentCount, &r.ConfirmedCount, &r.CreatedAt)
	return &r, err
}

// List devolve as escalas do escopo, com filtro opcional de período/ministério.
func (r *Repo) List(ctx context.Context, tx pgx.Tx, from, to, ministryID string) ([]Roster, error) {
	rows, err := tx.Query(ctx, `
		SELECT `+rosterCols+`
		FROM rosters r
		LEFT JOIN ministries mi ON mi.id = r.ministry_id
		LEFT JOIN church_events ce ON ce.id = r.event_id
		LEFT JOIN event_kinds ek ON ek.id = ce.kind_id
		LEFT JOIN event_kinds ek2 ON ek2.id = r.event_kind_id
		WHERE ($1 = '' OR r.starts_at::date >= $1::date)
		  AND ($2 = '' OR r.starts_at::date <= $2::date)
		  AND ($3 = '' OR r.ministry_id = $3::uuid)
		ORDER BY r.starts_at DESC LIMIT 300`, from, to, ministryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Roster{}
	for rows.Next() {
		ro, err := scanRoster(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *ro)
	}
	return out, rows.Err()
}

// Get devolve a escala com os escalados.
func (r *Repo) Get(ctx context.Context, tx pgx.Tx, id string) (*Roster, error) {
	ro, err := scanRoster(tx.QueryRow(ctx, `
		SELECT `+rosterCols+`
		FROM rosters r
		LEFT JOIN ministries mi ON mi.id = r.ministry_id
		LEFT JOIN church_events ce ON ce.id = r.event_id
		LEFT JOIN event_kinds ek ON ek.id = ce.kind_id
		LEFT JOIN event_kinds ek2 ON ek2.id = r.event_kind_id
		WHERE r.id = $1::uuid`, id))
	if err != nil {
		return nil, err
	}
	if ro.Assignments, err = r.ListAssignments(ctx, tx, id); err != nil {
		return nil, err
	}
	return ro, nil
}

// ListAssignments devolve os voluntários de uma escala.
func (r *Repo) ListAssignments(ctx context.Context, tx pgx.Tx, rosterID string) ([]Assignment, error) {
	rows, err := tx.Query(ctx, `
		SELECT a.id::text, a.roster_id::text, a.member_id::text, mb.full_name, a.role, a.status, a.responded_at, a.notes
		FROM roster_assignments a
		JOIN members mb ON mb.id = a.member_id
		WHERE a.roster_id = $1::uuid
		ORDER BY mb.full_name`, rosterID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Assignment{}
	for rows.Next() {
		var a Assignment
		if err := rows.Scan(&a.ID, &a.RosterID, &a.MemberID, &a.MemberName, &a.Role, &a.Status, &a.RespondedAt, &a.Notes); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// Create insere a escala (e, opcionalmente, já os escalados).
func (r *Repo) Create(ctx context.Context, tx pgx.Tx, tenantID, branchID, actorID string, in CreateInput) (*Roster, error) {
	starts, err := parseTime(in.StartsAt)
	if err != nil {
		return nil, err
	}
	ends, err := parseOptionalTime(in.EndsAt)
	if err != nil {
		return nil, err
	}
	status := "rascunho"
	if in.Status != nil && validRosterStatus(*in.Status) {
		status = *in.Status
	}
	var newID string
	err = tx.QueryRow(ctx, `
		INSERT INTO rosters (tenant_id, branch_id, ministry_id, event_id, event_kind_id, title, starts_at, ends_at, location, notes, status, created_by)
		VALUES ($1, NULLIF($2,'')::uuid, NULLIF($3,'')::uuid, NULLIF($4,'')::uuid, NULLIF($5,'')::uuid, $6, $7, $8, $9, $10, $11, $12::uuid)
		RETURNING id::text`,
		tenantID, branchID, str(in.MinistryID), str(in.EventID), str(in.EventKindID),
		in.Title, starts, ends, in.Location, in.Notes, status, actorID).Scan(&newID)
	if err != nil {
		return nil, err
	}
	// Gera o evento (grade + convocados) quando pedido e não há evento vinculado.
	if in.CreateEvent && str(in.EventID) == "" && str(in.EventKindID) != "" {
		if err := r.generateEvent(ctx, tx, newID); err != nil {
			return nil, err
		}
	}
	if len(in.Assignments) > 0 {
		if err := r.SetAssignments(ctx, tx, newID, in.Assignments); err != nil {
			return nil, err
		}
	}
	if err := r.syncInvitees(ctx, tx, newID); err != nil {
		return nil, err
	}
	return r.Get(ctx, tx, newID)
}

// Update edita a escala.
func (r *Repo) Update(ctx context.Context, tx pgx.Tx, id string, in UpdateInput) (*Roster, error) {
	starts, err := parseOptionalTime(in.StartsAt)
	if err != nil {
		return nil, err
	}
	ends, err := parseOptionalTime(in.EndsAt)
	if err != nil {
		return nil, err
	}
	var updatedID string
	err = tx.QueryRow(ctx, `
		UPDATE rosters SET
			ministry_id = CASE WHEN $2::boolean THEN NULLIF($3,'')::uuid ELSE ministry_id END,
			event_id = CASE WHEN $4::boolean THEN NULLIF($5,'')::uuid ELSE event_id END,
			event_kind_id = CASE WHEN $6::boolean THEN NULLIF($7,'')::uuid ELSE event_kind_id END,
			title = COALESCE(NULLIF($8,''), title),
			starts_at = COALESCE($9::timestamptz, starts_at),
			ends_at = CASE WHEN $10::boolean THEN $11::timestamptz ELSE ends_at END,
			location = COALESCE($12, location),
			notes = COALESCE($13, notes),
			status = COALESCE($14, status),
			updated_at = now()
		WHERE id = $1::uuid
		RETURNING id::text`,
		id, in.MinistryID != nil, str(in.MinistryID), in.EventID != nil, str(in.EventID),
		in.EventKindID != nil, str(in.EventKindID),
		str(in.Title), starts, in.EndsAt != nil, ends, in.Location, in.Notes, in.Status).Scan(&updatedID)
	if err != nil {
		return nil, err
	}
	if err := r.syncInvitees(ctx, tx, updatedID); err != nil {
		return nil, err
	}
	return r.Get(ctx, tx, updatedID)
}

// generateEvent cria o evento da escala (na grade de eventos) e o vincula,
// marcando generated_event para os convocados serem sincronizados.
func (r *Repo) generateEvent(ctx context.Context, tx pgx.Tx, rosterID string) error {
	var eventID string
	err := tx.QueryRow(ctx, `
		INSERT INTO church_events (tenant_id, branch_id, kind_id, starts_at, ends_at, participants_count, attendance_mode, notes, created_by)
		SELECT r.tenant_id, r.branch_id, r.event_kind_id, r.starts_at, r.ends_at, 0, 'nominal', r.title, r.created_by
		FROM rosters r WHERE r.id = $1::uuid
		RETURNING id::text`, rosterID).Scan(&eventID)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `
		UPDATE rosters SET event_id = $2::uuid, generated_event = true, updated_at = now()
		WHERE id = $1::uuid`, rosterID, eventID)
	return err
}

// syncInvitees reflete os escalados (pessoas + ministério da escala) como
// convocados/responsabilidades do evento GERADO pela escala. Eventos externos
// (generated_event = false) não são tocados.
func (r *Repo) syncInvitees(ctx context.Context, tx pgx.Tx, rosterID string) error {
	var eventID, ministryID *string
	var generated bool
	if err := tx.QueryRow(ctx, `
		SELECT event_id::text, ministry_id::text, generated_event FROM rosters WHERE id = $1::uuid`,
		rosterID).Scan(&eventID, &ministryID, &generated); err != nil {
		return err
	}
	if eventID == nil || !generated {
		return nil
	}
	// Limpa os convocados que a escala gerencia (pessoas + o ministério dela).
	if _, err := tx.Exec(ctx, `
		DELETE FROM event_invitees
		WHERE event_id = $1::uuid
		  AND (member_id IS NOT NULL OR ($2::uuid IS NOT NULL AND ministry_id = $2::uuid))`,
		*eventID, ministryID); err != nil {
		return err
	}
	if ministryID != nil {
		if _, err := tx.Exec(ctx, `
			INSERT INTO event_invitees (tenant_id, branch_id, event_id, ministry_id)
			SELECT r.tenant_id, r.branch_id, r.event_id, r.ministry_id
			FROM rosters r WHERE r.id = $1::uuid AND r.ministry_id IS NOT NULL
			ON CONFLICT DO NOTHING`, rosterID); err != nil {
			return err
		}
	}
	_, err := tx.Exec(ctx, `
		INSERT INTO event_invitees (tenant_id, branch_id, event_id, member_id)
		SELECT r.tenant_id, r.branch_id, r.event_id, a.member_id
		FROM rosters r JOIN roster_assignments a ON a.roster_id = r.id
		WHERE r.id = $1::uuid AND r.event_id IS NOT NULL
		ON CONFLICT DO NOTHING`, rosterID)
	return err
}

// Delete remove a escala. Quando deleteEvent é true e o evento foi GERADO pela
// escala, remove também o evento da grade (com convocados e chamada por
// cascade). Eventos externos nunca são apagados por aqui.
func (r *Repo) Delete(ctx context.Context, tx pgx.Tx, id string, deleteEvent bool) error {
	var eventID *string
	var generated bool
	if err := tx.QueryRow(ctx, `
		SELECT event_id::text, generated_event FROM rosters WHERE id = $1::uuid`, id).
		Scan(&eventID, &generated); err != nil {
		return err
	}
	if deleteEvent && generated && eventID != nil {
		if _, err := tx.Exec(ctx, `DELETE FROM church_events WHERE id = $1::uuid`, *eventID); err != nil {
			return err
		}
	}
	tag, err := tx.Exec(ctx, `DELETE FROM rosters WHERE id = $1::uuid`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// SetAssignments substitui a lista de escalados, PRESERVANDO a confirmação de
// quem já estava (o status só é recriado para os novos).
func (r *Repo) SetAssignments(ctx context.Context, tx pgx.Tx, rosterID string, items []AssignmentInput) error {
	var exists bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM rosters WHERE id = $1::uuid)`, rosterID).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return pgx.ErrNoRows
	}
	// Remove os que saíram da escala.
	if len(items) == 0 {
		if _, err := tx.Exec(ctx, `DELETE FROM roster_assignments WHERE roster_id = $1::uuid`, rosterID); err != nil {
			return err
		}
		return r.syncInvitees(ctx, tx, rosterID)
	}
	if _, err := tx.Exec(ctx, `
		DELETE FROM roster_assignments
		WHERE roster_id = $1::uuid
		  AND member_id <> ALL(string_to_array($2, ',')::uuid[])`, rosterID, idList(items)); err != nil {
		return err
	}
	for _, it := range items {
		if _, err := tx.Exec(ctx, `
			INSERT INTO roster_assignments (tenant_id, branch_id, roster_id, member_id, role)
			SELECT r.tenant_id, r.branch_id, r.id, m.id, $3
			FROM rosters r
			JOIN members m ON m.id = $2::uuid AND m.tenant_id = r.tenant_id AND m.branch_id = r.branch_id
			WHERE r.id = $1::uuid
			ON CONFLICT (roster_id, member_id) DO UPDATE SET role = EXCLUDED.role`,
			rosterID, it.MemberID, it.Role); err != nil {
			return err
		}
	}
	return r.syncInvitees(ctx, tx, rosterID)
}

// Respond registra a confirmação/recusa de um escalado.
func (r *Repo) Respond(ctx context.Context, tx pgx.Tx, rosterID, assignmentID, status string, notes *string) (*Assignment, error) {
	if status != "confirmado" && status != "recusado" && status != "convidado" {
		return nil, ErrInvalidStatus
	}
	var a Assignment
	err := tx.QueryRow(ctx, `
		UPDATE roster_assignments SET
			status = $3,
			responded_at = CASE WHEN $3 = 'convidado' THEN NULL ELSE now() END,
			notes = COALESCE($4, notes)
		WHERE id = $1::uuid AND roster_id = $2::uuid
		RETURNING id::text, roster_id::text, member_id::text,
			(SELECT full_name FROM members WHERE id = member_id), role, status, responded_at, notes`,
		assignmentID, rosterID, status, notes).Scan(&a.ID, &a.RosterID, &a.MemberID, &a.MemberName, &a.Role, &a.Status, &a.RespondedAt, &a.Notes)
	return &a, err
}

// Conflicts lista os choques de agenda dos escalados desta escala com outras
// escalas (não canceladas) que se sobrepõem no tempo.
func (r *Repo) Conflicts(ctx context.Context, tx pgx.Tx, rosterID string) ([]Conflict, error) {
	rows, err := tx.Query(ctx, `
		SELECT DISTINCT a.member_id::text, mb.full_name, r2.id::text, r2.title, r2.starts_at
		FROM roster_assignments a
		JOIN rosters r ON r.id = a.roster_id
		JOIN members mb ON mb.id = a.member_id
		JOIN roster_assignments a2 ON a2.member_id = a.member_id AND a2.roster_id <> r.id
		JOIN rosters r2 ON r2.id = a2.roster_id AND r2.status <> 'cancelada'
		WHERE r.id = $1::uuid
		  AND r2.starts_at < COALESCE(r.ends_at, r.starts_at + interval '2 hours')
		  AND COALESCE(r2.ends_at, r2.starts_at + interval '2 hours') > r.starts_at
		ORDER BY mb.full_name`, rosterID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Conflict{}
	for rows.Next() {
		var c Conflict
		if err := rows.Scan(&c.MemberID, &c.MemberName, &c.OtherID, &c.OtherTitle, &c.OtherStarts); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// Suggestions lista candidatos de um ministério para uma janela de tempo,
// marcando quem já está ocupado (conflito) e ordenando por frequência recente.
func (r *Repo) Suggestions(ctx context.Context, tx pgx.Tx, ministryID, startsAt, endsAt, excludeRosterID string) ([]Suggestion, error) {
	if ministryID == "" {
		return []Suggestion{}, nil
	}
	var ends any
	if endsAt != "" {
		ends = endsAt
	}
	rows, err := tx.Query(ctx, `
		SELECT mb.id::text, mb.full_name, f.frequency,
			EXISTS (
				SELECT 1 FROM roster_assignments a
				JOIN rosters r ON r.id = a.roster_id AND r.status <> 'cancelada'
				WHERE a.member_id = mb.id
				  AND ($4 = '' OR r.id <> $4::uuid)
				  AND r.starts_at < COALESCE($3::timestamptz, $2::timestamptz + interval '2 hours')
				  AND COALESCE(r.ends_at, r.starts_at + interval '2 hours') > $2::timestamptz
			) AS busy
		FROM ministry_members mm
		JOIN members mb ON mb.id = mm.member_id
		LEFT JOIN member_frequency_history f ON f.member_id = mb.id AND f.ended_at IS NULL
		WHERE mm.ministry_id = $1::uuid
		ORDER BY (f.frequency IS DISTINCT FROM 'frequente'), mb.full_name`,
		ministryID, startsAt, ends, excludeRosterID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Suggestion{}
	for rows.Next() {
		var s Suggestion
		if err := rows.Scan(&s.MemberID, &s.MemberName, &s.Frequency, &s.Busy); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// ---- helpers ----

func validRosterStatus(s string) bool {
	switch s {
	case "rascunho", "publicada", "concluida", "cancelada":
		return true
	}
	return false
}

func idList(items []AssignmentInput) string {
	out := ""
	for _, it := range items {
		if it.MemberID == "" {
			continue
		}
		if out != "" {
			out += ","
		}
		out += it.MemberID
	}
	return out
}

func parseTime(s string) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	return time.Parse("2006-01-02T15:04", s)
}

func parseOptionalTime(s *string) (*time.Time, error) {
	if s == nil || *s == "" {
		return nil, nil
	}
	t, err := parseTime(*s)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func str(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func nullStr(s *string) any {
	if s == nil || *s == "" {
		return nil
	}
	return *s
}
