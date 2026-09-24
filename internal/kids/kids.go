package kids

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// ErrInvalidInput marca erros de validacao (HTTP 400).
var ErrInvalidInput = errors.New("invalid input")

type Repo struct{}

// ---------------------------------------------------------------------------
// Trilhas e licoes (conteudo)
// ---------------------------------------------------------------------------

type Track struct {
	ID          string    `json:"id"`
	TenantID    string    `json:"-"`
	BranchID    string    `json:"branch_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	AgeMin      *int      `json:"age_min,omitempty"`
	AgeMax      *int      `json:"age_max,omitempty"`
	IsActive    bool      `json:"is_active"`
	LessonCount int       `json:"lesson_count"`
	CreatedAt   time.Time `json:"created_at"`
}

type TrackInput struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	AgeMin      *int    `json:"age_min"`
	AgeMax      *int    `json:"age_max"`
	IsActive    *bool   `json:"is_active"`
}

const trackSelect = `
	t.id::text, t.tenant_id::text, COALESCE(t.branch_id::text,''), t.name,
	COALESCE(t.description,''), t.age_min, t.age_max, t.is_active,
	(SELECT count(*) FROM kids_lessons l WHERE l.track_id = t.id), t.created_at`

func scanTrack(s rowScanner) (*Track, error) {
	var t Track
	err := s.Scan(&t.ID, &t.TenantID, &t.BranchID, &t.Name, &t.Description,
		&t.AgeMin, &t.AgeMax, &t.IsActive, &t.LessonCount, &t.CreatedAt)
	return &t, err
}

func (r *Repo) ListTracks(ctx context.Context, tx pgx.Tx) ([]Track, error) {
	rows, err := tx.Query(ctx, `SELECT `+trackSelect+` FROM kids_tracks t ORDER BY t.name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Track{}
	for rows.Next() {
		t, err := scanTrack(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *t)
	}
	return out, rows.Err()
}

func (r *Repo) GetTrack(ctx context.Context, tx pgx.Tx, id string) (*Track, error) {
	return scanTrack(tx.QueryRow(ctx, `SELECT `+trackSelect+` FROM kids_tracks t WHERE t.id = $1::uuid`, id))
}

func (r *Repo) CreateTrack(ctx context.Context, tx pgx.Tx, tenantID, branchID string, in TrackInput) (*Track, error) {
	if in.Name == nil || *in.Name == "" {
		return nil, fmt.Errorf("%w: name e obrigatorio", ErrInvalidInput)
	}
	if err := validateAges(in.AgeMin, in.AgeMax); err != nil {
		return nil, err
	}
	var id string
	err := tx.QueryRow(ctx, `
		INSERT INTO kids_tracks (tenant_id, branch_id, name, description, age_min, age_max, is_active)
		VALUES ($1, NULLIF($2,'')::uuid, $3, NULLIF($4,''), $5, $6, COALESCE($7, true))
		RETURNING id::text`,
		tenantID, branchID, *in.Name, strOrEmpty(in.Description), in.AgeMin, in.AgeMax, in.IsActive).Scan(&id)
	if err != nil {
		return nil, err
	}
	return r.GetTrack(ctx, tx, id)
}

func (r *Repo) UpdateTrack(ctx context.Context, tx pgx.Tx, id string, in TrackInput) (*Track, error) {
	t, err := r.GetTrack(ctx, tx, id)
	if err != nil {
		return nil, err
	}
	applyTrack(t, in)
	if err := validateAges(t.AgeMin, t.AgeMax); err != nil {
		return nil, err
	}
	_, err = tx.Exec(ctx, `
		UPDATE kids_tracks SET name=$2, description=NULLIF($3,''), age_min=$4, age_max=$5, is_active=$6
		WHERE id=$1::uuid`, id, t.Name, t.Description, t.AgeMin, t.AgeMax, t.IsActive)
	if err != nil {
		return nil, err
	}
	return r.GetTrack(ctx, tx, id)
}

func (r *Repo) DeleteTrack(ctx context.Context, tx pgx.Tx, id string) error {
	return execExpectRow(ctx, tx, `DELETE FROM kids_tracks WHERE id=$1::uuid`, id)
}

func applyTrack(t *Track, in TrackInput) {
	if in.Name != nil && *in.Name != "" {
		t.Name = *in.Name
	}
	if in.Description != nil {
		t.Description = *in.Description
	}
	if in.AgeMin != nil {
		t.AgeMin = in.AgeMin
	}
	if in.AgeMax != nil {
		t.AgeMax = in.AgeMax
	}
	if in.IsActive != nil {
		t.IsActive = *in.IsActive
	}
}

type Lesson struct {
	ID        string    `json:"id"`
	TrackID   string    `json:"track_id"`
	Position  int       `json:"position"`
	Title     string    `json:"title"`
	Objective string    `json:"objective"`
	Verse     string    `json:"verse"`
	Content   string    `json:"content"`
	Materials string    `json:"materials"`
	CreatedAt time.Time `json:"created_at"`
}

type LessonInput struct {
	Position  *int    `json:"position"`
	Title     *string `json:"title"`
	Objective *string `json:"objective"`
	Verse     *string `json:"verse"`
	Content   *string `json:"content"`
	Materials *string `json:"materials"`
}

const lessonSelect = `
	id::text, track_id::text, position, title, COALESCE(objective,''),
	COALESCE(verse,''), COALESCE(content,''), COALESCE(materials,''), created_at`

func scanLesson(s rowScanner) (*Lesson, error) {
	var l Lesson
	err := s.Scan(&l.ID, &l.TrackID, &l.Position, &l.Title, &l.Objective,
		&l.Verse, &l.Content, &l.Materials, &l.CreatedAt)
	return &l, err
}

func (r *Repo) ListLessons(ctx context.Context, tx pgx.Tx, trackID string) ([]Lesson, error) {
	rows, err := tx.Query(ctx, `SELECT `+lessonSelect+` FROM kids_lessons WHERE track_id=$1::uuid ORDER BY position`, trackID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Lesson{}
	for rows.Next() {
		l, err := scanLesson(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *l)
	}
	return out, rows.Err()
}

func (r *Repo) CreateLesson(ctx context.Context, tx pgx.Tx, tenantID, branchID, trackID string, in LessonInput) (*Lesson, error) {
	if in.Title == nil || *in.Title == "" {
		return nil, fmt.Errorf("%w: title e obrigatorio", ErrInvalidInput)
	}
	position := 1
	if in.Position != nil {
		position = *in.Position
	} else {
		_ = tx.QueryRow(ctx, `SELECT COALESCE(max(position),0)+1 FROM kids_lessons WHERE track_id=$1::uuid`, trackID).Scan(&position)
	}
	return scanLesson(tx.QueryRow(ctx, `
		INSERT INTO kids_lessons (tenant_id, branch_id, track_id, position, title, objective, verse, content, materials)
		VALUES ($1, NULLIF($2,'')::uuid, $3::uuid, $4, $5, NULLIF($6,''), NULLIF($7,''), NULLIF($8,''), NULLIF($9,''))
		RETURNING `+lessonSelect,
		tenantID, branchID, trackID, position, *in.Title,
		strOrEmpty(in.Objective), strOrEmpty(in.Verse), strOrEmpty(in.Content), strOrEmpty(in.Materials)))
}

func (r *Repo) UpdateLesson(ctx context.Context, tx pgx.Tx, id string, in LessonInput) (*Lesson, error) {
	l, err := scanLesson(tx.QueryRow(ctx, `SELECT `+lessonSelect+` FROM kids_lessons WHERE id=$1::uuid`, id))
	if err != nil {
		return nil, err
	}
	if in.Position != nil {
		l.Position = *in.Position
	}
	if in.Title != nil && *in.Title != "" {
		l.Title = *in.Title
	}
	if in.Objective != nil {
		l.Objective = *in.Objective
	}
	if in.Verse != nil {
		l.Verse = *in.Verse
	}
	if in.Content != nil {
		l.Content = *in.Content
	}
	if in.Materials != nil {
		l.Materials = *in.Materials
	}
	return scanLesson(tx.QueryRow(ctx, `
		UPDATE kids_lessons SET position=$2, title=$3, objective=NULLIF($4,''),
		       verse=NULLIF($5,''), content=NULLIF($6,''), materials=NULLIF($7,'')
		WHERE id=$1::uuid RETURNING `+lessonSelect,
		id, l.Position, l.Title, l.Objective, l.Verse, l.Content, l.Materials))
}

func (r *Repo) DeleteLesson(ctx context.Context, tx pgx.Tx, id string) error {
	return execExpectRow(ctx, tx, `DELETE FROM kids_lessons WHERE id=$1::uuid`, id)
}

// ---------------------------------------------------------------------------
// Turmas
// ---------------------------------------------------------------------------

type Class struct {
	ID          string    `json:"id"`
	TenantID    string    `json:"-"`
	BranchID    string    `json:"branch_id"`
	Name        string    `json:"name"`
	AgeMin      *int      `json:"age_min,omitempty"`
	AgeMax      *int      `json:"age_max,omitempty"`
	TrackID     string    `json:"track_id"`
	TrackName   string    `json:"track_name"`
	Room        string    `json:"room"`
	LeaderID    string    `json:"leader_member_id"`
	LeaderName  string    `json:"leader_name"`
	IsActive    bool      `json:"is_active"`
	EnrollCount int       `json:"enrollment_count"`
	CreatedAt   time.Time `json:"created_at"`
}

type ClassInput struct {
	Name     *string `json:"name"`
	AgeMin   *int    `json:"age_min"`
	AgeMax   *int    `json:"age_max"`
	TrackID  *string `json:"track_id"`
	Room     *string `json:"room"`
	LeaderID *string `json:"leader_member_id"`
	IsActive *bool   `json:"is_active"`
	// BranchID permite que a Sede escolha a filial da turma; sem ele usa-se a
	// filial da sessao. E obrigatorio para quem nao tem filial (Sede).
	BranchID *string `json:"branch_id"`
}

const classSelect = `
	c.id::text, c.tenant_id::text, c.branch_id::text, c.name, c.age_min, c.age_max,
	COALESCE(c.track_id::text,''), COALESCE(tr.name,''), COALESCE(c.room,''),
	COALESCE(c.leader_member_id::text,''), COALESCE(m.full_name,''), c.is_active,
	(SELECT count(*) FROM kids_enrollments e WHERE e.class_id = c.id AND e.status = 'active'),
	c.created_at
	FROM kids_classes c
	LEFT JOIN kids_tracks tr ON tr.id = c.track_id
	LEFT JOIN members m ON m.id = c.leader_member_id`

func scanClass(s rowScanner) (*Class, error) {
	var c Class
	err := s.Scan(&c.ID, &c.TenantID, &c.BranchID, &c.Name, &c.AgeMin, &c.AgeMax,
		&c.TrackID, &c.TrackName, &c.Room, &c.LeaderID, &c.LeaderName, &c.IsActive,
		&c.EnrollCount, &c.CreatedAt)
	return &c, err
}

func (r *Repo) ListClasses(ctx context.Context, tx pgx.Tx) ([]Class, error) {
	rows, err := tx.Query(ctx, `SELECT `+classSelect+` ORDER BY c.name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Class{}
	for rows.Next() {
		c, err := scanClass(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *c)
	}
	return out, rows.Err()
}

func (r *Repo) GetClass(ctx context.Context, tx pgx.Tx, id string) (*Class, error) {
	return scanClass(tx.QueryRow(ctx, `SELECT `+classSelect+` WHERE c.id=$1::uuid`, id))
}

func (r *Repo) CreateClass(ctx context.Context, tx pgx.Tx, tenantID, branchID string, in ClassInput) (*Class, error) {
	if in.Name == nil || *in.Name == "" {
		return nil, fmt.Errorf("%w: name e obrigatorio", ErrInvalidInput)
	}
	if err := validateAges(in.AgeMin, in.AgeMax); err != nil {
		return nil, err
	}
	branch := branchID
	if in.BranchID != nil && *in.BranchID != "" {
		branch = *in.BranchID
	}
	if branch == "" {
		return nil, fmt.Errorf("%w: informe a filial (branch_id) da turma", ErrInvalidInput)
	}
	var id string
	err := tx.QueryRow(ctx, `
		INSERT INTO kids_classes (tenant_id, branch_id, name, age_min, age_max, track_id, room, leader_member_id, is_active)
		VALUES ($1, $2::uuid, $3, $4, $5, NULLIF($6,'')::uuid, NULLIF($7,''), NULLIF($8,'')::uuid, COALESCE($9, true))
		RETURNING id::text`,
		tenantID, branch, *in.Name, in.AgeMin, in.AgeMax,
		strOrEmpty(in.TrackID), strOrEmpty(in.Room), strOrEmpty(in.LeaderID), in.IsActive).Scan(&id)
	if err != nil {
		return nil, err
	}
	return r.GetClass(ctx, tx, id)
}

func applyClass(c *Class, in ClassInput) {
	if in.Name != nil && *in.Name != "" {
		c.Name = *in.Name
	}
	if in.AgeMin != nil {
		c.AgeMin = in.AgeMin
	}
	if in.AgeMax != nil {
		c.AgeMax = in.AgeMax
	}
	if in.TrackID != nil {
		c.TrackID = *in.TrackID
	}
	if in.Room != nil {
		c.Room = *in.Room
	}
	if in.LeaderID != nil {
		c.LeaderID = *in.LeaderID
	}
	if in.IsActive != nil {
		c.IsActive = *in.IsActive
	}
}

func (r *Repo) UpdateClass(ctx context.Context, tx pgx.Tx, id string, in ClassInput) (*Class, error) {
	c, err := r.GetClass(ctx, tx, id)
	if err != nil {
		return nil, err
	}
	applyClass(c, in)
	if err := validateAges(c.AgeMin, c.AgeMax); err != nil {
		return nil, err
	}
	var out string
	err = tx.QueryRow(ctx, `
		UPDATE kids_classes SET name=$2, age_min=$3, age_max=$4, track_id=NULLIF($5,'')::uuid,
		       room=NULLIF($6,''), leader_member_id=NULLIF($7,'')::uuid, is_active=$8
		WHERE id=$1::uuid RETURNING id::text`,
		id, c.Name, c.AgeMin, c.AgeMax, c.TrackID, c.Room, c.LeaderID, c.IsActive).Scan(&out)
	if err != nil {
		return nil, err
	}
	return r.GetClass(ctx, tx, id)
}

func (r *Repo) DeleteClass(ctx context.Context, tx pgx.Tx, id string) error {
	return execExpectRow(ctx, tx, `DELETE FROM kids_classes WHERE id=$1::uuid`, id)
}
