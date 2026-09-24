package kids

import (
	"context"

	"github.com/jackc/pgx/v5"
)

// EvolutionRow é a evolução de uma criança na turma.
type EvolutionRow struct {
	EnrollmentID    string  `json:"enrollment_id"`
	MemberID        string  `json:"member_id"`
	MemberName      string  `json:"member_name"`
	BirthDate       *string `json:"birth_date,omitempty"`
	Attended        int     `json:"attended"`
	TotalSessions   int     `json:"total_sessions"`
	AttendancePct   float64 `json:"attendance_pct"`
	LessonsAttended int     `json:"lessons_attended"`
	TotalLessons    int     `json:"total_lessons"`
	ProgressPct     float64 `json:"progress_pct"`
	LastLesson      string  `json:"last_lesson"`
}

// ClassEvolution é o relatório de evolução de uma turma.
type ClassEvolution struct {
	ClassID       string         `json:"class_id"`
	ClassName     string         `json:"class_name"`
	TrackName     string         `json:"track_name"`
	TotalSessions int            `json:"total_sessions"`
	TotalLessons  int            `json:"total_lessons"`
	Rows          []EvolutionRow `json:"rows"`
}

// Evolution monta o relatório de evolução da turma no período [from, to].
func (r *Repo) Evolution(ctx context.Context, tx pgx.Tx, classID, from, to string) (*ClassEvolution, error) {
	class, err := r.GetClass(ctx, tx, classID)
	if err != nil {
		return nil, err
	}
	out := &ClassEvolution{ClassID: class.ID, ClassName: class.Name, TrackName: class.TrackName}

	if class.TrackID != "" {
		_ = tx.QueryRow(ctx, `SELECT count(*) FROM kids_lessons WHERE track_id=$1::uuid`, class.TrackID).
			Scan(&out.TotalLessons)
	}
	_ = tx.QueryRow(ctx, `
		SELECT count(*) FROM kids_sessions
		WHERE class_id=$1::uuid
		  AND ($2='' OR starts_at >= $2::timestamptz)
		  AND ($3='' OR starts_at <= $3::timestamptz)`, classID, from, to).
		Scan(&out.TotalSessions)

	rows, err := tx.Query(ctx, `
		SELECT e.id::text, e.member_id::text, m.full_name, m.birth_date::text,
		       COUNT(k.id) FILTER (WHERE k.status='present' AND s.id IS NOT NULL) AS attended,
		       COUNT(DISTINCT s.lesson_id) FILTER (WHERE k.status='present' AND s.id IS NOT NULL) AS lessons_attended,
		       COALESCE((SELECT title FROM kids_lessons
		                 WHERE id = (SELECT s2.lesson_id FROM kids_checkins k2
		                             JOIN kids_sessions s2 ON s2.id=k2.session_id
		                             WHERE k2.enrollment_id=e.id AND k2.status='present'
		                               AND s2.lesson_id IS NOT NULL AND s2.id IS NOT NULL
		                             ORDER BY s2.starts_at DESC LIMIT 1)), '') AS last_lesson
		FROM kids_enrollments e
		JOIN members m ON m.id = e.member_id
		LEFT JOIN kids_checkins k ON k.enrollment_id = e.id
		LEFT JOIN kids_sessions s ON s.id = k.session_id
		     AND ($2='' OR s.starts_at >= $2::timestamptz)
		     AND ($3='' OR s.starts_at <= $3::timestamptz)
		WHERE e.class_id=$1::uuid AND e.status='active'
		GROUP BY e.id, m.id
		ORDER BY m.full_name`, classID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out.Rows = []EvolutionRow{}
	for rows.Next() {
		var e EvolutionRow
		if err := rows.Scan(&e.EnrollmentID, &e.MemberID, &e.MemberName, &e.BirthDate,
			&e.Attended, &e.LessonsAttended, &e.LastLesson); err != nil {
			return nil, err
		}
		e.TotalSessions = out.TotalSessions
		e.TotalLessons = out.TotalLessons
		if out.TotalSessions > 0 {
			e.AttendancePct = float64(e.Attended) / float64(out.TotalSessions) * 100
		}
		if out.TotalLessons > 0 {
			e.ProgressPct = float64(e.LessonsAttended) / float64(out.TotalLessons) * 100
		}
		out.Rows = append(out.Rows, e)
	}
	return out, rows.Err()
}
