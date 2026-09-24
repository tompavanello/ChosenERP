package announcements

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
)

// Run é uma execução de comunicado agendado (histórico).
type Run struct {
	ID                string    `json:"id"`
	AnnouncementID    string    `json:"announcement_id"`
	AnnouncementTitle string    `json:"announcement_title"`
	ScheduleType      string    `json:"schedule_type"`
	PeriodKey         *string   `json:"period_key,omitempty"`
	RecipientCount    int       `json:"recipient_count"`
	FiredAt           time.Time `json:"fired_at"`
}

// CreateRun registra uma execução do agendamento.
func (r *Repo) CreateRun(ctx context.Context, tx pgx.Tx, a *Announcement, periodKey string, count int) error {
	var p *string
	if periodKey != "" {
		p = &periodKey
	}
	_, err := tx.Exec(ctx, `
		INSERT INTO announcement_runs
			(tenant_id, branch_id, announcement_id, schedule_type, period_key, recipient_count)
		VALUES ($1, NULLIF($2,'')::uuid, $3::uuid, $4, $5, $6)`,
		a.TenantID, a.BranchID, a.ID, a.ScheduleType, p, count)
	return err
}

// ListRuns devolve as execuções mais recentes no escopo RLS da sessão.
func (r *Repo) ListRuns(ctx context.Context, tx pgx.Tx, limit int) ([]Run, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := tx.Query(ctx, `
		SELECT r.id::text, r.announcement_id::text, COALESCE(a.title,''),
		       r.schedule_type, r.period_key, r.recipient_count, r.fired_at
		FROM announcement_runs r
		LEFT JOIN announcements a ON a.id = r.announcement_id
		ORDER BY r.fired_at DESC
		LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Run{}
	for rows.Next() {
		var run Run
		if err := rows.Scan(&run.ID, &run.AnnouncementID, &run.AnnouncementTitle,
			&run.ScheduleType, &run.PeriodKey, &run.RecipientCount, &run.FiredAt); err != nil {
			return nil, err
		}
		out = append(out, run)
	}
	return out, rows.Err()
}
