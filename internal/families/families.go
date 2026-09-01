package families

import (
	"context"

	"github.com/jackc/pgx/v5"
)

type Family struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	HeadID      *string `json:"head_id,omitempty"`
	BranchID    string  `json:"branch_id"`
	MemberCount int64   `json:"member_count"`
}

type Repo struct{}

// List retorna famílias do escopo com contagem de membros vinculados.
func (r *Repo) List(ctx context.Context, tx pgx.Tx) ([]Family, error) {
	rows, err := tx.Query(ctx, `
		SELECT f.id::text, f.name, COALESCE(f.head_id::text,''), f.branch_id::text,
		       (SELECT COUNT(DISTINCT x) FROM (SELECT member_id AS x FROM member_relationships WHERE family_id=f.id
		        UNION SELECT related_id FROM member_relationships WHERE family_id=f.id) c)
		FROM families f ORDER BY f.name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Family{}
	for rows.Next() {
		var f Family
		var head string
		if err := rows.Scan(&f.ID, &f.Name, &head, &f.BranchID, &f.MemberCount); err != nil {
			return nil, err
		}
		if head != "" {
			f.HeadID = &head
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

// Create cria uma família.
func (r *Repo) Create(ctx context.Context, tx pgx.Tx, tenantID, branchID, name string) (*Family, error) {
	var f Family
	err := tx.QueryRow(ctx, `
		INSERT INTO families (tenant_id, branch_id, name)
		VALUES ($1, $2::uuid, $3)
		RETURNING id::text, name, COALESCE(head_id::text,''), branch_id::text`,
		tenantID, branchID, name).
		Scan(&f.ID, &f.Name, &f.HeadID, &f.BranchID)
	return &f, err
}

// LinkMember associa um membro à família criando um vínculo real com o anchor
// (relateID ou o head da família). Se não houver anchor, o membro vira o head.
func (r *Repo) LinkMember(ctx context.Context, tx pgx.Tx, familyID, memberID, relateID, relation string) error {
	if relation == "" {
		relation = "relative"
	}
	anchor := relateID
	if anchor == "" {
		if err := tx.QueryRow(ctx, `SELECT COALESCE(head_id::text,'') FROM families WHERE id=$1::uuid`, familyID).Scan(&anchor); err != nil {
			return err
		}
	}
	if anchor == "" || anchor == memberID {
		// Sem anchor: define este membro como head da família (autovínculo descartado).
		_, err := tx.Exec(ctx, `UPDATE families SET head_id=$1::uuid WHERE id=$2::uuid AND head_id IS NULL`, memberID, familyID)
		return err
	}
	_, err := tx.Exec(ctx, `
		INSERT INTO member_relationships (member_id, related_id, family_id, kind)
		VALUES ($1::uuid, $2::uuid, $3::uuid, $4)
		ON CONFLICT (member_id, related_id, kind) DO NOTHING`,
		memberID, anchor, familyID, relation)
	return err
}
