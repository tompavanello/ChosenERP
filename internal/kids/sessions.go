package kids

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"time"

	"github.com/jackc/pgx/v5"
)

// Session é um encontro da turma (ministra uma lição).
type Session struct {
	ID           string     `json:"id"`
	ClassID      string     `json:"class_id"`
	ClassName    string     `json:"class_name"`
	LessonID     string     `json:"lesson_id"`
	LessonTitle  string     `json:"lesson_title"`
	LessonPos    int        `json:"lesson_position"`
	StartsAt     time.Time  `json:"starts_at"`
	EndsAt       *time.Time `json:"ends_at,omitempty"`
	Status       string     `json:"status"`
	Notes        string     `json:"notes"`
	CheckinCount int        `json:"checkin_count"`
}

type SessionInput struct {
	ClassID  *string `json:"class_id"`
	LessonID *string `json:"lesson_id"`
	StartsAt *string `json:"starts_at"`
	EndsAt   *string `json:"ends_at"`
	Status   *string `json:"status"`
	Notes    *string `json:"notes"`
}

const sessionSelect = `
	s.id::text, s.class_id::text, c.name, COALESCE(s.lesson_id::text,''),
	COALESCE(l.title,''), COALESCE(l.position,0), s.starts_at, s.ends_at, s.status,
	COALESCE(s.notes,''),
	(SELECT count(*) FROM kids_checkins k WHERE k.session_id=s.id AND k.status='present')
	FROM kids_sessions s
	JOIN kids_classes c ON c.id = s.class_id
	LEFT JOIN kids_lessons l ON l.id = s.lesson_id`

func scanSession(s rowScanner) (*Session, error) {
	var x Session
	err := s.Scan(&x.ID, &x.ClassID, &x.ClassName, &x.LessonID, &x.LessonTitle,
		&x.LessonPos, &x.StartsAt, &x.EndsAt, &x.Status, &x.Notes, &x.CheckinCount)
	return &x, err
}

func (r *Repo) ListSessions(ctx context.Context, tx pgx.Tx, classID, from, to string) ([]Session, error) {
	q := `SELECT ` + sessionSelect + ` WHERE ($1='' OR s.class_id=$1::uuid)
	      AND ($2='' OR s.starts_at >= $2::timestamptz)
	      AND ($3='' OR s.starts_at <= $3::timestamptz)
	      ORDER BY s.starts_at DESC`
	rows, err := tx.Query(ctx, q, classID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Session{}
	for rows.Next() {
		s, err := scanSession(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *s)
	}
	return out, rows.Err()
}

func (r *Repo) GetSession(ctx context.Context, tx pgx.Tx, id string) (*Session, error) {
	return scanSession(tx.QueryRow(ctx, `SELECT `+sessionSelect+` WHERE s.id=$1::uuid`, id))
}

func (r *Repo) CreateSession(ctx context.Context, tx pgx.Tx, in SessionInput) (*Session, error) {
	if in.ClassID == nil || *in.ClassID == "" || in.StartsAt == nil || *in.StartsAt == "" {
		return nil, fmt.Errorf("%w: class_id e starts_at são obrigatórios", ErrInvalidInput)
	}
	// A filial do encontro segue a turma (a Sede não tem branch própria).
	class, err := r.GetClass(ctx, tx, *in.ClassID)
	if err != nil {
		return nil, err
	}
	var id string
	err = tx.QueryRow(ctx, `
		INSERT INTO kids_sessions (tenant_id, branch_id, class_id, lesson_id, starts_at, ends_at, status, notes)
		VALUES ($1, $2::uuid, $3::uuid, NULLIF($4,'')::uuid, $5::timestamptz,
		        NULLIF($6,'')::timestamptz, COALESCE(NULLIF($7,''),'scheduled'), NULLIF($8,''))
		RETURNING id::text`,
		class.TenantID, class.BranchID, *in.ClassID, strOrEmpty(in.LessonID), *in.StartsAt,
		strOrEmpty(in.EndsAt), strOrEmpty(in.Status), strOrEmpty(in.Notes)).Scan(&id)
	if err != nil {
		return nil, err
	}
	return r.GetSession(ctx, tx, id)
}

func (r *Repo) UpdateSession(ctx context.Context, tx pgx.Tx, id string, in SessionInput) (*Session, error) {
	s, err := r.GetSession(ctx, tx, id)
	if err != nil {
		return nil, err
	}
	if in.LessonID != nil {
		s.LessonID = *in.LessonID
	}
	if in.StartsAt != nil && *in.StartsAt != "" {
		t, perr := time.Parse(time.RFC3339, *in.StartsAt)
		if perr != nil {
			return nil, fmt.Errorf("%w: starts_at inválido", ErrInvalidInput)
		}
		s.StartsAt = t
	}
	if in.EndsAt != nil {
		if *in.EndsAt == "" {
			s.EndsAt = nil
		} else {
			t, perr := time.Parse(time.RFC3339, *in.EndsAt)
			if perr != nil {
				return nil, fmt.Errorf("%w: ends_at inválido", ErrInvalidInput)
			}
			s.EndsAt = &t
		}
	}
	if in.Status != nil && *in.Status != "" {
		s.Status = *in.Status
	}
	if in.Notes != nil {
		s.Notes = *in.Notes
	}
	_, err = tx.Exec(ctx, `
		UPDATE kids_sessions SET lesson_id=NULLIF($2,'')::uuid, starts_at=$3::timestamptz,
		       ends_at=NULLIF($4,'')::timestamptz, status=$5, notes=NULLIF($6,'')
		WHERE id=$1::uuid`,
		id, s.LessonID, s.StartsAt.Format(time.RFC3339), timeOrEmpty(s.EndsAt), s.Status, s.Notes)
	if err != nil {
		return nil, err
	}
	return r.GetSession(ctx, tx, id)
}

func (r *Repo) DeleteSession(ctx context.Context, tx pgx.Tx, id string) error {
	return execExpectRow(ctx, tx, `DELETE FROM kids_sessions WHERE id=$1::uuid`, id)
}

// ---------------------------------------------------------------------------
// Check-in / check-out
// ---------------------------------------------------------------------------

// RosterEntry é a criança na chamada de um encontro.
type RosterEntry struct {
	CheckinID           string     `json:"checkin_id"`
	EnrollmentID        string     `json:"enrollment_id"`
	MemberID            string     `json:"member_id"`
	MemberName          string     `json:"member_name"`
	BirthDate           *string    `json:"birth_date,omitempty"`
	Status              string     `json:"status"` // present | absent | "" (não marcado)
	SecurityCode        string     `json:"security_code"`
	CheckinAt           *time.Time `json:"checkin_at,omitempty"`
	CheckoutAt          *time.Time `json:"checkout_at,omitempty"`
	DropoffGuardianID   string     `json:"dropoff_guardian_id"`
	PickupGuardianID    string     `json:"pickup_guardian_id"`
	DietaryRestrictions string     `json:"dietary_restrictions"`
	Guardians           string     `json:"guardians"`
}

// Roster lista as crianças ativas da turma do encontro com o estado do check-in.
func (r *Repo) Roster(ctx context.Context, tx pgx.Tx, sessionID string) ([]RosterEntry, error) {
	rows, err := tx.Query(ctx, `
		SELECT COALESCE(k.id::text,''), e.id::text, e.member_id::text, m.full_name,
		       m.birth_date::text, COALESCE(k.status,''), COALESCE(k.security_code,''),
		       k.checkin_at, k.checkout_at,
		       COALESCE(k.dropoff_guardian_id::text,''), COALESCE(k.pickup_guardian_id::text,''),
		       COALESCE(e.dietary_restrictions,''),
		       COALESCE(string_agg(DISTINCT gm.full_name, ', '), '')
		FROM kids_enrollments e
		JOIN members m ON m.id = e.member_id
		LEFT JOIN kids_checkins k ON k.session_id=$1::uuid AND k.enrollment_id=e.id
		LEFT JOIN kids_guardians g ON g.enrollment_id=e.id
		LEFT JOIN members gm ON gm.id=g.member_id
		WHERE e.class_id = (SELECT class_id FROM kids_sessions WHERE id=$1::uuid)
		  AND e.status='active'
		GROUP BY k.id, e.id, m.id
		ORDER BY m.full_name`, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []RosterEntry{}
	for rows.Next() {
		var e RosterEntry
		if err := rows.Scan(&e.CheckinID, &e.EnrollmentID, &e.MemberID, &e.MemberName,
			&e.BirthDate, &e.Status, &e.SecurityCode, &e.CheckinAt, &e.CheckoutAt,
			&e.DropoffGuardianID, &e.PickupGuardianID, &e.DietaryRestrictions, &e.Guardians); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

type CheckinInput struct {
	EnrollmentID      string `json:"enrollment_id"`
	DropoffGuardianID string `json:"dropoff_guardian_id"`
}

// CheckIn registra a entrada de uma criança, gerando o código de segurança.
func (r *Repo) CheckIn(ctx context.Context, tx pgx.Tx, sessionID string, in CheckinInput) (*RosterEntry, error) {
	sess, err := r.GetSession(ctx, tx, sessionID)
	if err != nil {
		return nil, err
	}
	enr, err := r.GetEnrollment(ctx, tx, in.EnrollmentID)
	if err != nil {
		return nil, err
	}
	if enr.ClassID != sess.ClassID {
		return nil, fmt.Errorf("%w: matrícula não pertence à turma do encontro", ErrInvalidInput)
	}
	code := newSecurityCode()
	_, err = tx.Exec(ctx, `
		INSERT INTO kids_checkins (tenant_id, branch_id, session_id, enrollment_id, status, security_code, checkin_at, dropoff_guardian_id)
		VALUES (
			(SELECT tenant_id FROM kids_sessions WHERE id=$1::uuid),
			(SELECT branch_id FROM kids_sessions WHERE id=$1::uuid),
			$1::uuid, $2::uuid, 'present', $3, now(), NULLIF($4,'')::uuid)
		ON CONFLICT (session_id, enrollment_id) DO UPDATE SET
			status='present',
			checkin_at=COALESCE(kids_checkins.checkin_at, now()),
			security_code=COALESCE(kids_checkins.security_code, EXCLUDED.security_code),
			dropoff_guardian_id=COALESCE(EXCLUDED.dropoff_guardian_id, kids_checkins.dropoff_guardian_id),
			checkout_at=NULL`,
		sessionID, in.EnrollmentID, code, in.DropoffGuardianID)
	if err != nil {
		return nil, err
	}
	return r.rosterEntry(ctx, tx, sessionID, in.EnrollmentID)
}

// CheckOut finaliza a saída; quando o código é informado, precisa conferir.
func (r *Repo) CheckOut(ctx context.Context, tx pgx.Tx, sessionID, enrollmentID, code, pickupGuardianID string) (*RosterEntry, error) {
	var stored *string
	err := tx.QueryRow(ctx, `
		SELECT security_code FROM kids_checkins
		WHERE session_id=$1::uuid AND enrollment_id=$2::uuid`, sessionID, enrollmentID).Scan(&stored)
	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("%w: a criança ainda não fez check-in", ErrInvalidInput)
	}
	if err != nil {
		return nil, err
	}
	if code != "" && stored != nil && code != *stored {
		return nil, fmt.Errorf("%w: código de segurança incorreto", ErrInvalidInput)
	}
	_, err = tx.Exec(ctx, `
		UPDATE kids_checkins SET checkout_at=now(), pickup_guardian_id=NULLIF($3,'')::uuid
		WHERE session_id=$1::uuid AND enrollment_id=$2::uuid`,
		sessionID, enrollmentID, pickupGuardianID)
	if err != nil {
		return nil, err
	}
	return r.rosterEntry(ctx, tx, sessionID, enrollmentID)
}

// MarkAbsent registra ausência (ou limpa o check-in com status ”).
func (r *Repo) MarkAbsent(ctx context.Context, tx pgx.Tx, sessionID, enrollmentID string, absent bool) error {
	if !absent {
		return execExpectRow(ctx, tx, `
			DELETE FROM kids_checkins WHERE session_id=$1::uuid AND enrollment_id=$2::uuid`,
			sessionID, enrollmentID)
	}
	_, err := tx.Exec(ctx, `
		INSERT INTO kids_checkins (tenant_id, branch_id, session_id, enrollment_id, status)
		VALUES (
			(SELECT tenant_id FROM kids_sessions WHERE id=$1::uuid),
			(SELECT branch_id FROM kids_sessions WHERE id=$1::uuid),
			$1::uuid, $2::uuid, 'absent')
		ON CONFLICT (session_id, enrollment_id) DO UPDATE SET
			status='absent', checkin_at=NULL, checkout_at=NULL, security_code=NULL`,
		sessionID, enrollmentID)
	return err
}

func (r *Repo) rosterEntry(ctx context.Context, tx pgx.Tx, sessionID, enrollmentID string) (*RosterEntry, error) {
	var e RosterEntry
	err := tx.QueryRow(ctx, `
		SELECT COALESCE(k.id::text,''), e.id::text, e.member_id::text, m.full_name,
		       m.birth_date::text, COALESCE(k.status,''), COALESCE(k.security_code,''),
		       k.checkin_at, k.checkout_at,
		       COALESCE(k.dropoff_guardian_id::text,''), COALESCE(k.pickup_guardian_id::text,''),
		       COALESCE(e.dietary_restrictions,''),
		       COALESCE((SELECT string_agg(DISTINCT gm.full_name, ', ')
		                 FROM kids_guardians g JOIN members gm ON gm.id=g.member_id
		                 WHERE g.enrollment_id=e.id), '')
		FROM kids_enrollments e
		JOIN members m ON m.id=e.member_id
		LEFT JOIN kids_checkins k ON k.session_id=$1::uuid AND k.enrollment_id=e.id
		WHERE e.id=$2::uuid`, sessionID, enrollmentID).
		Scan(&e.CheckinID, &e.EnrollmentID, &e.MemberID, &e.MemberName, &e.BirthDate,
			&e.Status, &e.SecurityCode, &e.CheckinAt, &e.CheckoutAt,
			&e.DropoffGuardianID, &e.PickupGuardianID, &e.DietaryRestrictions, &e.Guardians)
	return &e, err
}

// newSecurityCode gera um código numérico de 4 dígitos.
func newSecurityCode() string {
	n, err := rand.Int(rand.Reader, big.NewInt(10000))
	if err != nil {
		return "0000"
	}
	return fmt.Sprintf("%04d", n.Int64())
}

func timeOrEmpty(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format(time.RFC3339)
}
