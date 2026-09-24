package announcements

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// ErrInvalidInput marca erros de validacao do input (viram HTTP 400).
var ErrInvalidInput = errors.New("invalid input")

// Announcement e um aviso/publicacao do app do membro. A partir da migracao
// 000041 tambem guarda a segmentacao e o agendamento do disparo.
type Announcement struct {
	ID                    string         `json:"id"`
	TenantID              string         `json:"-"`
	BranchID              string         `json:"branch_id"`
	Title                 string         `json:"title"`
	Body                  string         `json:"body"`
	Audience              string         `json:"audience"`
	AudienceFilter        AudienceFilter `json:"audience_filter"`
	Channel               string         `json:"channel"`
	ScheduleType          string         `json:"schedule_type"`
	ScheduleAt            *time.Time     `json:"schedule_at,omitempty"`
	ScheduleTime          *string        `json:"schedule_time,omitempty"`
	ScheduleEventID       *string        `json:"schedule_event_id,omitempty"`
	ScheduleOffsetMinutes int            `json:"schedule_offset_minutes"`
	LastRunAt             *time.Time     `json:"last_run_at,omitempty"`
	RunCount              int            `json:"run_count"`
	IsActive              bool           `json:"is_active"`
	PublishedAt           time.Time      `json:"published_at"`
	CreatedAt             time.Time      `json:"created_at"`
}

// UpsertInput e o corpo de criacao/edicao. Campos nulos (ponteiro nil) nao sao
// alterados - permite PATCH parcial.
type UpsertInput struct {
	Title                 *string         `json:"title"`
	Body                  *string         `json:"body"`
	Audience              *string         `json:"audience"`
	AudienceFilter        *AudienceFilter `json:"audience_filter"`
	Channel               *string         `json:"channel"`
	ScheduleType          *string         `json:"schedule_type"`
	ScheduleAt            *string         `json:"schedule_at"`
	ScheduleTime          *string         `json:"schedule_time"`
	ScheduleEventID       *string         `json:"schedule_event_id"`
	ScheduleOffsetMinutes *int            `json:"schedule_offset_minutes"`
	IsActive              *bool           `json:"is_active"`
}

type Repo struct{}

// announcementSelect e a lista de colunas usada por todas as leituras.
const announcementSelect = `
	id::text, tenant_id::text, COALESCE(branch_id::text,''), title, body, audience,
	audience_filter::text, channel, schedule_type, schedule_at,
	to_char(schedule_time,'HH24:MI'), schedule_event_id::text,
	schedule_offset_minutes, last_run_at, run_count, is_active, published_at, created_at`

type rowScanner interface{ Scan(dest ...any) error }

func scanAnnouncement(s rowScanner) (*Announcement, error) {
	var a Announcement
	var filterJSON string
	if err := s.Scan(&a.ID, &a.TenantID, &a.BranchID, &a.Title, &a.Body, &a.Audience,
		&filterJSON, &a.Channel, &a.ScheduleType, &a.ScheduleAt,
		&a.ScheduleTime, &a.ScheduleEventID,
		&a.ScheduleOffsetMinutes, &a.LastRunAt, &a.RunCount, &a.IsActive,
		&a.PublishedAt, &a.CreatedAt); err != nil {
		return nil, err
	}
	if filterJSON != "" {
		_ = json.Unmarshal([]byte(filterJSON), &a.AudienceFilter)
	}
	return &a, nil
}

// List retorna avisos ativos e publicados do escopo (RLS) da sessao.
func (r *Repo) List(ctx context.Context, tx pgx.Tx) ([]Announcement, error) {
	rows, err := tx.Query(ctx, `
		SELECT `+announcementSelect+`
		FROM announcements
		WHERE is_active AND published_at <= now()
		ORDER BY published_at DESC LIMIT 100`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Announcement{}
	for rows.Next() {
		a, err := scanAnnouncement(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *a)
	}
	return out, rows.Err()
}

// GetByID retorna um comunicado pelo ID no escopo RLS da sessao.
func (r *Repo) GetByID(ctx context.Context, tx pgx.Tx, id string) (*Announcement, error) {
	return scanAnnouncement(tx.QueryRow(ctx, `
		SELECT `+announcementSelect+` FROM announcements WHERE id = $1::uuid`, id))
}

// Delete remove o comunicado (as entregas e execucoes caem em cascata).
func (r *Repo) Delete(ctx context.Context, tx pgx.Tx, id string) error {
	tag, err := tx.Exec(ctx, `DELETE FROM announcements WHERE id = $1::uuid`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// Create insere um aviso no escopo RLS da sessao.
func (r *Repo) Create(ctx context.Context, tx pgx.Tx, tenantID, branchID string, in UpsertInput) (*Announcement, error) {
	a := &Announcement{
		TenantID:       tenantID,
		BranchID:       branchID,
		Audience:       "everyone",
		Channel:        "whatsapp",
		ScheduleType:   "manual",
		IsActive:       true,
		AudienceFilter: AudienceFilter{},
	}
	if err := a.apply(in, true); err != nil {
		return nil, err
	}
	filter, _ := json.Marshal(a.AudienceFilter)
	return scanAnnouncement(tx.QueryRow(ctx, `
		INSERT INTO announcements
			(tenant_id, branch_id, title, body, audience, audience_filter, channel,
			 schedule_type, schedule_at, schedule_time, schedule_event_id,
			 schedule_offset_minutes, is_active)
		VALUES ($1, NULLIF($2,'')::uuid, $3, $4, $5, $6::jsonb, $7,
		        $8, $9, NULLIF($10,'')::time, NULLIF($11,'')::uuid, $12, $13)
		RETURNING `+announcementSelect,
		tenantID, branchID, a.Title, a.Body, a.Audience, string(filter), a.Channel,
		a.ScheduleType, a.ScheduleAt, strOrEmpty(a.ScheduleTime), strOrEmpty(a.ScheduleEventID),
		a.ScheduleOffsetMinutes, a.IsActive))
}

// Update aplica um PATCH parcial sobre o comunicado existente.
func (r *Repo) Update(ctx context.Context, tx pgx.Tx, id string, in UpsertInput) (*Announcement, error) {
	a, err := scanAnnouncement(tx.QueryRow(ctx, `
		SELECT `+announcementSelect+` FROM announcements WHERE id = $1::uuid`, id))
	if err != nil {
		return nil, err
	}
	if err := a.apply(in, false); err != nil {
		return nil, err
	}
	filter, _ := json.Marshal(a.AudienceFilter)
	return scanAnnouncement(tx.QueryRow(ctx, `
		UPDATE announcements SET
			title = $2, body = $3, audience = $4, audience_filter = $5::jsonb,
			channel = $6, schedule_type = $7, schedule_at = $8,
			schedule_time = NULLIF($9,'')::time, schedule_event_id = NULLIF($10,'')::uuid,
			schedule_offset_minutes = $11, is_active = $12
		WHERE id = $1::uuid
		RETURNING `+announcementSelect,
		id, a.Title, a.Body, a.Audience, string(filter), a.Channel,
		a.ScheduleType, a.ScheduleAt, strOrEmpty(a.ScheduleTime), strOrEmpty(a.ScheduleEventID),
		a.ScheduleOffsetMinutes, a.IsActive))
}

// apply valida e mescla o input. `requireTitle` existe para a criacao, onde o
// titulo e obrigatorio.
func (a *Announcement) apply(in UpsertInput, requireTitle bool) error {
	if in.Title != nil {
		a.Title = *in.Title
	}
	if in.Body != nil {
		a.Body = *in.Body
	}
	if in.Audience != nil && *in.Audience != "" {
		a.Audience = *in.Audience
	}
	if in.AudienceFilter != nil {
		a.AudienceFilter = *in.AudienceFilter
	}
	if in.Channel != nil && *in.Channel != "" {
		a.Channel = *in.Channel
	}
	if in.ScheduleOffsetMinutes != nil {
		a.ScheduleOffsetMinutes = *in.ScheduleOffsetMinutes
	}
	if in.IsActive != nil {
		a.IsActive = *in.IsActive
	}

	if in.ScheduleType != nil && *in.ScheduleType != "" {
		a.ScheduleType = *in.ScheduleType
	}
	switch a.ScheduleType {
	case "manual":
		a.ScheduleAt, a.ScheduleTime, a.ScheduleEventID = nil, nil, nil
	case "once":
		if in.ScheduleAt != nil {
			a.ScheduleAt = nil
			if *in.ScheduleAt != "" {
				t, err := time.Parse(time.RFC3339, *in.ScheduleAt)
				if err != nil {
					return fmt.Errorf("%w: schedule_at invalido: use RFC3339", ErrInvalidInput)
				}
				a.ScheduleAt = &t
			}
		}
		if a.ScheduleAt == nil {
			return fmt.Errorf("%w: schedule_at e obrigatorio para agendamento unico", ErrInvalidInput)
		}
		a.ScheduleTime, a.ScheduleEventID = nil, nil
	case "daily":
		if in.ScheduleTime != nil {
			a.ScheduleTime = nil
			if *in.ScheduleTime != "" {
				if _, err := time.Parse("15:04", *in.ScheduleTime); err != nil {
					return fmt.Errorf("%w: schedule_time invalido: use HH:MM", ErrInvalidInput)
				}
				s := *in.ScheduleTime
				a.ScheduleTime = &s
			}
		}
		if a.ScheduleTime == nil {
			return fmt.Errorf("%w: schedule_time e obrigatorio para agendamento diario", ErrInvalidInput)
		}
		a.ScheduleAt, a.ScheduleEventID = nil, nil
	case "event":
		if in.ScheduleEventID != nil {
			a.ScheduleEventID = nil
			if *in.ScheduleEventID != "" {
				s := *in.ScheduleEventID
				a.ScheduleEventID = &s
			}
		}
		if a.ScheduleEventID == nil {
			return fmt.Errorf("%w: schedule_event_id e obrigatorio para agendamento por evento", ErrInvalidInput)
		}
		a.ScheduleAt, a.ScheduleTime = nil, nil
	default:
		return fmt.Errorf("%w: schedule_type invalido: %q", ErrInvalidInput, a.ScheduleType)
	}

	if requireTitle && a.Title == "" {
		return fmt.Errorf("%w: title e obrigatorio", ErrInvalidInput)
	}
	return nil
}

func strOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// ScheduledAnnouncement e o comunicado agendado visto pelo scheduler, ja com os
// dados da igreja e (quando for o caso) o inicio do evento vinculado.
type ScheduledAnnouncement struct {
	Announcement
	TenantName    string
	Timezone      string
	EventStartsAt *time.Time
}

// ListScheduled devolve os comunicados ativos com agendamento (para o worker).
func (r *Repo) ListScheduled(ctx context.Context, tx pgx.Tx) ([]ScheduledAnnouncement, error) {
	rows, err := tx.Query(ctx, `
		SELECT a.id::text, a.tenant_id::text, COALESCE(a.branch_id::text,''), a.title, a.body,
		       a.audience, a.audience_filter::text, a.channel, a.schedule_type, a.schedule_at,
		       to_char(a.schedule_time,'HH24:MI'), a.schedule_event_id::text,
		       a.schedule_offset_minutes, a.last_run_at, a.run_count, a.is_active,
		       a.published_at, a.created_at,
		       t.name, COALESCE(NULLIF(t.timezone,''),'America/Sao_Paulo'), e.starts_at
		FROM announcements a
		JOIN tenants t ON t.id = a.tenant_id
		LEFT JOIN church_events e ON e.id = a.schedule_event_id
		WHERE a.schedule_type <> 'manual' AND a.is_active
		ORDER BY a.tenant_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ScheduledAnnouncement{}
	for rows.Next() {
		var s ScheduledAnnouncement
		var filterJSON string
		if err := rows.Scan(&s.ID, &s.TenantID, &s.BranchID, &s.Title, &s.Body,
			&s.Audience, &filterJSON, &s.Channel, &s.ScheduleType, &s.ScheduleAt,
			&s.ScheduleTime, &s.ScheduleEventID, &s.ScheduleOffsetMinutes,
			&s.LastRunAt, &s.RunCount, &s.IsActive, &s.PublishedAt, &s.CreatedAt,
			&s.TenantName, &s.Timezone, &s.EventStartsAt); err != nil {
			return nil, err
		}
		if filterJSON != "" {
			_ = json.Unmarshal([]byte(filterJSON), &s.AudienceFilter)
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// CreateScheduledDelivery enfileira o comunicado para um destinatario do
// agendamento. A chave de dedupe evita reenviar o mesmo disparo no mesmo
// periodo (dia/evento); devolve false quando ja existia.
func (r *Repo) CreateScheduledDelivery(ctx context.Context, tx pgx.Tx, a *Announcement, provider, recipient, recipientName, dedupeKey string) (bool, error) {
	var id string
	err := tx.QueryRow(ctx, `
		INSERT INTO announcement_deliveries
			(tenant_id, branch_id, announcement_id, channel, provider, recipient,
			 recipient_name, source, dedupe_key)
		VALUES ($1, NULLIF($2,'')::uuid, $3::uuid, $4, $5, $6, $7, 'schedule', $8)
		ON CONFLICT (tenant_id, dedupe_key) WHERE dedupe_key IS NOT NULL DO NOTHING
		RETURNING id::text`,
		a.TenantID, a.BranchID, a.ID, a.Channel, provider, recipient, recipientName, dedupeKey).Scan(&id)
	if err == pgx.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// MarkRun registra que o comunicado foi disparado por agendamento.
func (r *Repo) MarkRun(ctx context.Context, tx pgx.Tx, id string) error {
	_, err := tx.Exec(ctx, `
		UPDATE announcements
		SET last_run_at = now(), run_count = run_count + 1
		WHERE id = $1::uuid`, id)
	return err
}
