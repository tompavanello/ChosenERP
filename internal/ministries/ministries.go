package ministries

import (
	"context"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

// Ministry e um ministerio/area de voluntariado da igreja.
type Ministry struct {
	ID          string    `json:"id"`
	BranchID    string    `json:"branch_id"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	Description *string   `json:"description,omitempty"`
	LeaderID    *string   `json:"leader_id,omitempty"`
	LeaderName  *string   `json:"leader_name,omitempty"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
}

// MinistryMember e um voluntario de um ministerio.
type MinistryMember struct {
	MinistryID string  `json:"ministry_id"`
	MemberID   string  `json:"member_id"`
	MemberName string  `json:"member_name"`
	Role       string  `json:"role"`
	StartedAt  *string `json:"started_at,omitempty"`
}

type CreateInput struct {
	Name        string  `json:"name"`
	Description *string `json:"description"`
	LeaderID    *string `json:"leader_id"`
}

type UpdateInput struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	LeaderID    *string `json:"leader_id"`
	IsActive    *bool   `json:"is_active"`
}

type Repo struct{}

// List retorna os ministerios do escopo.
func (r *Repo) List(ctx context.Context, tx pgx.Tx) ([]Ministry, error) {
	rows, err := tx.Query(ctx, `
		SELECT m.id::text, m.branch_id::text, m.name, m.slug, m.description,
		       m.leader_id::text, lm.full_name, m.is_active, m.created_at
		FROM ministries m
		LEFT JOIN members lm ON lm.id = m.leader_id
		ORDER BY m.name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Ministry{}
	for rows.Next() {
		var m Ministry
		// LEFT JOIN: lm.full_name e NULL quando o ministerio nao tem lider.
		var leader *string
		if err := rows.Scan(&m.ID, &m.BranchID, &m.Name, &m.Slug, &m.Description,
			&m.LeaderID, &leader, &m.IsActive, &m.CreatedAt); err != nil {
			return nil, err
		}
		if leader != nil && *leader != "" {
			m.LeaderName = leader
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// Create insere um ministerio no escopo RLS da sessao.
func (r *Repo) Create(ctx context.Context, tx pgx.Tx, tenantID, branchID string, in CreateInput) (*Ministry, error) {
	if in.Name == "" {
		return nil, pgx.ErrNoRows
	}
	slug := slugify(in.Name)
	var m Ministry
	err := tx.QueryRow(ctx, `
		INSERT INTO ministries (tenant_id, branch_id, name, slug, description, leader_id)
		VALUES ($1, $2::uuid, $3, $4, $5, NULLIF($6,'')::uuid)
		RETURNING id::text, branch_id::text, name, slug, description, leader_id::text, is_active, created_at`,
		tenantID, branchID, in.Name, slug, in.Description, nullStr(in.LeaderID)).
		Scan(&m.ID, &m.BranchID, &m.Name, &m.Slug, &m.Description, &m.LeaderID, &m.IsActive, &m.CreatedAt)
	return &m, err
}

// Update edita nome/descricao/responsavel/situacao do ministerio.
func (r *Repo) Update(ctx context.Context, tx pgx.Tx, id string, in UpdateInput) (*Ministry, error) {
	var m Ministry
	err := tx.QueryRow(ctx, `
		UPDATE ministries SET
			name = COALESCE(NULLIF($2,''), name),
			description = COALESCE($3, description),
			leader_id = CASE WHEN $4::boolean THEN NULLIF($5,'')::uuid ELSE leader_id END,
			is_active = COALESCE($6::boolean, is_active)
		WHERE id = $1::uuid
		RETURNING id::text, branch_id::text, name, slug, description, leader_id::text, is_active, created_at`,
		id, str(in.Name), in.Description, in.LeaderID != nil, str(in.LeaderID), in.IsActive).
		Scan(&m.ID, &m.BranchID, &m.Name, &m.Slug, &m.Description, &m.LeaderID, &m.IsActive, &m.CreatedAt)
	return &m, err
}

// Delete remove um ministerio (vinculos e referencia de grupo caem por FK).
func (r *Repo) Delete(ctx context.Context, tx pgx.Tx, id string) error {
	tag, err := tx.Exec(ctx, `DELETE FROM ministries WHERE id = $1::uuid`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// ListMembers retorna os voluntarios de um ministerio.
func (r *Repo) ListMembers(ctx context.Context, tx pgx.Tx, ministryID string) ([]MinistryMember, error) {
	rows, err := tx.Query(ctx, `
		SELECT mm.ministry_id::text, mm.member_id::text, mb.full_name,
		       mm.role, mm.started_at::text
		FROM ministry_members mm
		JOIN members mb ON mb.id = mm.member_id
		WHERE mm.ministry_id = $1::uuid
		ORDER BY mb.full_name`, ministryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []MinistryMember{}
	for rows.Next() {
		var m MinistryMember
		if err := rows.Scan(&m.MinistryID, &m.MemberID, &m.MemberName, &m.Role, &m.StartedAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// AddMember vincula um voluntario a um ministerio.
func (r *Repo) AddMember(ctx context.Context, tx pgx.Tx, tenantID, ministryID, memberID, role string) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO ministry_members (ministry_id, member_id, role)
		VALUES ($1::uuid, $2::uuid, $3)`, ministryID, memberID, role)
	return err
}

// RemoveMember remove um voluntario de um ministerio.
func (r *Repo) RemoveMember(ctx context.Context, tx pgx.Tx, ministryID, memberID string) error {
	_, err := tx.Exec(ctx, `
		DELETE FROM ministry_members
		WHERE ministry_id = $1::uuid AND member_id = $2::uuid`, ministryID, memberID)
	return err
}

func slugify(s string) string {
	out := make([]rune, 0, len(s))
	lastDash := false
	for _, r := range strings.ToLower(s) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			out = append(out, r)
			lastDash = false
		default:
			if len(out) > 0 && !lastDash {
				out = append(out, '-')
				lastDash = true
			}
		}
	}
	for len(out) > 0 && out[len(out)-1] == '-' {
		out = out[:len(out)-1]
	}
	if len(out) == 0 {
		return "grupo"
	}
	return string(out)
}

func nullStr(s *string) *string {
	if s != nil && *s == "" {
		return nil
	}
	return s
}

func str(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
