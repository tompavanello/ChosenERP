package visitors

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"

	"chosenerp/internal/members"
)

// Visitante e trilha de acolhimento.
type Visitor struct {
	ID           string    `json:"id"`
	BranchID     string    `json:"branch_id"`
	FirstName    string    `json:"first_name"`
	LastName     string    `json:"last_name"`
	FullName     string    `json:"full_name"`
	Email        *string   `json:"email,omitempty"`
	Phone        *string   `json:"phone,omitempty"`
	Whatsapp     *string   `json:"whatsapp,omitempty"`
	Source       *string   `json:"source,omitempty"`
	JourneyStage string    `json:"journey_stage"`
	CreatedAt    time.Time `json:"created_at"`
}

type CreateInput struct {
	FirstName string  `json:"first_name"`
	LastName  string  `json:"last_name"`
	Email     *string `json:"email"`
	Phone     *string `json:"phone"`
	Whatsapp  *string `json:"whatsapp"`
	Source    *string `json:"source"`
}

type Repo struct{}

func (r *Repo) List(ctx context.Context, tx pgx.Tx) ([]Visitor, error) {
	rows, err := tx.Query(ctx, `
		SELECT id::text, branch_id::text, first_name, last_name,
		       (first_name || ' ' || last_name) AS full_name, email, phone, whatsapp,
		       source, journey_stage, created_at
		FROM visitors ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Visitor{}
	for rows.Next() {
		var v Visitor
		if err := rows.Scan(&v.ID, &v.BranchID, &v.FirstName, &v.LastName, &v.FullName,
			&v.Email, &v.Phone, &v.Whatsapp, &v.Source, &v.JourneyStage, &v.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (r *Repo) Create(ctx context.Context, tx pgx.Tx, tenantID, branchID string, in CreateInput) (*Visitor, error) {
	var v Visitor
	err := tx.QueryRow(ctx, `
		INSERT INTO visitors (tenant_id, branch_id, first_name, last_name, email, phone, whatsapp, source)
		VALUES ($1, $2::uuid, $3, $4, $5, $6, $7, $8)
		RETURNING id::text, branch_id::text, first_name, last_name,
		          (first_name || ' ' || last_name) AS full_name, email, phone, whatsapp,
		          source, journey_stage, created_at`,
		tenantID, branchID, in.FirstName, in.LastName, in.Email, in.Phone, in.Whatsapp, in.Source).
		Scan(&v.ID, &v.BranchID, &v.FirstName, &v.LastName, &v.FullName,
			&v.Email, &v.Phone, &v.Whatsapp, &v.Source, &v.JourneyStage, &v.CreatedAt)
	return &v, err
}

func (r *Repo) Get(ctx context.Context, tx pgx.Tx, id string) (*Visitor, error) {
	var v Visitor
	err := tx.QueryRow(ctx, `
		SELECT id::text, branch_id::text, first_name, last_name,
		       (first_name || ' ' || last_name) AS full_name, email, phone, whatsapp,
		       source, journey_stage, created_at
		FROM visitors WHERE id = $1::uuid`, id).
		Scan(&v.ID, &v.BranchID, &v.FirstName, &v.LastName, &v.FullName,
			&v.Email, &v.Phone, &v.Whatsapp, &v.Source, &v.JourneyStage, &v.CreatedAt)
	return &v, err
}

// UpdateStage avanca a trilha de acolhimento.
func (r *Repo) UpdateStage(ctx context.Context, tx pgx.Tx, id, stage string) (*Visitor, error) {
	var v Visitor
	err := tx.QueryRow(ctx, `
		UPDATE visitors SET journey_stage = $2, updated_at = now() WHERE id = $1::uuid
		RETURNING id::text, branch_id::text, first_name, last_name,
		          (first_name || ' ' || last_name) AS full_name, email, phone, whatsapp,
		          source, journey_stage, created_at`,
		id, stage).
		Scan(&v.ID, &v.BranchID, &v.FirstName, &v.LastName, &v.FullName,
			&v.Email, &v.Phone, &v.Whatsapp, &v.Source, &v.JourneyStage, &v.CreatedAt)
	return &v, err
}

// Convert transforma um visitante em membro. Cria o membro (via members.Repo,
// para nao duplicar a projecao/insercao) e marca o visitante como convertido.
func (r *Repo) Convert(ctx context.Context, tx pgx.Tx, visitorID string, tenantID, branchID string, in members.CreateInput, actorID string) (*members.Member, error) {
	m, err := (&members.Repo{}).Create(ctx, tx, tenantID, branchID, in, actorID)
	if err != nil {
		return nil, err
	}

	// Mark the visitor as converted
	_, _ = tx.Exec(ctx, `
		UPDATE visitors SET journey_stage = 'converted', converted_to_member_id = $1::uuid, updated_at = now()
		WHERE id = $2::uuid`,
		m.ID, visitorID)

	return m, nil
}

// RevertConversion reverte a conversao de um visitante em membro (remove o membro criado).
func (r *Repo) RevertConversion(ctx context.Context, tx pgx.Tx, visitorID string) error {
	var memberID string
	err := tx.QueryRow(ctx, `
		SELECT converted_to_member_id::text FROM visitors WHERE id = $1::uuid`, visitorID).
		Scan(&memberID)
	if err != nil {
		return err
	}

	// Remove the created member
	_, _ = tx.Exec(ctx, `DELETE FROM members WHERE id = $1::uuid`, memberID)

	// Revert the visitor
	_, _ = tx.Exec(ctx, `
		UPDATE visitors SET journey_stage = 'welcome', converted_to_member_id = NULL, updated_at = now()
		WHERE id = $1::uuid`,
		visitorID)

	return nil
}

// ContactInput e o que o proprio visitante pode editar no app (nome/e-mail e
// WhatsApp). O telefone e identificador de login e nao muda por aqui.
type ContactInput struct {
	FirstName *string `json:"first_name"`
	LastName  *string `json:"last_name"`
	Email     *string `json:"email"`
	Whatsapp  *string `json:"whatsapp"`
}

// UpdateContact atualiza os dados de contato do visitante (escopo self).
func (r *Repo) UpdateContact(ctx context.Context, tx pgx.Tx, visitorID string, in ContactInput) (*Visitor, error) {
	var v Visitor
	err := tx.QueryRow(ctx, `
		UPDATE visitors SET
			first_name = COALESCE(NULLIF(btrim($2),''), first_name),
			last_name  = COALESCE(NULLIF(btrim($3),''), last_name),
			email      = COALESCE($4, email),
			whatsapp   = COALESCE($5, whatsapp),
			updated_at = now()
		WHERE id = $1::uuid
		RETURNING id::text, branch_id::text, first_name, last_name,
		          (first_name || ' ' || last_name) AS full_name, email, phone, whatsapp,
		          source, journey_stage, created_at`,
		visitorID, deref(in.FirstName), deref(in.LastName), in.Email, in.Whatsapp).
		Scan(&v.ID, &v.BranchID, &v.FirstName, &v.LastName, &v.FullName,
			&v.Email, &v.Phone, &v.Whatsapp, &v.Source, &v.JourneyStage, &v.CreatedAt)
	return &v, err
}

// JourneyStep e uma transicao registrada na trilha de acolhimento.
type JourneyStep struct {
	ID         string    `json:"id"`
	FromStage  *string   `json:"from_stage,omitempty"`
	ToStage    string    `json:"to_stage"`
	Notes      *string   `json:"notes,omitempty"`
	ChangedBy  *string   `json:"changed_by,omitempty"`
	OccurredAt time.Time `json:"occurred_at"`
}

// ListJourney devolve a trilha de etapas do visitante (mais recente primeiro).
func (r *Repo) ListJourney(ctx context.Context, tx pgx.Tx, visitorID string) ([]JourneyStep, error) {
	rows, err := tx.Query(ctx, `
		SELECT id::text, from_stage, to_stage, notes, changed_by::text, occurred_at
		FROM visitor_journey
		WHERE visitor_id = $1::uuid
		ORDER BY occurred_at DESC, created_at DESC`, visitorID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []JourneyStep{}
	for rows.Next() {
		var s JourneyStep
		if err := rows.Scan(&s.ID, &s.FromStage, &s.ToStage, &s.Notes, &s.ChangedBy, &s.OccurredAt); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// PromoteToMember promove o acesso do visitante a membro (aprovacao da
// secretaria), preservando a identidade e ligando o cadastro de membro.
func (r *Repo) PromoteToMember(ctx context.Context, tx pgx.Tx, userID, tenantID, memberID string) error {
	_, err := tx.Exec(ctx,
		`SELECT promote_visitor_to_member($1::uuid, $2::uuid, $3::uuid)`,
		userID, tenantID, memberID)
	return err
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
