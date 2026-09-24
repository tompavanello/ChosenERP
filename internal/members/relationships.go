package members

import (
	"context"

	"github.com/jackc/pgx/v5"
)

// Relationship e um vinculo entre dois membros (familia ou discipulado).
type Relationship struct {
	ID        string `json:"id"`
	OtherID   string `json:"related_id"`
	OtherName string `json:"related_name"`
	Kind      string `json:"kind"`
	Relation  string `json:"relation"`
}

// FamilyMember e um membro pertencente a uma familia.
//
// A projecao e relativa ao CHEFE da familia (nao a quem consulta): "Relation"
// descreve o parentesco do membro com o chefe. E o que a planilha do cliente
// mostra - o cabeca da familia e as pessoas listadas abaixo dele.
type FamilyMember struct {
	ID               string  `json:"id"`
	FullName         string  `json:"full_name"`
	Relation         string  `json:"relation"`       // ex.: Conjuge, Filho(a), Parente
	Kind             string  `json:"kind,omitempty"` // chave crua do vinculo
	IsHead           bool    `json:"is_head"`
	MembershipStatus string  `json:"membership_status"`
	Phone            *string `json:"phone,omitempty"`
	Whatsapp         *string `json:"whatsapp,omitempty"`
	Email            *string `json:"email,omitempty"`
	BirthDate        *string `json:"birth_date,omitempty"`
	PhotoURL         *string `json:"photo_url,omitempty"`
}

const relQuery = `
SELECT DISTINCT ON (other_id) id::text, other_id, other_name, kind
FROM (
    -- vinculo direto: o kind ja e a relacao do OUTRO com este membro
    SELECT r.id, r.related_id::text AS other_id, m.full_name AS other_name, r.kind, 0 AS pref
    FROM member_relationships r JOIN members m ON m.id = r.related_id
    WHERE r.member_id = $1 AND r.member_id <> r.related_id
    UNION ALL
    -- vinculo reverso: inverte o kind para obter a relacao do outro com este membro
    SELECT r.id, r.member_id::text AS other_id, m.full_name AS other_name,
           CASE r.kind
               WHEN 'parent' THEN 'child'
               WHEN 'child' THEN 'parent'
               WHEN 'discipler' THEN 'disciple'
               WHEN 'disciple' THEN 'discipler'
               ELSE r.kind
           END AS kind,
           1 AS pref
    FROM member_relationships r JOIN members m ON m.id = r.member_id
    WHERE r.related_id = $1 AND r.member_id <> r.related_id
) t
ORDER BY other_id, pref`

func relationLabel(kind string, isFrom bool) string {
	switch kind {
	case "spouse":
		return "Conjuge"
	case "parent":
		if isFrom {
			return "Pai/Mae"
		}
		return "Filho(a)"
	case "child":
		if isFrom {
			return "Filho(a)"
		}
		return "Pai/Mae"
	case "disciple":
		if isFrom {
			return "Discipulo(a)"
		}
		return "Discipulador(a)"
	case "discipler":
		if isFrom {
			return "Discipulador(a)"
		}
		return "Discipulo(a)"
	case "dependent":
		return "Dependente"
	default:
		return "Parente"
	}
}

// GetTree retorna o membro e seus vinculos (familia + discipulado).
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
		if err := rows.Scan(&rid, &oid, &oname, &kind); err != nil {
			return nil, nil, err
		}
		rels = append(rels, Relationship{ID: rid, OtherID: oid, OtherName: oname, Kind: kind, Relation: relationLabel(kind, true)})
	}
	return m, rels, rows.Err()
}

// inverseKind devolve o vinculo reciproco: se o OUTRO e meu "parent"
// (pai/mae), entao eu sou o "child" dele, e assim por diante. Pares simetricos
// (spouse, dependent, relative) permanecem iguais.
func inverseKind(kind string) string {
	switch kind {
	case "parent":
		return "child"
	case "child":
		return "parent"
	case "discipler":
		return "disciple"
	case "disciple":
		return "discipler"
	default:
		return kind
	}
}

// AddRelationship cria o vinculo nos DOIS sentidos: member_id -> related_id com
// `kind`, e related_id -> member_id com o vinculo inverso. Assim a referencia
// fica correta tambem na ficha do outro membro (o destino). Idempotente.
func (r *Repo) AddRelationship(ctx context.Context, tx pgx.Tx, memberID, relatedID, kind string) error {
	if memberID == relatedID {
		return nil
	}
	if kind == "" {
		kind = "relative"
	}
	_, err := tx.Exec(ctx, `
		INSERT INTO member_relationships (member_id, related_id, kind)
		VALUES ($1::uuid, $2::uuid, $3), ($2::uuid, $1::uuid, $4)
		ON CONFLICT (member_id, related_id, kind) DO NOTHING`,
		memberID, relatedID, kind, inverseKind(kind))
	return err
}

// FamilyMembers retorna os membros de uma familia, o chefe primeiro.
//
// A versao anterior fazia `JOIN members m ON m.id IN (r.member_id, r.related_id)`,
// que devolvia as DUAS pontas de cada vinculo e rotulava ambas com o mesmo
// `r.kind` - ou seja, o pai aparecia como "Filho(a)" e o conjuge do conjuge
// entrava na lista. Agora cada pessoa e resolvida uma unica vez e o parentesco
// e o dela com o chefe da familia.
func (r *Repo) FamilyMembers(ctx context.Context, tx pgx.Tx, familyID string) ([]FamilyMember, error) {
	rows, err := tx.Query(ctx, `
		WITH fam AS (
			SELECT id, head_id FROM families WHERE id = $1::uuid
		), pessoas AS (
			SELECT head_id AS id FROM fam WHERE head_id IS NOT NULL
			UNION
			SELECT r.member_id   FROM member_relationships r JOIN fam ON r.family_id = fam.id
			UNION
			SELECT r.related_id  FROM member_relationships r JOIN fam ON r.family_id = fam.id
		)
		SELECT m.id::text, m.full_name, m.membership_status, m.phone, m.whatsapp,
		       m.email, m.birth_date::text, m.photo_url,
		       (m.id = fam.head_id) AS is_head,
		       COALESCE(cr.kind, ''), COALESCE(cr.member_id = fam.head_id, false)
		FROM pessoas p
		CROSS JOIN fam
		JOIN members m ON m.id = p.id
		LEFT JOIN LATERAL (
			SELECT r.kind, r.member_id
			FROM member_relationships r
			WHERE r.family_id = fam.id
			  AND ((r.member_id = fam.head_id AND r.related_id = m.id)
			    OR (r.related_id = fam.head_id AND r.member_id = m.id))
			LIMIT 1
		) cr ON true
		ORDER BY is_head DESC, m.full_name`, familyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []FamilyMember{}
	for rows.Next() {
		var fm FamilyMember
		var kind string
		var fromHead bool
		if err := rows.Scan(&fm.ID, &fm.FullName, &fm.MembershipStatus, &fm.Phone,
			&fm.Whatsapp, &fm.Email, &fm.BirthDate, &fm.PhotoURL,
			&fm.IsHead, &kind, &fromHead); err != nil {
			return nil, err
		}
		fm.Kind = kind
		if fm.IsHead {
			fm.Relation = "Chefe da familia"
		} else if kind == "" {
			fm.Relation = "Parente"
		} else {
			fm.Relation = relationLabel(kind, fromHead)
		}
		out = append(out, fm)
	}
	return out, rows.Err()
}
