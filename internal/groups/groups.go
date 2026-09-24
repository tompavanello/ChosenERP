package groups

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5"
)

// SmallGroup é uma célula/EBD/grupo familiar.
type SmallGroup struct {
	ID          string    `json:"id"`
	BranchID    string    `json:"branch_id"`
	MinistryID  *string   `json:"ministry_id,omitempty"`
	Name        string    `json:"name"`
	Kind        string    `json:"kind"`
	LeaderID    *string   `json:"leader_id,omitempty"`
	LeaderName  *string   `json:"leader_name,omitempty"`
	Address     string    `json:"address,omitempty"`
	MaxMembers  *int      `json:"max_members,omitempty"`
	Weekday     *int      `json:"weekday,omitempty"`
	MeetingTime *string   `json:"meeting_time,omitempty"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
}

// AttendanceCheckin agrega os dados de check-in infantil/frequência.
type AttendanceCheckin struct {
	GroupID    string  `json:"group_id"`
	GroupName  string  `json:"group_name"`
	MemberID   *string `json:"member_id,omitempty"`
	MemberName string  `json:"member_name"`
	AttendedAt string  `json:"attended_at"`
	Present    bool    `json:"present"`
}

type CreateInput struct {
	Name        string  `json:"name"`
	Kind        string  `json:"kind"`
	MinistryID  *string `json:"ministry_id"`
	LeaderID    *string `json:"leader_id"`
	Address     *string `json:"address"`
	MaxMembers  *int    `json:"max_members"`
	Weekday     *int    `json:"weekday"`
	MeetingTime *string `json:"meeting_time"`
}

type UpdateInput struct {
	Name        *string `json:"name"`
	Kind        *string `json:"kind"`
	MinistryID  *string `json:"ministry_id"`
	LeaderID    *string `json:"leader_id"`
	Address     *string `json:"address"`
	MaxMembers  *int    `json:"max_members"`
	Weekday     *int    `json:"weekday"`
	MeetingTime *string `json:"meeting_time"`
	IsActive    *bool   `json:"is_active"`
}

// AttendanceInput registra a frequência de um membro num grupo.
type AttendanceInput struct {
	MemberID   *string `json:"member_id"`
	MemberName string  `json:"member_name"`
	Present    bool    `json:"present"`
	AttendedAt *string `json:"attended_at"`
}

type Repo struct{}

// List retorna os grupos do escopo.
func (r *Repo) List(ctx context.Context, tx pgx.Tx) ([]SmallGroup, error) {
	rows, err := tx.Query(ctx, `
		SELECT s.id::text, s.branch_id::text, s.ministry_id::text, s.name, s.kind,
		       s.leader_id::text, lm.full_name, COALESCE(s.address::text,''),
		       s.max_members, s.weekday, s.meeting_time::text, s.is_active, s.created_at
		FROM small_groups s
		LEFT JOIN members lm ON lm.id = s.leader_id
		ORDER BY s.name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []SmallGroup{}
	for rows.Next() {
		var g SmallGroup
		// LEFT JOIN: lm.full_name é NULL quando o grupo não tem líder.
		var leader *string
		if err := rows.Scan(&g.ID, &g.BranchID, &g.MinistryID, &g.Name, &g.Kind,
			&g.LeaderID, &leader, &g.Address, &g.MaxMembers, &g.Weekday, &g.MeetingTime, &g.IsActive, &g.CreatedAt); err != nil {
			return nil, err
		}
		if leader != nil && *leader != "" {
			g.LeaderName = leader
		}
		out = append(out, g)
	}
	return out, rows.Err()
}

// Create insere um grupo no escopo RLS da sessão.
func (r *Repo) Create(ctx context.Context, tx pgx.Tx, tenantID, branchID string, in CreateInput) (*SmallGroup, error) {
	var g SmallGroup
	var addrJSON []byte
	if in.Address != nil && *in.Address != "" {
		addrJSON, _ = json.Marshal(map[string]string{"text": *in.Address})
	}
	err := tx.QueryRow(ctx, `
		INSERT INTO small_groups
			(tenant_id, branch_id, ministry_id, name, kind, leader_id, address, max_members, weekday, meeting_time)
		VALUES
			($1, $2::uuid, NULLIF($3,'')::uuid, $4, $5, NULLIF($6,'')::uuid, $7, $8, $9, NULLIF($10,'')::time)
		RETURNING id::text, branch_id::text, ministry_id::text, name, kind, leader_id::text,
		          COALESCE(address->>'text',''), max_members, weekday, meeting_time::text, is_active, created_at`,
		tenantID, branchID, nullStr(in.MinistryID), in.Name, in.Kind, nullStr(in.LeaderID),
		addrJSON, in.MaxMembers, in.Weekday, in.MeetingTime).
		Scan(&g.ID, &g.BranchID, &g.MinistryID, &g.Name, &g.Kind,
			&g.LeaderID, &g.Address, &g.MaxMembers, &g.Weekday, &g.MeetingTime, &g.IsActive, &g.CreatedAt)
	return &g, err
}

// Update edita os dados do grupo/célula.
func (r *Repo) Update(ctx context.Context, tx pgx.Tx, id string, in UpdateInput) (*SmallGroup, error) {
	var addrJSON []byte
	if in.Address != nil && *in.Address != "" {
		addrJSON, _ = json.Marshal(map[string]string{"text": *in.Address})
	}
	var g SmallGroup
	err := tx.QueryRow(ctx, `
		UPDATE small_groups SET
			name = COALESCE(NULLIF($2,''), name),
			kind = COALESCE(NULLIF($3,''), kind),
			ministry_id = CASE WHEN $4::boolean THEN NULLIF($5,'')::uuid ELSE ministry_id END,
			leader_id = CASE WHEN $6::boolean THEN NULLIF($7,'')::uuid ELSE leader_id END,
			address = CASE WHEN $8::boolean THEN $9::jsonb ELSE address END,
			max_members = COALESCE($10, max_members),
			weekday = COALESCE($11, weekday),
			meeting_time = COALESCE(NULLIF($12,'')::time, meeting_time),
			is_active = COALESCE($13::boolean, is_active)
		WHERE id = $1::uuid
		RETURNING id::text, branch_id::text, ministry_id::text, name, kind, leader_id::text,
		          COALESCE(address->>'text',''), max_members, weekday, meeting_time::text, is_active, created_at`,
		id, str(in.Name), str(in.Kind), in.MinistryID != nil, str(in.MinistryID),
		in.LeaderID != nil, str(in.LeaderID), in.Address != nil, addrJSON,
		in.MaxMembers, in.Weekday, str(in.MeetingTime), in.IsActive).
		Scan(&g.ID, &g.BranchID, &g.MinistryID, &g.Name, &g.Kind,
			&g.LeaderID, &g.Address, &g.MaxMembers, &g.Weekday, &g.MeetingTime, &g.IsActive, &g.CreatedAt)
	return &g, err
}

// Delete remove um grupo/célula (frequência cai por FK).
func (r *Repo) Delete(ctx context.Context, tx pgx.Tx, id string) error {
	tag, err := tx.Exec(ctx, `DELETE FROM small_groups WHERE id = $1::uuid`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// CheckIn registra a presença (apenas com member_id; anônimos são ignorados no agregado).
func (r *Repo) CheckIn(ctx context.Context, tx pgx.Tx, tenantID, branchID, groupID string, in AttendanceInput) (*AttendanceCheckin, error) {
	when := time.Now()
	if in.AttendedAt != nil && *in.AttendedAt != "" {
		if t, err := time.Parse(time.RFC3339, *in.AttendedAt); err == nil {
			when = t
		}
	}
	var a AttendanceCheckin
	err := tx.QueryRow(ctx, `
		INSERT INTO group_attendance (tenant_id, branch_id, small_group_id, member_id, present, attended_at)
		VALUES ($1, $2::uuid, $3::uuid, NULLIF($4,'')::uuid, $5, $6)
		RETURNING small_group_id::text, member_id::text, attended_at::text, present`,
		tenantID, branchID, groupID, nullStr(in.MemberID), in.Present, when).
		Scan(&a.GroupID, &a.MemberID, &a.AttendedAt, &a.Present)
	if err != nil {
		return nil, err
	}
	a.MemberName = in.MemberName
	return &a, nil
}

// ListAttendance retorna a frequência de um grupo (para relatório/quadro).
func (r *Repo) ListAttendance(ctx context.Context, tx pgx.Tx, groupID string) ([]AttendanceCheckin, error) {
	rows, err := tx.Query(ctx, `
		SELECT ga.small_group_id::text, s.name, ga.member_id::text,
		       COALESCE(m.full_name,'Visitante/Anônimo'), ga.attended_at::text, ga.present
		FROM group_attendance ga
		JOIN small_groups s ON s.id = ga.small_group_id
		LEFT JOIN members m ON m.id = ga.member_id
		WHERE ga.small_group_id = $1::uuid
		ORDER BY ga.attended_at DESC LIMIT 200`, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []AttendanceCheckin{}
	for rows.Next() {
		var a AttendanceCheckin
		if err := rows.Scan(&a.GroupID, &a.GroupName, &a.MemberID, &a.MemberName, &a.AttendedAt, &a.Present); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func nullStr(s *string) *string {
	if s != nil && *s == "" {
		return nil
	}
	return s
}

func str(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
