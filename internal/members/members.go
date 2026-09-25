package members

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

// Address e o endereco do MEMBRO (requisito 1.1). Guardado como jsonb para
// acomodar variacoes sem migracao a cada campo novo.
type Address struct {
	ZipCode    string `json:"zip_code,omitempty"`
	Street     string `json:"street,omitempty"`
	Number     string `json:"number,omitempty"`
	Complement string `json:"complement,omitempty"`
	District   string `json:"district,omitempty"`
	City       string `json:"city,omitempty"`
	State      string `json:"state,omitempty"`
}

type Member struct {
	ID               string  `json:"id"`
	BranchID         string  `json:"branch_id"`
	FirstName        string  `json:"first_name"`
	LastName         string  `json:"last_name"`
	FullName         string  `json:"full_name"`
	Nickname         *string `json:"nickname,omitempty"`
	Email            *string `json:"email,omitempty"`
	Phone            *string `json:"phone,omitempty"`
	Whatsapp         *string `json:"whatsapp,omitempty"`
	BirthDate        *string `json:"birth_date,omitempty"`
	Gender           *string `json:"gender,omitempty"`
	MaritalStatus    *string `json:"marital_status,omitempty"`
	MembershipStatus string  `json:"membership_status"`
	// RollClass e DERIVADO da situacao (requisito 1.2): Ativo = professo; como
	// nao existe "professo inativo", a professorate acompanha membership_status.
	RollClass       string   `json:"roll_class"`
	Profession      *string  `json:"profession,omitempty"`
	Office          *string  `json:"office,omitempty"` // legado: ver member_cargos
	Nationality     *string  `json:"nationality,omitempty"`
	Education       *string  `json:"education,omitempty"`
	Notes           *string  `json:"notes,omitempty"`
	CPF             *string  `json:"cpf,omitempty"`
	RG              *string  `json:"rg,omitempty"`
	BaptismDate     *string  `json:"baptism_date,omitempty"`
	BaptismLocation *string  `json:"baptism_location,omitempty"`
	JoinedAt        *string  `json:"joined_at,omitempty"`
	Address         *Address `json:"address,omitempty"`
	ExitReason      *string  `json:"exit_reason,omitempty"`
	ExitedAt        *string  `json:"exited_at,omitempty"`
	MarriageDate    *string  `json:"marriage_date,omitempty"`
	PhotoURL        *string  `json:"photo_url,omitempty"`
	// Cargos ativos do membro (funcoes/ministerios, requisito 1.3). Vem de
	// member_cargos; pode ter mais de um.
	Cargos    []CargoRef `json:"cargos,omitempty"`
	CreatedAt time.Time  `json:"created_at"`

	// Numero da carteirinha ja emitida (documents.document_ref). Ausente quando
	// o membro ainda nao tem carteirinha. O qr_token NAO e exposto na listagem:
	// quem precisa dele usa GET /members/{id}/card.
	CardRef *string `json:"card_ref,omitempty"`
}

// CargoRef e o cargo ativo resumido para exibicao em listas.
type CargoRef struct {
	ID     string  `json:"id"`
	Name   string  `json:"name"`
	Kind   string  `json:"kind"`
	EndsAt *string `json:"ends_at,omitempty"`
}

// rollClassOf deriva a classificacao no Rol a partir da situacao.
func rollClassOf(status string) string {
	if status == "active" {
		return "professo"
	}
	return "nao_professo"
}

// decodeCargos converte o json_agg da projecao em CargoRef.
// A lista vem sempre presente (COALESCE com '[]'), entao um JSON invalido aqui
// e bug de SQL, nao dado ausente.
func decodeCargos(raw string, m *Member) error {
	if raw == "" {
		return nil
	}
	return json.Unmarshal([]byte(raw), &m.Cargos)
}

// HasCargoVencendo informa se algum cargo ativo tem mandato vencido ou vencendo
// nos proximos `dias`. Usado para sinalizar a situacao na UI.
func (m Member) HasCargoVencendo(hoje time.Time, dias int) bool {
	limite := hoje.AddDate(0, 0, dias)
	for _, c := range m.Cargos {
		if c.EndsAt == nil {
			continue
		}
		fim, err := time.Parse("2006-01-02", *c.EndsAt)
		if err != nil {
			continue
		}
		if !fim.After(limite) {
			return true
		}
	}
	return false
}

type Repo struct{}

// Projecao unica compartilhada por List/Get/Update/Create - evita que uma
// coluna nova entre em uma query e falte em outra.
const memberCols = `m.id::text, m.branch_id::text, m.first_name, m.last_name,
	m.full_name, m.nickname, m.email, m.phone, m.whatsapp, m.birth_date::text,
	m.gender, m.marital_status, m.membership_status, m.profession, m.office,
	m.nationality, m.education, m.notes,
	m.cpf, m.rg, m.baptism_date::text, m.baptism_location, m.joined_at::text,
	m.photo_url, m.created_at,
	m.address::text, m.exit_reason, m.exited_at::text, m.marriage_date::text,
	(SELECT d.document_ref FROM documents d
	  WHERE d.member_id = m.id AND d.kind = 'membership_card'
	  ORDER BY d.created_at DESC LIMIT 1) AS card_ref,
	COALESCE((
		SELECT json_agg(json_build_object(
		           'id', c.id::text, 'name', c.name, 'kind', c.kind,
		           'ends_at', mc.ends_at::text)
		       ORDER BY c.sort_order, c.name)
		FROM member_cargos mc
		JOIN cargos c ON c.id = mc.cargo_id
		WHERE mc.member_id = m.id AND mc.status = 'ativo'
	), '[]'::json)::text AS cargos`

func scanMember(row pgx.Row) (*Member, error) {
	var m Member
	var cargosJSON string
	var addressRaw *string
	err := row.Scan(&m.ID, &m.BranchID, &m.FirstName, &m.LastName, &m.FullName,
		&m.Nickname, &m.Email, &m.Phone, &m.Whatsapp, &m.BirthDate, &m.Gender,
		&m.MaritalStatus, &m.MembershipStatus, &m.Profession, &m.Office,
		&m.Nationality, &m.Education, &m.Notes,
		&m.CPF, &m.RG, &m.BaptismDate, &m.BaptismLocation, &m.JoinedAt,
		&m.PhotoURL, &m.CreatedAt, &addressRaw, &m.ExitReason, &m.ExitedAt,
		&m.MarriageDate, &m.CardRef, &cargosJSON)
	if err != nil {
		return nil, err
	}
	if addressRaw != nil && *addressRaw != "" {
		var a Address
		if err := json.Unmarshal([]byte(*addressRaw), &a); err == nil {
			m.Address = &a
		}
	}
	m.RollClass = rollClassOf(m.MembershipStatus)
	if err := decodeCargos(cargosJSON, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

// List retorna membros do escopo definido pela transacao (RLS).
// Se q for informado, filtra por full_name ILIKE (case-insensitive).
func (r *Repo) List(ctx context.Context, tx pgx.Tx, q string) ([]Member, error) {
	rows, err := tx.Query(ctx, `
		SELECT `+memberCols+`
		FROM members m
		WHERE ($1 = '' OR m.full_name ILIKE '%' || $1 || '%')
		ORDER BY m.full_name`, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Member{}
	for rows.Next() {
		m, err := scanMember(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *m)
	}
	return out, rows.Err()
}

func (r *Repo) Get(ctx context.Context, tx pgx.Tx, id string) (*Member, error) {
	return scanMember(tx.QueryRow(ctx, `
		SELECT `+memberCols+`
		FROM members m WHERE m.id = $1`, id))
}

type UpdateInput struct {
	FirstName        *string  `json:"first_name"`
	LastName         *string  `json:"last_name"`
	Nickname         *string  `json:"nickname"`
	Email            *string  `json:"email"`
	Phone            *string  `json:"phone"`
	Whatsapp         *string  `json:"whatsapp"`
	BirthDate        *string  `json:"birth_date"`
	Gender           *string  `json:"gender"`
	MaritalStatus    *string  `json:"marital_status"`
	Profession       *string  `json:"profession"`
	Office           *string  `json:"office"` // legado: mantido para nao quebrar clientes antigos (readJSON rejeita campo desconhecido)
	Nationality      *string  `json:"nationality"`
	Education        *string  `json:"education"`
	Notes            *string  `json:"notes"`
	MembershipStatus *string  `json:"membership_status"`
	CPF              *string  `json:"cpf"`
	RG               *string  `json:"rg"`
	BaptismDate      *string  `json:"baptism_date"`
	BaptismLocation  *string  `json:"baptism_location"`
	JoinedAt         *string  `json:"joined_at"`
	Address          *Address `json:"address"`
	ExitReason       *string  `json:"exit_reason"`
	ExitedAt         *string  `json:"exited_at"`
	MarriageDate     *string  `json:"marriage_date"`
	// photo_url NAO e aceito aqui de proposito: so o endpoint de upload pode
	// gravar a foto, senao o cliente poderia apontar para javascript:, para um
	// host de terceiros ou para arquivo de outro tenant.
}

// Update edita campos do perfil do membro.
// Semantica de PATCH: campo nil (ausente no JSON) mantem o valor atual.
//
// Quando a situacao muda, grava automaticamente uma entrada no historico
// eclesiastico (requisito 1.8) - o historico e consequencia, nao digitacao.
func (r *Repo) Update(ctx context.Context, tx pgx.Tx, id string, in UpdateInput, actorID string) (*Member, error) {
	var oldStatus string
	if err := tx.QueryRow(ctx,
		`SELECT membership_status FROM members WHERE id = $1::uuid`, id).Scan(&oldStatus); err != nil {
		return nil, err
	}

	// O UPDATE devolve so o id e a leitura completa vem do Get: `RETURNING` com
	// subquery (cargos/card_ref) enxerga o snapshot ANTERIOR a escrita, entao
	// devolveria cargos desatualizados.
	var updatedID string
	err := tx.QueryRow(ctx, `
		UPDATE members m SET
			first_name = COALESCE($2, m.first_name),
			last_name = COALESCE($3, m.last_name),
			-- full_name e coluna persistida: precisa acompanhar nome/sobrenome.
			full_name = btrim(COALESCE($2, m.first_name) || ' ' || COALESCE($3, m.last_name)),
			nickname = COALESCE($4, m.nickname),
			email = COALESCE($5, m.email),
			phone = COALESCE($6, m.phone),
			whatsapp = COALESCE($7, m.whatsapp),
			birth_date = COALESCE($8::date, m.birth_date),
			gender = COALESCE($9, m.gender),
			marital_status = COALESCE($10, m.marital_status),
			profession = COALESCE($11, m.profession),
			office = COALESCE($12, m.office),
			nationality = COALESCE($23, m.nationality),
			education = COALESCE($24, m.education),
			notes = COALESCE($25, m.notes),
			membership_status = COALESCE($13, m.membership_status),
			cpf = COALESCE($14, m.cpf),
			rg = COALESCE($15, m.rg),
			baptism_date = COALESCE($16::date, m.baptism_date),
			baptism_location = COALESCE($17, m.baptism_location),
			joined_at = COALESCE($18::date, m.joined_at),
			address = COALESCE($19::jsonb, m.address),
			exit_reason = COALESCE($20, m.exit_reason),
			exited_at = COALESCE($21::date, m.exited_at),
			marriage_date = COALESCE($22::date, m.marriage_date),
			updated_at = now()
		WHERE m.id = $1::uuid
		RETURNING m.id::text`,
		id, in.FirstName, in.LastName, in.Nickname, in.Email, in.Phone, in.Whatsapp,
		in.BirthDate, in.Gender, in.MaritalStatus, in.Profession, in.Office,
		in.MembershipStatus, in.CPF, in.RG, in.BaptismDate, in.BaptismLocation,
		in.JoinedAt, in.Address, in.ExitReason, in.ExitedAt, in.MarriageDate,
		in.Nationality, in.Education, in.Notes).Scan(&updatedID)
	if err != nil {
		return nil, err
	}

	newStatus := oldStatus
	if in.MembershipStatus != nil && *in.MembershipStatus != "" {
		newStatus = *in.MembershipStatus
	}
	if newStatus != oldStatus {
		notes := "Situacao alterada de '" + oldStatus + "' para '" + newStatus + "'"
		if in.ExitReason != nil && *in.ExitReason != "" {
			notes += " (motivo: " + *in.ExitReason + ")"
		}
		if err := insertHistory(ctx, tx, id, historyKindForStatus(newStatus), notes, actorID); err != nil {
			return nil, err
		}
	}
	return r.Get(ctx, tx, updatedID)
}

// SetPhoto grava a foto do membro. Chamado apenas pelo endpoint de upload, que
// monta a URL no servidor.
func (r *Repo) SetPhoto(ctx context.Context, tx pgx.Tx, id string, photoURL *string) (*Member, error) {
	var updatedID string
	err := tx.QueryRow(ctx, `
		UPDATE members SET photo_url = $2, updated_at = now()
		WHERE id = $1::uuid RETURNING id::text`, id, photoURL).Scan(&updatedID)
	if err != nil {
		return nil, err
	}
	return r.Get(ctx, tx, updatedID)
}

// Delete remove definitivamente o membro no escopo da sessao (RLS). As tabelas
// dependentes (historico, mandatos, presencas, etc.) tem FK ON DELETE
// CASCADE/SET NULL, entao o Postgres limpa/desvincula os registros associados.
func (r *Repo) Delete(ctx context.Context, tx pgx.Tx, id string) error {
	tag, err := tx.Exec(ctx, `DELETE FROM members WHERE id = $1::uuid`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

type CreateInput struct {
	FirstName        string   `json:"first_name"`
	LastName         string   `json:"last_name"`
	Nickname         *string  `json:"nickname"`
	Email            *string  `json:"email"`
	Phone            *string  `json:"phone"`
	Whatsapp         *string  `json:"whatsapp"`
	BirthDate        *string  `json:"birth_date"`
	Gender           *string  `json:"gender"`
	MaritalStatus    *string  `json:"marital_status"`
	Profession       *string  `json:"profession"`
	Office           *string  `json:"office"`
	Nationality      *string  `json:"nationality"`
	Education        *string  `json:"education"`
	Notes            *string  `json:"notes"`
	MembershipStatus string   `json:"membership_status"`
	CPF              *string  `json:"cpf"`
	RG               *string  `json:"rg"`
	BaptismDate      *string  `json:"baptism_date"`
	BaptismLocation  *string  `json:"baptism_location"`
	JoinedAt         *string  `json:"joined_at"`
	Address          *Address `json:"address"`
	MarriageDate     *string  `json:"marriage_date"`
	// photo_url fica de fora: so o upload grava a foto (ver UpdateInput).
}

// Create insere um membro. O branch_id vem da sessao RLS (nao do client).
func (r *Repo) Create(ctx context.Context, tx pgx.Tx, tenantID, branchID string, in CreateInput, actorID string) (*Member, error) {
	status := in.MembershipStatus
	if status == "" {
		status = "member"
	}
	full := strings.TrimSpace(in.FirstName + " " + in.LastName)
	var newID string
	err := tx.QueryRow(ctx, `
		INSERT INTO members
			(tenant_id, branch_id, first_name, last_name, full_name,
			 nickname, email, phone, whatsapp, birth_date, gender,
			 marital_status, profession, office, membership_status,
			 cpf, rg, baptism_date, baptism_location, joined_at, address, marriage_date,
			 nationality, education, notes)
		VALUES
			($1, $2, $3, $4, $5, $6, $7, $8, $9, $10::date, $11,
			 $12, $13, $14, $15, $16, $17, $18::date, $19, $20::date, $21::jsonb, $22::date,
			 $23, $24, $25)
		RETURNING id::text`,
		tenantID, branchID, in.FirstName, in.LastName, full,
		in.Nickname, in.Email, in.Phone, in.Whatsapp, in.BirthDate, in.Gender,
		in.MaritalStatus, in.Profession, in.Office, status,
		in.CPF, in.RG, in.BaptismDate, in.BaptismLocation, in.JoinedAt,
		in.Address, in.MarriageDate, in.Nationality, in.Education, in.Notes).Scan(&newID)
	if err != nil {
		return nil, err
	}
	// Primeira entrada do historico (linha do tempo comeca no cadastro).
	if err := insertHistory(ctx, tx, newID, "cadastro", "Membro cadastrado", actorID); err != nil {
		return nil, err
	}
	return r.Get(ctx, tx, newID)
}
