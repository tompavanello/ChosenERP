package families

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5"
)

type Family struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Code        *string         `json:"code,omitempty"`
	HeadID      *string         `json:"head_id,omitempty"`
	HeadName    *string         `json:"head_name,omitempty"`
	BranchID    string          `json:"branch_id"`
	Address     json.RawMessage `json:"address,omitempty"`
	MemberCount int64           `json:"member_count"`
}

type Repo struct{}

// familyCols é a projeção única de família: nome do chefe e contagem de
// membros vêm por subquery para o select não multiplicar linhas.
const familyCols = `f.id::text, f.name, f.code, f.head_id::text, f.branch_id::text, f.address,
	(SELECT m.full_name FROM members m WHERE m.id = f.head_id) AS head_name,
	(SELECT COUNT(DISTINCT x) FROM (
	     SELECT member_id AS x FROM member_relationships WHERE family_id = f.id
	     UNION SELECT related_id FROM member_relationships WHERE family_id = f.id
	 ) c) AS member_count`

func scanFamily(row pgx.Row) (*Family, error) {
	var f Family
	if err := row.Scan(&f.ID, &f.Name, &f.Code, &f.HeadID, &f.BranchID, &f.Address,
		&f.HeadName, &f.MemberCount); err != nil {
		return nil, err
	}
	return &f, nil
}

// List retorna famílias do escopo com contagem de membros vinculados.
func (r *Repo) List(ctx context.Context, tx pgx.Tx) ([]Family, error) {
	rows, err := tx.Query(ctx, `SELECT `+familyCols+` FROM families f ORDER BY f.name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Family{}
	for rows.Next() {
		f, err := scanFamily(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *f)
	}
	return out, rows.Err()
}

func (r *Repo) Get(ctx context.Context, tx pgx.Tx, id string) (*Family, error) {
	return scanFamily(tx.QueryRow(ctx, `SELECT `+familyCols+` FROM families f WHERE f.id = $1::uuid`, id))
}

// ListByMember devolve as famílias das quais o membro participa — seja como
// membro_id (origem do vínculo), related_id (destino) ou chefe da família.
// O chefe entra porque uma família pode existir antes do primeiro vínculo.
func (r *Repo) ListByMember(ctx context.Context, tx pgx.Tx, memberID string) ([]Family, error) {
	rows, err := tx.Query(ctx, `
		SELECT `+familyCols+`
		FROM families f
		WHERE f.head_id = $1::uuid
		   OR EXISTS (
			SELECT 1 FROM member_relationships r
			WHERE r.family_id = f.id AND (r.member_id = $1::uuid OR r.related_id = $1::uuid)
		   )
		ORDER BY f.name`, memberID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Family{}
	for rows.Next() {
		f, err := scanFamily(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *f)
	}
	return out, rows.Err()
}

// Create cria uma família com o próximo código livre do tenant.
//
// O código é calculado dentro da própria transação (MAX+1). Duas criações
// simultâneas colidiriam no índice uq_families_tenant_code e a segunda falha
// com erro claro — preferível a dois "#007" na mesma igreja.
func (r *Repo) Create(ctx context.Context, tx pgx.Tx, tenantID, branchID, name string) (*Family, error) {
	var id string
	err := tx.QueryRow(ctx, `
		INSERT INTO families (tenant_id, branch_id, name, code)
		VALUES ($1, $2::uuid, $3,
		        lpad(((SELECT COALESCE(MAX(NULLIF(code, '')::int), 0) FROM families WHERE tenant_id = $1) + 1)::text, 3, '0'))
		RETURNING id::text`, tenantID, branchID, name).Scan(&id)
	if err != nil {
		return nil, err
	}
	return r.Get(ctx, tx, id)
}

type UpdateInput struct {
	Name    *string         `json:"name"`
	HeadID  *string         `json:"head_id"`
	Address json.RawMessage `json:"address"`
}

// Update renomeia, troca o chefe ou grava o endereço (jsonb).
// PATCH: campo ausente (nil) mantém o valor atual.
func (r *Repo) Update(ctx context.Context, tx pgx.Tx, id string, in UpdateInput) (*Family, error) {
	var updatedID string
	err := tx.QueryRow(ctx, `
		UPDATE families f SET
			name    = COALESCE($2, f.name),
			head_id = COALESCE(NULLIF($3,'')::uuid, f.head_id),
			address = COALESCE($4::jsonb, f.address)
		WHERE f.id = $1::uuid
		RETURNING f.id::text`,
		id, in.Name, in.HeadID, jsonOrNil(in.Address)).Scan(&updatedID)
	if err != nil {
		return nil, err
	}
	return r.Get(ctx, tx, updatedID)
}

// Delete remove a família. member_relationships.family_id é ON DELETE SET NULL,
// então os vínculos de parentesco sobrevivem — o que se perde é só o
// agrupamento, nunca a relação familiar em si.
func (r *Repo) Delete(ctx context.Context, tx pgx.Tx, id string) error {
	tag, err := tx.Exec(ctx, `DELETE FROM families WHERE id = $1::uuid`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
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

// UnlinkMember desfaz APENAS o agrupamento desta família: as colunas de
// parentesco do vínculo continuam existindo, mas deixam de pertencer a esta
// família. Apagar a linha seria destrutivo — o mesmo registro pode ser o
// vínculo "Cônjuge" fora do contexto familiar.
func (r *Repo) UnlinkMember(ctx context.Context, tx pgx.Tx, familyID, memberID string) error {
	_, err := tx.Exec(ctx, `
		UPDATE member_relationships SET family_id = NULL
		WHERE family_id = $1::uuid AND (member_id = $2::uuid OR related_id = $2::uuid)`,
		familyID, memberID)
	if err != nil {
		return err
	}
	// Quem saiu da família não pode continuar sendo o chefe dela.
	_, err = tx.Exec(ctx,
		`UPDATE families SET head_id = NULL WHERE id = $1::uuid AND head_id = $2::uuid`,
		familyID, memberID)
	return err
}

// jsonOrNil devolve nil para JSON vazio, para o COALESCE do UPDATE manter o
// endereço anterior em vez de zerá-lo.
func jsonOrNil(raw json.RawMessage) any {
	if len(raw) == 0 {
		return nil
	}
	return string(raw)
}
