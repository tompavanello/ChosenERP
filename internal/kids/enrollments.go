package kids

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// Enrollment é a matrícula de um membro criança em uma turma.
type Enrollment struct {
	ID                  string    `json:"id"`
	ClassID             string    `json:"class_id"`
	ClassName           string    `json:"class_name"`
	MemberID            string    `json:"member_id"`
	MemberName          string    `json:"member_name"`
	BirthDate           *string   `json:"birth_date,omitempty"`
	Status              string    `json:"status"`
	StartDate           string    `json:"start_date"`
	EndDate             *string   `json:"end_date,omitempty"`
	DietaryRestrictions string    `json:"dietary_restrictions"`
	Notes               string    `json:"notes"`
	GuardiansCount      int       `json:"guardians_count"`
	CreatedAt           time.Time `json:"created_at"`
}

type EnrollmentInput struct {
	MemberID            *string `json:"member_id"`
	Status              *string `json:"status"`
	StartDate           *string `json:"start_date"`
	EndDate             *string `json:"end_date"`
	DietaryRestrictions *string `json:"dietary_restrictions"`
	Notes               *string `json:"notes"`
}

const enrollmentSelect = `
	e.id::text, e.class_id::text, c.name, e.member_id::text, m.full_name,
	m.birth_date::text, e.status, e.start_date::text, e.end_date::text,
	COALESCE(e.dietary_restrictions,''), COALESCE(e.notes,''),
	(SELECT count(*) FROM kids_guardians g WHERE g.enrollment_id = e.id), e.created_at
	FROM kids_enrollments e
	JOIN kids_classes c ON c.id = e.class_id
	JOIN members m ON m.id = e.member_id`

func scanEnrollment(s rowScanner) (*Enrollment, error) {
	var e Enrollment
	err := s.Scan(&e.ID, &e.ClassID, &e.ClassName, &e.MemberID, &e.MemberName,
		&e.BirthDate, &e.Status, &e.StartDate, &e.EndDate,
		&e.DietaryRestrictions, &e.Notes, &e.GuardiansCount, &e.CreatedAt)
	return &e, err
}

func (r *Repo) ListEnrollments(ctx context.Context, tx pgx.Tx, classID string) ([]Enrollment, error) {
	rows, err := tx.Query(ctx, `SELECT `+enrollmentSelect+` WHERE e.class_id=$1::uuid ORDER BY m.full_name`, classID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Enrollment{}
	for rows.Next() {
		e, err := scanEnrollment(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *e)
	}
	return out, rows.Err()
}

func (r *Repo) GetEnrollment(ctx context.Context, tx pgx.Tx, id string) (*Enrollment, error) {
	return scanEnrollment(tx.QueryRow(ctx, `SELECT `+enrollmentSelect+` WHERE e.id=$1::uuid`, id))
}

func (r *Repo) CreateEnrollment(ctx context.Context, tx pgx.Tx, classID string, in EnrollmentInput) (*Enrollment, error) {
	if in.MemberID == nil || *in.MemberID == "" {
		return nil, fmt.Errorf("%w: member_id é obrigatório", ErrInvalidInput)
	}
	// A filial da matrícula segue a turma (garante coerência de escopo).
	class, err := r.GetClass(ctx, tx, classID)
	if err != nil {
		return nil, err
	}
	var id string
	err = tx.QueryRow(ctx, `
		INSERT INTO kids_enrollments (tenant_id, branch_id, class_id, member_id, status, start_date, end_date, dietary_restrictions, notes)
		VALUES ($1, $2::uuid, $3::uuid, $4::uuid, COALESCE(NULLIF($5,''),'active'),
		        COALESCE(NULLIF($6,'')::date, current_date), NULLIF($7,'')::date, NULLIF($8,''), NULLIF($9,''))
		RETURNING id::text`,
		class.TenantID, class.BranchID, classID, *in.MemberID, strOrEmpty(in.Status),
		strOrEmpty(in.StartDate), strOrEmpty(in.EndDate),
		strOrEmpty(in.DietaryRestrictions), strOrEmpty(in.Notes)).Scan(&id)
	if err != nil {
		return nil, err
	}
	return r.GetEnrollment(ctx, tx, id)
}

func (r *Repo) UpdateEnrollment(ctx context.Context, tx pgx.Tx, id string, in EnrollmentInput) (*Enrollment, error) {
	e, err := r.GetEnrollment(ctx, tx, id)
	if err != nil {
		return nil, err
	}
	if in.Status != nil && *in.Status != "" {
		e.Status = *in.Status
	}
	if in.StartDate != nil && *in.StartDate != "" {
		e.StartDate = *in.StartDate
	}
	if in.EndDate != nil {
		if *in.EndDate == "" {
			e.EndDate = nil
		} else {
			e.EndDate = in.EndDate
		}
	}
	if in.DietaryRestrictions != nil {
		e.DietaryRestrictions = *in.DietaryRestrictions
	}
	if in.Notes != nil {
		e.Notes = *in.Notes
	}
	_, err = tx.Exec(ctx, `
		UPDATE kids_enrollments SET status=$2, start_date=$3::date, end_date=NULLIF($4,'')::date,
		       dietary_restrictions=NULLIF($5,''), notes=NULLIF($6,'')
		WHERE id=$1::uuid`,
		id, e.Status, e.StartDate, strOrEmpty(e.EndDate), e.DietaryRestrictions, e.Notes)
	if err != nil {
		return nil, err
	}
	return r.GetEnrollment(ctx, tx, id)
}

func (r *Repo) DeleteEnrollment(ctx context.Context, tx pgx.Tx, id string) error {
	return execExpectRow(ctx, tx, `DELETE FROM kids_enrollments WHERE id=$1::uuid`, id)
}

// ---------------------------------------------------------------------------
// Responsáveis
// ---------------------------------------------------------------------------

type Guardian struct {
	ID           string `json:"id"`
	EnrollmentID string `json:"enrollment_id"`
	MemberID     string `json:"member_id"`
	MemberName   string `json:"member_name"`
	Relationship string `json:"relationship"`
	IsPrimary    bool   `json:"is_primary"`
}

type GuardianInput struct {
	MemberID     *string `json:"member_id"`
	Relationship *string `json:"relationship"`
	IsPrimary    *bool   `json:"is_primary"`
}

func (r *Repo) ListGuardians(ctx context.Context, tx pgx.Tx, enrollmentID string) ([]Guardian, error) {
	rows, err := tx.Query(ctx, `
		SELECT g.id::text, g.enrollment_id::text, g.member_id::text, m.full_name,
		       COALESCE(g.relationship,''), g.is_primary
		FROM kids_guardians g
		JOIN members m ON m.id = g.member_id
		WHERE g.enrollment_id=$1::uuid
		ORDER BY g.is_primary DESC, m.full_name`, enrollmentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Guardian{}
	for rows.Next() {
		var g Guardian
		if err := rows.Scan(&g.ID, &g.EnrollmentID, &g.MemberID, &g.MemberName,
			&g.Relationship, &g.IsPrimary); err != nil {
			return nil, err
		}
		out = append(out, g)
	}
	return out, rows.Err()
}

func (r *Repo) AddGuardian(ctx context.Context, tx pgx.Tx, enrollmentID string, in GuardianInput) (*Guardian, error) {
	if in.MemberID == nil || *in.MemberID == "" {
		return nil, fmt.Errorf("%w: member_id é obrigatório", ErrInvalidInput)
	}
	enr, err := r.GetEnrollment(ctx, tx, enrollmentID)
	if err != nil {
		return nil, err
	}
	var id string
	err = tx.QueryRow(ctx, `
		INSERT INTO kids_guardians (tenant_id, branch_id, enrollment_id, member_id, relationship, is_primary)
		VALUES (
			(SELECT tenant_id FROM kids_enrollments WHERE id=$1::uuid),
			(SELECT branch_id FROM kids_enrollments WHERE id=$1::uuid),
			$1::uuid, $2::uuid, NULLIF($3,''), COALESCE($4,false))
		RETURNING id::text`,
		enrollmentID, *in.MemberID, strOrEmpty(in.Relationship), in.IsPrimary).Scan(&id)
	if err != nil {
		return nil, err
	}
	_ = enr
	var g Guardian
	err = tx.QueryRow(ctx, `
		SELECT g.id::text, g.enrollment_id::text, g.member_id::text, m.full_name,
		       COALESCE(g.relationship,''), g.is_primary
		FROM kids_guardians g JOIN members m ON m.id=g.member_id
		WHERE g.id=$1::uuid`, id).
		Scan(&g.ID, &g.EnrollmentID, &g.MemberID, &g.MemberName, &g.Relationship, &g.IsPrimary)
	return &g, err
}

func (r *Repo) DeleteGuardian(ctx context.Context, tx pgx.Tx, id string) error {
	return execExpectRow(ctx, tx, `DELETE FROM kids_guardians WHERE id=$1::uuid`, id)
}
