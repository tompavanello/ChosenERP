package members

import (
	"context"

	"github.com/jackc/pgx/v5"
)

// Relationship é um vínculo entre dois membros (família ou discipulado).
type Relationship struct {
	ID        string `json:"id"`
	OtherID   string `json:"related_id"`
	OtherName string `json:"related_name"`
	Kind      string `json:"kind"`
	Relation  string `json:"relation"`
}

// FamilyMember é um membro pertencente a uma família.
type FamilyMember struct {
	ID       string `json:"id"`
	FullName string `json:"full_name"`
	Relation string `json:"relation"`
}

const relQuery = `
SELECT id::text, other_id, other_name, kind, is_from
FROM (
    SELECT r.id, r.related_id::text AS other_id, m.full_name AS other_name, r.kind, true AS is_from
    FROM member_relationships r JOIN members m ON m.id = r.related_id
    WHERE r.member_id = $1 AND r.member_id <> r.related_id
    UNION ALL
    SELECT r.id, r.member_id::text AS other_id, m.full_name AS other_name, r.kind, false AS is_from
    FROM member_relationships r JOIN members m ON m.id = r.member_id
    WHERE r.related_id = $1 AND r.member_id <> r.related_id
) t`

func relationLabel(kind string, isFrom bool) string {
	switch kind {
	case "spouse":
		return "Cônjuge"
	case "parent":
		if isFrom {
			return "Pai/Mãe"
		}
		return "Filho(a)"
	case "child":
		if isFrom {
			return "Filho(a)"
		}
		return "Pai/Mãe"
	case "disciple":
		if isFrom {
			return "Discípulo(a)"
		}
		return "Discipulador(a)"
	case "discipler":
		if isFrom {
			return "Discipulador(a)"
		}
		return "Discípulo(a)"
	case "dependent":
		return "Dependente"
	default:
		return "Parente"
	}
}

// GetTree retorna o membro e seus vínculos (família + discipulado).
func (r *Repo) GetTree(ctx context.Context, tx pgx.Tx, id string) (*Member, []Relationship, error) {
	m, err := r.Get(ctx, tx, id)
	if err != nil {
		return nil, nil, err
	}
	rows, err := tx.Query(ctx, relQuery, id)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	rels := []Relationship{}
	for rows.Next() {
		var rid, oid, oname, kind string
		var isFrom bool
		if err := rows.Scan(&rid, &oid, &oname, &kind, &isFrom); err != nil {
			return nil, nil, err
		}
		rels = append(rels, Relationship{ID: rid, OtherID: oid, OtherName: oname, Kind: kind, Relation: relationLabel(kind, isFrom)})
	}
	return m, rels, rows.Err()
}

// AddRelationship cria um vínculo member_id -> related_id (com reversão automática p/ spouse).
func (r *Repo) AddRelationship(ctx context.Context, tx pgx.Tx, memberID, relatedID, kind string) error {
	if kind == "spouse" {
		_, err := tx.Exec(ctx, `
			INSERT INTO member_relationships (member_id, related_id, kind)
			VALUES ($1::uuid, $2::uuid, 'spouse'), ($2::uuid, $1::uuid, 'spouse')
			ON CONFLICT (member_id, related_id, kind) DO NOTHING`, memberID, relatedID)
		return err
	}
	_, err := tx.Exec(ctx, `
		INSERT INTO member_relationships (member_id, related_id, kind)
		VALUES ($1::uuid, $2::uuid, $3)
		ON CONFLICT (member_id, related_id, kind) DO NOTHING`, memberID, relatedID, kind)
	return err
}

// FamilyMembers retorna os membros vinculados a uma família.
func (r *Repo) FamilyMembers(ctx context.Context, tx pgx.Tx, familyID string) ([]FamilyMember, error) {
	rows, err := tx.Query(ctx, `
		SELECT DISTINCT m.id::text, m.full_name, r.kind
		FROM member_relationships r
		JOIN members m ON m.id IN (r.member_id, r.related_id)
		WHERE r.family_id = $1::uuid`, familyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []FamilyMember{}
	for rows.Next() {
		var fm FamilyMember
		if err := rows.Scan(&fm.ID, &fm.FullName, &fm.Relation); err != nil {
			return nil, err
		}
		out = append(out, fm)
	}
	return out, rows.Err()
}
