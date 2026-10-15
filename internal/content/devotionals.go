// Package content gerencia a area de conteudo do membro: devocionais (planos
// com dias + progresso), videos do YouTube e a config do culto ao vivo.
// O isolamento multi-tenant e do RLS; o escopo "igreja toda" usa branch_id NULL.
package content

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
)

// ---- Devocionais ----

// Plan e um plano devocional (conjunto ordenado de dias).
type Plan struct {
	ID            string    `json:"id"`
	BranchID      *string   `json:"branch_id,omitempty"`
	Title         string    `json:"title"`
	Description   *string   `json:"description,omitempty"`
	CoverImageURL *string   `json:"cover_image_url,omitempty"`
	StartDate     *string   `json:"start_date,omitempty"`
	IsActive      bool      `json:"is_active"`
	DayCount      int       `json:"day_count"`
	CreatedAt     time.Time `json:"created_at"`
}

// Day e um dia do plano devocional.
type Day struct {
	ID             string  `json:"id"`
	DayNumber      int     `json:"day_number"`
	Title          string  `json:"title"`
	BibleReference *string `json:"bible_reference,omitempty"`
	BibleText      *string `json:"bible_text,omitempty"`
	Body           string  `json:"body"`
	Prayer         *string `json:"prayer,omitempty"`
	VideoURL       *string `json:"video_url,omitempty"`
}

// PlanInput e o corpo de criacao/edicao de um plano.
type PlanInput struct {
	Title         string  `json:"title"`
	Description   *string `json:"description"`
	CoverImageURL *string `json:"cover_image_url"`
	StartDate     *string `json:"start_date"`
	IsActive      *bool   `json:"is_active"`
	BranchID      *string `json:"branch_id"`
}

// DayInput e um dia enviado no lote (substitui os dias do plano).
type DayInput struct {
	DayNumber      int     `json:"day_number"`
	Title          string  `json:"title"`
	BibleReference *string `json:"bible_reference"`
	BibleText      *string `json:"bible_text"`
	Body           string  `json:"body"`
	Prayer         *string `json:"prayer"`
	VideoURL       *string `json:"video_url"`
}

const planCols = `p.id::text, p.branch_id::text, p.title, p.description, p.cover_image_url,
	p.start_date::text, p.is_active,
	(SELECT count(*) FROM devotional_plan_days d WHERE d.plan_id = p.id)::int, p.created_at`

func scanPlan(row pgx.Row) (*Plan, error) {
	var p Plan
	err := row.Scan(&p.ID, &p.BranchID, &p.Title, &p.Description, &p.CoverImageURL,
		&p.StartDate, &p.IsActive, &p.DayCount, &p.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// ListPlans lista os planos do escopo. activeOnly restringe aos ativos.
func (r *Repo) ListPlans(ctx context.Context, tx pgx.Tx, activeOnly bool) ([]Plan, error) {
	rows, err := tx.Query(ctx, `
		SELECT `+planCols+`
		FROM devotional_plans p
		WHERE (NOT $1 OR p.is_active)
		ORDER BY p.created_at DESC`, activeOnly)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Plan{}
	for rows.Next() {
		p, err := scanPlan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *p)
	}
	return out, rows.Err()
}

// GetPlan carrega um plano pelo id.
func (r *Repo) GetPlan(ctx context.Context, tx pgx.Tx, id string) (*Plan, error) {
	return scanPlan(tx.QueryRow(ctx, `SELECT `+planCols+` FROM devotional_plans p WHERE p.id = $1::uuid`, id))
}

// ListDays devolve os dias ordenados de um plano.
func (r *Repo) ListDays(ctx context.Context, tx pgx.Tx, planID string) ([]Day, error) {
	rows, err := tx.Query(ctx, `
		SELECT id::text, day_number, title, bible_reference, bible_text, body, prayer, video_url
		FROM devotional_plan_days
		WHERE plan_id = $1::uuid
		ORDER BY day_number`, planID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Day{}
	for rows.Next() {
		var d Day
		if err := rows.Scan(&d.ID, &d.DayNumber, &d.Title, &d.BibleReference, &d.BibleText,
			&d.Body, &d.Prayer, &d.VideoURL); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// CreatePlan insere um plano no escopo RLS da sessao.
func (r *Repo) CreatePlan(ctx context.Context, tx pgx.Tx, userID string, in PlanInput) (*Plan, error) {
	active := true
	if in.IsActive != nil {
		active = *in.IsActive
	}
	var id string
	err := tx.QueryRow(ctx, `
		INSERT INTO devotional_plans
			(tenant_id, branch_id, title, description, cover_image_url, start_date, is_active, created_by)
		VALUES (current_tenant(), NULLIF($1,'')::uuid, $2, $3, $4, NULLIF($5,'')::date, $6, NULLIF($7,'')::uuid)
		RETURNING id::text`,
		deref(in.BranchID), in.Title, in.Description, in.CoverImageURL, deref(in.StartDate), active, userID).Scan(&id)
	if err != nil {
		return nil, err
	}
	return r.GetPlan(ctx, tx, id)
}

// UpdatePlan edita um plano (titulo vazio mantem).
func (r *Repo) UpdatePlan(ctx context.Context, tx pgx.Tx, id string, in PlanInput) (*Plan, error) {
	tag, err := tx.Exec(ctx, `
		UPDATE devotional_plans SET
			title = COALESCE(NULLIF(btrim($2),''), title),
			description = COALESCE($3, description),
			cover_image_url = COALESCE($4, cover_image_url),
			start_date = COALESCE(NULLIF($5,'')::date, start_date),
			is_active = COALESCE($6, is_active),
			updated_at = now()
		WHERE id = $1::uuid`,
		id, in.Title, in.Description, in.CoverImageURL, deref(in.StartDate), in.IsActive)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, pgx.ErrNoRows
	}
	return r.GetPlan(ctx, tx, id)
}

// DeletePlan remove o plano (dias/progresso caem em cascata).
func (r *Repo) DeletePlan(ctx context.Context, tx pgx.Tx, id string) error {
	tag, err := tx.Exec(ctx, `DELETE FROM devotional_plans WHERE id = $1::uuid`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// ReplaceDays substitui todos os dias do plano pelo lote informado.
func (r *Repo) ReplaceDays(ctx context.Context, tx pgx.Tx, planID string, days []DayInput) error {
	if _, err := tx.Exec(ctx, `DELETE FROM devotional_plan_days WHERE plan_id = $1::uuid`, planID); err != nil {
		return err
	}
	for _, d := range days {
		if _, err := tx.Exec(ctx, `
			INSERT INTO devotional_plan_days
				(tenant_id, branch_id, plan_id, day_number, title, bible_reference, bible_text, body, prayer, video_url)
			SELECT p.tenant_id, p.branch_id, p.id, $2, $3, $4, $5, $6, $7, $8
			FROM devotional_plans p WHERE p.id = $1::uuid`,
			planID, d.DayNumber, d.Title, d.BibleReference, d.BibleText, d.Body, d.Prayer, d.VideoURL); err != nil {
			return err
		}
	}
	return nil
}

// CompletedDays devolve os numeros de dia concluidos pelo usuario.
func (r *Repo) CompletedDays(ctx context.Context, tx pgx.Tx, planID, userID string) ([]int, error) {
	rows, err := tx.Query(ctx, `
		SELECT day_number FROM devotional_progress
		WHERE plan_id = $1::uuid AND user_id = $2::uuid
		ORDER BY day_number`, planID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []int{}
	for rows.Next() {
		var n int
		if err := rows.Scan(&n); err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

// SetDayComplete marca/desmarca um dia como concluido pelo usuario.
func (r *Repo) SetDayComplete(ctx context.Context, tx pgx.Tx, tenantID, planID, userID, memberID string, day int, complete bool) error {
	if !complete {
		_, err := tx.Exec(ctx, `
			DELETE FROM devotional_progress
			WHERE plan_id = $1::uuid AND user_id = $2::uuid AND day_number = $3`,
			planID, userID, day)
		return err
	}
	_, err := tx.Exec(ctx, `
		INSERT INTO devotional_progress (tenant_id, plan_id, user_id, member_id, day_number)
		VALUES ($1::uuid, $2::uuid, $3::uuid, NULLIF($4,'')::uuid, $5)
		ON CONFLICT (plan_id, user_id, day_number) DO NOTHING`,
		tenantID, planID, userID, memberID, day)
	return err
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
