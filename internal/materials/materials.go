package materials

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
)

// Material e um material de estudo (arquivo enviado ou link) disponivel para a
// igreja/filial ou para um grupo/celula especifico.
type Material struct {
	ID          string    `json:"id"`
	BranchID    string    `json:"branch_id"`
	GroupID     *string   `json:"group_id,omitempty"`
	GroupName   *string   `json:"group_name,omitempty"`
	MinistryID  *string   `json:"ministry_id,omitempty"`
	Title       string    `json:"title"`
	Description *string   `json:"description,omitempty"`
	Kind        string    `json:"kind"`
	URL         *string   `json:"url,omitempty"`
	FileName    *string   `json:"file_name,omitempty"`
	FileSize    *int64    `json:"file_size,omitempty"`
	MimeType    *string   `json:"mime_type,omitempty"`
	IsPublished bool      `json:"is_published"`
	CreatedAt   time.Time `json:"created_at"`
	// DiskName e o nome opaco do arquivo no disco. NUNCA vai para o JSON: e
	// interno (usado pelo download e pela remocao).
	DiskName *string `json:"-"`
}

// CreateInput sao os dados de um material novo (arquivo ou link).
type CreateInput struct {
	Title       string
	Description *string
	GroupID     *string
	MinistryID  *string
	Kind        string
	URL         *string
	DiskName    *string
	FileName    *string
	FileSize    *int64
	MimeType    *string
	IsPublished bool
}

type Repo struct{}

const selectCols = `
	m.id::text, m.branch_id::text, m.group_id::text, g.name, m.ministry_id::text,
	m.title, m.description, m.kind, m.url, m.file_name, m.file_size, m.mime_type,
	m.is_published, m.created_at, m.disk_name`

func scan(row pgx.Row) (*Material, error) {
	var m Material
	err := row.Scan(&m.ID, &m.BranchID, &m.GroupID, &m.GroupName, &m.MinistryID,
		&m.Title, &m.Description, &m.Kind, &m.URL, &m.FileName, &m.FileSize,
		&m.MimeType, &m.IsPublished, &m.CreatedAt, &m.DiskName)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

// List retorna os materiais do escopo. Quando groupID e informado, filtra por
// grupo.
func (r *Repo) List(ctx context.Context, tx pgx.Tx, groupID string) ([]Material, error) {
	rows, err := tx.Query(ctx, `
		SELECT `+selectCols+`
		FROM study_materials m
		LEFT JOIN small_groups g ON g.id = m.group_id
		WHERE ($1 = '' OR m.group_id = NULLIF($1,'')::uuid)
		ORDER BY m.created_at DESC`, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Material{}
	for rows.Next() {
		var m Material
		if err := rows.Scan(&m.ID, &m.BranchID, &m.GroupID, &m.GroupName, &m.MinistryID,
			&m.Title, &m.Description, &m.Kind, &m.URL, &m.FileName, &m.FileSize,
			&m.MimeType, &m.IsPublished, &m.CreatedAt, &m.DiskName); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// ListForMember retorna os materiais publicados gerais (sem grupo) e os dos
// grupos dos quais o membro participa.
func (r *Repo) ListForMember(ctx context.Context, tx pgx.Tx, memberID string) ([]Material, error) {
	rows, err := tx.Query(ctx, `
		SELECT `+selectCols+`
		FROM study_materials m
		LEFT JOIN small_groups g ON g.id = m.group_id
		WHERE m.is_published
		  AND (m.group_id IS NULL OR m.group_id IN (
		        SELECT gm.group_id FROM group_members gm WHERE gm.member_id = $1::uuid))
		ORDER BY m.created_at DESC`, memberID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Material{}
	for rows.Next() {
		var m Material
		if err := rows.Scan(&m.ID, &m.BranchID, &m.GroupID, &m.GroupName, &m.MinistryID,
			&m.Title, &m.Description, &m.Kind, &m.URL, &m.FileName, &m.FileSize,
			&m.MimeType, &m.IsPublished, &m.CreatedAt, &m.DiskName); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// Get retorna um material pelo id (inclui o nome do arquivo no disco).
func (r *Repo) Get(ctx context.Context, tx pgx.Tx, id string) (*Material, error) {
	return scan(tx.QueryRow(ctx, `
		SELECT `+selectCols+`
		FROM study_materials m
		LEFT JOIN small_groups g ON g.id = m.group_id
		WHERE m.id = $1::uuid`, id))
}

// Create insere um material no escopo RLS da sessao.
func (r *Repo) Create(ctx context.Context, tx pgx.Tx, tenantID, branchID, userID string, in CreateInput) (*Material, error) {
	return scan(tx.QueryRow(ctx, `
		INSERT INTO study_materials
			(tenant_id, branch_id, group_id, ministry_id, title, description, kind, url,
			 disk_name, file_name, file_size, mime_type, is_published, created_by)
		VALUES
			($1, $2::uuid, NULLIF($3,'')::uuid, NULLIF($4,'')::uuid, $5, $6, $7, $8,
			 $9, $10, $11, $12, $13, NULLIF($14,'')::uuid)
		RETURNING `+selectColsForReturning,
		tenantID, branchID, str(in.GroupID), str(in.MinistryID), in.Title, in.Description,
		in.Kind, in.URL, in.DiskName, in.FileName, in.FileSize, in.MimeType, in.IsPublished, userID))
}

// selectColsForReturning e a mesma lista, mas as colunas vem de RETURNING (sem o
// prefixo de tabela m/g e sem JOIN) - o INSERT nao tem os aliases.
const selectColsForReturning = `
	id::text, branch_id::text, group_id::text, NULL::text, ministry_id::text,
	title, description, kind, url, file_name, file_size, mime_type,
	is_published, created_at, disk_name`

// Delete remove um material pelo id.
func (r *Repo) Delete(ctx context.Context, tx pgx.Tx, id string) error {
	tag, err := tx.Exec(ctx, `DELETE FROM study_materials WHERE id = $1::uuid`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func str(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
