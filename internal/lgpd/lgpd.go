// Package lgpd cobre o compliance de dados (consentimento, portabilidade e
// anonimizacao). O isolamento e do RLS; a autorizacao e do handler.
package lgpd

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5"
)

type Term struct {
	ID        string    `json:"id"`
	Version   int       `json:"version"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
}

type Consent struct {
	ID          string     `json:"id"`
	TermID      string     `json:"term_id"`
	TermTitle   string     `json:"term_title"`
	Version     int        `json:"version"`
	SubjectType string     `json:"subject_type"`
	SubjectID   string     `json:"subject_id"`
	Consented   bool       `json:"consented"`
	ConsentedAt *time.Time `json:"consented_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

type Repo struct{}

// ---- Termos de consentimento ----

func (r *Repo) ListTerms(ctx context.Context, tx pgx.Tx) ([]Term, error) {
	rows, err := tx.Query(ctx, `
		SELECT id::text, version, title, body, is_active, created_at
		FROM consent_terms ORDER BY version DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Term{}
	for rows.Next() {
		var t Term
		if err := rows.Scan(&t.ID, &t.Version, &t.Title, &t.Body, &t.IsActive, &t.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (r *Repo) CreateTerm(ctx context.Context, tx pgx.Tx, tenantID, title, body string) (*Term, error) {
	var t Term
	err := tx.QueryRow(ctx, `
		WITH next AS (SELECT COALESCE(MAX(version), 0) + 1 AS v FROM consent_terms WHERE tenant_id = $1)
		INSERT INTO consent_terms (tenant_id, version, title, body, is_active)
		SELECT $1, next.v, $2, $3, true FROM next
		RETURNING id::text, version, title, body, is_active, created_at`,
		tenantID, title, body).
		Scan(&t.ID, &t.Version, &t.Title, &t.Body, &t.IsActive, &t.CreatedAt)
	return &t, err
}

// ---- Consentimentos ----

func (r *Repo) ListConsents(ctx context.Context, tx pgx.Tx, subjectType, subjectID string) ([]Consent, error) {
	rows, err := tx.Query(ctx, `
		SELECT c.id::text, c.consent_term_id::text, t.title, t.version,
		       c.subject_type, c.subject_id::text, c.consented, c.consented_at, c.created_at
		FROM member_consents c
		JOIN consent_terms t ON t.id = c.consent_term_id
		WHERE c.subject_type = $1 AND c.subject_id = $2::uuid
		ORDER BY c.created_at DESC`, subjectType, subjectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Consent{}
	for rows.Next() {
		var c Consent
		if err := rows.Scan(&c.ID, &c.TermID, &c.TermTitle, &c.Version, &c.SubjectType,
			&c.SubjectID, &c.Consented, &c.ConsentedAt, &c.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// RecordConsent grava um consentimento (ou revogacao) para uma pessoa.
func (r *Repo) RecordConsent(ctx context.Context, tx pgx.Tx, tenantID, branchID, subjectType, subjectID, termID string, consented bool, ip, userAgent string) (*Consent, error) {
	var c Consent
	var consentedAt *time.Time
	if consented {
		now := time.Now()
		consentedAt = &now
	}
	err := tx.QueryRow(ctx, `
		INSERT INTO member_consents (tenant_id, branch_id, consent_term_id, subject_type, subject_id, consented, consented_at, ip, user_agent)
		VALUES ($1, NULLIF($2,'')::uuid, $3::uuid, $4, $5::uuid, $6, $7, NULLIF($8,''), NULLIF($9,''))
		RETURNING id::text, consent_term_id::text, subject_type, subject_id::text, consented, consented_at, created_at`,
		tenantID, branchID, termID, subjectType, subjectID, consented, consentedAt, ip, userAgent).
		Scan(&c.ID, &c.TermID, &c.SubjectType, &c.SubjectID, &c.Consented, &c.ConsentedAt, &c.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// ---- Portabilidade (exportacao dos dados do titular) ----

// ExportMember devolve, em JSON, tudo que o sistema guarda sobre o membro.
func (r *Repo) ExportMember(ctx context.Context, tx pgx.Tx, memberID string) ([]byte, error) {
	var exists bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM members WHERE id = $1::uuid)`, memberID).Scan(&exists); err != nil {
		return nil, err
	}
	if !exists {
		return nil, pgx.ErrNoRows
	}
	agg := func(sql string) (json.RawMessage, error) {
		var s *string
		if err := tx.QueryRow(ctx, sql, memberID).Scan(&s); err != nil {
			return nil, err
		}
		if s == nil || *s == "" {
			return json.RawMessage("null"), nil
		}
		return json.RawMessage(*s), nil
	}
	rows := func(sql string) (json.RawMessage, error) {
		var s *string
		if err := tx.QueryRow(ctx, sql, memberID).Scan(&s); err != nil {
			return nil, err
		}
		if s == nil || *s == "" {
			return json.RawMessage("[]"), nil
		}
		return json.RawMessage(*s), nil
	}

	out := map[string]any{
		"exported_at": time.Now().UTC().Format(time.RFC3339),
		"lgpd_notice": "Dados pessoais exportados a pedido do titular (LGPD art. 18).",
	}
	var err error
	if out["member"], err = agg(`SELECT to_json(t)::text FROM (
		SELECT id, full_name, nickname, email, phone, whatsapp, birth_date::text, gender,
		       marital_status, membership_status, profession, cpf, rg, address,
		       marriage_date::text, joined_at::text, baptism_date::text, baptism_location, created_at
		FROM members WHERE id = $1::uuid) t`); err != nil {
		return nil, err
	}
	if out["cargos"], err = rows(`SELECT json_agg(t ORDER BY t.started_at)::text FROM (
		SELECT mc.id, c.name, c.kind, mc.started_at::text, mc.ends_at::text, mc.status
		FROM member_cargos mc JOIN cargos c ON c.id = mc.cargo_id WHERE mc.member_id = $1::uuid) t`); err != nil {
		return nil, err
	}
	if out["relationships"], err = rows(`SELECT json_agg(t)::text FROM (
		SELECT r.kind, m.full_name AS related_name
		FROM member_relationships r JOIN members m ON m.id = r.related_id
		WHERE r.member_id = $1::uuid) t`); err != nil {
		return nil, err
	}
	if out["families"], err = rows(`SELECT json_agg(t)::text FROM (
		SELECT f.name, f.code FROM families f
		WHERE f.id IN (SELECT family_id FROM member_relationships WHERE member_id = $1::uuid AND family_id IS NOT NULL)
		   OR f.head_id = $1::uuid) t`); err != nil {
		return nil, err
	}
	if out["history"], err = rows(`SELECT json_agg(t ORDER BY t.occurred_at)::text FROM (
		SELECT occurred_at::text, kind, notes FROM member_history WHERE member_id = $1::uuid) t`); err != nil {
		return nil, err
	}
	if out["frequency"], err = rows(`SELECT json_agg(t ORDER BY t.started_at)::text FROM (
		SELECT frequency, started_at::text, ended_at::text, notes FROM member_frequency_history WHERE member_id = $1::uuid) t`); err != nil {
		return nil, err
	}
	if out["consents"], err = rows(`SELECT json_agg(t ORDER BY t.created_at)::text FROM (
		SELECT ct.title, ct.version, mc.consented, mc.consented_at::text, mc.created_at
		FROM member_consents mc JOIN consent_terms ct ON ct.id = mc.consent_term_id
		WHERE mc.subject_type = 'member' AND mc.subject_id = $1::uuid) t`); err != nil {
		return nil, err
	}
	if out["donations"], err = rows(`SELECT json_agg(t ORDER BY t.occurred_at)::text FROM (
		SELECT type, amount::float8, currency, occurred_at::text, description
		FROM financial_transactions WHERE donor_member_id = $1::uuid) t`); err != nil {
		return nil, err
	}
	if out["attendance"], err = rows(`SELECT json_agg(t ORDER BY t.starts_at)::text FROM (
		SELECT e.starts_at::text, e.kind_id::text, a.present
		FROM event_attendance a JOIN church_events e ON e.id = a.event_id
		WHERE a.member_id = $1::uuid) t`); err != nil {
		return nil, err
	}
	return json.MarshalIndent(out, "", "  ")
}

// ---- Anonimizacao ----

// AnonymizeMember apaga os dados pessoais do membro mantendo o id (integridade
// referencial e retencao fiscal dos lancamentos). Registra a acao no historico.
func (r *Repo) AnonymizeMember(ctx context.Context, tx pgx.Tx, memberID, actorID string) error {
	tag, err := tx.Exec(ctx, `
		UPDATE members SET
			first_name = 'Anonimizado',
			last_name = left(id::text, 8),
			full_name = 'Membro anonimizado',
			nickname = NULL, email = NULL, phone = NULL, whatsapp = NULL,
			birth_date = NULL, cpf = NULL, rg = NULL, address = NULL,
			photo_url = NULL, profession = NULL, marriage_date = NULL,
			baptism_location = NULL, extra_json = '{}'::jsonb,
			updated_at = now()
		WHERE id = $1::uuid`, memberID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO member_history (tenant_id, branch_id, member_id, kind, notes, created_by)
		SELECT m.tenant_id, m.branch_id, m.id, 'anonimizacao', 'Dados pessoais anonimizados (LGPD)', $2::uuid
		FROM members m WHERE m.id = $1::uuid`, memberID, actorID)
	return err
}
