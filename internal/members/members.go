package members

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
)

type Member struct {
	ID               string    `json:"id"`
	BranchID         string    `json:"branch_id"`
	FirstName        string    `json:"first_name"`
	LastName         string    `json:"last_name"`
	FullName         string    `json:"full_name"`
	Nickname         *string   `json:"nickname,omitempty"`
	Email            *string   `json:"email,omitempty"`
	Phone            *string   `json:"phone,omitempty"`
	Whatsapp         *string   `json:"whatsapp,omitempty"`
	BirthDate        *string   `json:"birth_date,omitempty"`
	Gender           *string   `json:"gender,omitempty"`
	MaritalStatus    *string   `json:"marital_status,omitempty"`
	MembershipStatus string    `json:"membership_status"`
	Profession       *string   `json:"profession,omitempty"`
	Office           *string   `json:"office,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
}

type Repo struct{}

// List retorna membros do escopo definido pela transação (RLS).
func (r *Repo) List(ctx context.Context, tx pgx.Tx) ([]Member, error) {
	rows, err := tx.Query(ctx, `
		SELECT id::text, branch_id::text, first_name, last_name, full_name,
		       nickname, email, phone, whatsapp, birth_date::text, gender,
		       marital_status, membership_status, profession, office, created_at
		FROM members
		ORDER BY full_name LIMIT 100`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Member{}
	for rows.Next() {
		var m Member
		if err := rows.Scan(&m.ID, &m.BranchID, &m.FirstName, &m.LastName, &m.FullName,
			&m.Nickname, &m.Email, &m.Phone, &m.Whatsapp, &m.BirthDate, &m.Gender,
			&m.MaritalStatus, &m.MembershipStatus, &m.Profession, &m.Office, &m.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (r *Repo) Get(ctx context.Context, tx pgx.Tx, id string) (*Member, error) {
	var m Member
	err := tx.QueryRow(ctx, `
		SELECT id::text, branch_id::text, first_name, last_name, full_name,
		       nickname, email, phone, whatsapp, birth_date::text, gender,
		       marital_status, membership_status, profession, office, created_at
		FROM members WHERE id = $1`, id).
		Scan(&m.ID, &m.BranchID, &m.FirstName, &m.LastName, &m.FullName,
			&m.Nickname, &m.Email, &m.Phone, &m.Whatsapp, &m.BirthDate, &m.Gender,
			&m.MaritalStatus, &m.MembershipStatus, &m.Profession, &m.Office, &m.CreatedAt)
	return &m, err
}

type UpdateInput struct {
	Nickname         *string `json:"nickname"`
	Email            *string `json:"email"`
	Phone            *string `json:"phone"`
	Whatsapp         *string `json:"whatsapp"`
	BirthDate        *string `json:"birth_date"`
	Gender           *string `json:"gender"`
	MaritalStatus    *string `json:"marital_status"`
	Profession       *string `json:"profession"`
	Office           *string `json:"office"`
	MembershipStatus *string `json:"membership_status"`
}

// Update edita campos do perfil do membro.
func (r *Repo) Update(ctx context.Context, tx pgx.Tx, id string, in UpdateInput) (*Member, error) {
	var m Member
	err := tx.QueryRow(ctx, `
		UPDATE members SET
			nickname = COALESCE($2, nickname),
			email = COALESCE($3, email),
			phone = COALESCE($4, phone),
			whatsapp = COALESCE($5, whatsapp),
			birth_date = COALESCE($6::date, birth_date),
			gender = COALESCE($7, gender),
			marital_status = COALESCE($8, marital_status),
			profession = COALESCE($9, profession),
			office = COALESCE($10, office),
			membership_status = COALESCE($11, membership_status),
			updated_at = now()
		WHERE id = $1::uuid
		RETURNING id::text, branch_id::text, first_name, last_name, full_name,
		          nickname, email, phone, whatsapp, birth_date::text, gender,
		          marital_status, membership_status, profession, office, created_at`,
		id, in.Nickname, in.Email, in.Phone, in.Whatsapp, in.BirthDate, in.Gender,
		in.MaritalStatus, in.Profession, in.Office, in.MembershipStatus).
		Scan(&m.ID, &m.BranchID, &m.FirstName, &m.LastName, &m.FullName,
			&m.Nickname, &m.Email, &m.Phone, &m.Whatsapp, &m.BirthDate, &m.Gender,
			&m.MaritalStatus, &m.MembershipStatus, &m.Profession, &m.Office, &m.CreatedAt)
	return &m, err
}

type CreateInput struct {
	FirstName        string  `json:"first_name"`
	LastName         string  `json:"last_name"`
	Nickname         *string `json:"nickname"`
	Email            *string `json:"email"`
	Phone            *string `json:"phone"`
	Whatsapp         *string `json:"whatsapp"`
	BirthDate        *string `json:"birth_date"`
	Gender           *string `json:"gender"`
	MaritalStatus    *string `json:"marital_status"`
	Profession       *string `json:"profession"`
	Office           *string `json:"office"`
	MembershipStatus string  `json:"membership_status"`
}

// Create insere um membro. O branch_id vem da sessão RLS (não do client).
func (r *Repo) Create(ctx context.Context, tx pgx.Tx, tenantID, branchID string, in CreateInput) (*Member, error) {
	status := in.MembershipStatus
	if status == "" {
		status = "member"
	}
	var m Member
	full := in.FirstName + " " + in.LastName
	err := tx.QueryRow(ctx, `
		INSERT INTO members
			(tenant_id, branch_id, first_name, last_name, full_name,
			 nickname, email, phone, whatsapp, birth_date, gender,
			 marital_status, profession, office, membership_status)
		VALUES
			($1, $2, $3, $4, $5, $6, $7, $8, $9, $10::date, $11,
			 $12, $13, $14, $15)
		RETURNING id::text, branch_id::text, first_name, last_name, full_name,
		          nickname, email, phone, whatsapp, birth_date::text, gender,
		          marital_status, membership_status, profession, office, created_at`,
		tenantID, branchID, in.FirstName, in.LastName, full,
		in.Nickname, in.Email, in.Phone, in.Whatsapp, in.BirthDate, in.Gender,
		in.MaritalStatus, in.Profession, in.Office, status).
		Scan(&m.ID, &m.BranchID, &m.FirstName, &m.LastName, &m.FullName,
			&m.Nickname, &m.Email, &m.Phone, &m.Whatsapp, &m.BirthDate, &m.Gender,
			&m.MaritalStatus, &m.MembershipStatus, &m.Profession, &m.Office, &m.CreatedAt)
	return &m, err
}
