package governance

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

// Erros de domínio da governança, traduzidos em HTTP pelos handlers.
var (
	ErrMinuteSigned  = errors.New("ata assinada não pode ser alterada")
	ErrVoteNotOpen   = errors.New("votação não está aberta")
	ErrAlreadyVoted  = errors.New("você já votou nesta votação")
	ErrOptionInvalid = errors.New("opção inválida para esta votação")
	ErrVoteClosed    = errors.New("votação já encerrada")
)

// ---- Tipos ----

// Option é uma opção de voto.
type Option struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Order int    `json:"sort_order"`
	Votes int    `json:"votes"`
}

// Vote é uma votação.
type Vote struct {
	ID               string     `json:"id"`
	BranchID         string     `json:"branch_id"`
	MinuteID         *string    `json:"minute_id,omitempty"`
	MinuteTitle      *string    `json:"minute_title,omitempty"`
	Title            string     `json:"title"`
	Description      *string    `json:"description,omitempty"`
	Kind             string     `json:"kind"`
	Secret           bool       `json:"secret"`
	QuorumRequired   int        `json:"quorum_required"`
	MinAttendance    int        `json:"min_attendance"`
	OpensAt          *time.Time `json:"opens_at,omitempty"`
	ClosesAt         *time.Time `json:"closes_at,omitempty"`
	Status           string     `json:"status"`
	BallotCount      int        `json:"ballot_count"`
	ParticipantCount int        `json:"participant_count"`
	QuorumMet        bool       `json:"quorum_met"`
	Options          []Option   `json:"options,omitempty"`
	Result           *Result    `json:"result,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

// Result é a apuração de uma votação.
type Result struct {
	Total          int      `json:"total"`
	Participants   int      `json:"participants"`
	QuorumRequired int      `json:"quorum_required"`
	QuorumMet      bool     `json:"quorum_met"`
	Winner         string   `json:"winner,omitempty"`
	Options        []Option `json:"options"`
	GeneratedAt    string   `json:"generated_at"`
}

// VoteInput é o corpo de criação/edição de uma votação.
type VoteInput struct {
	MinuteID       *string  `json:"minute_id"`
	Title          string   `json:"title"`
	Description    *string  `json:"description"`
	Kind           *string  `json:"kind"`
	Secret         *bool    `json:"secret"`
	QuorumRequired *int     `json:"quorum_required"`
	MinAttendance  *int     `json:"min_attendance"`
	OpensAt        *string  `json:"opens_at"`
	ClosesAt       *string  `json:"closes_at"`
	Options        []string `json:"options"`
}

const voteCols = `v.id::text, v.branch_id::text, v.minute_id::text, mi.title, v.title, v.description,
	v.kind, v.secret, v.quorum_required, v.min_attendance, v.opens_at, v.closes_at, v.status,
	(SELECT count(*) FROM vote_ballots b WHERE b.vote_id = v.id)::int,
	(SELECT count(*) FROM vote_registrations r WHERE r.vote_id = v.id)::int,
	v.created_at, v.updated_at`

func scanVote(row pgx.Row) (*Vote, error) {
	var v Vote
	err := row.Scan(&v.ID, &v.BranchID, &v.MinuteID, &v.MinuteTitle, &v.Title, &v.Description,
		&v.Kind, &v.Secret, &v.QuorumRequired, &v.MinAttendance, &v.OpensAt, &v.ClosesAt, &v.Status,
		&v.BallotCount, &v.ParticipantCount, &v.CreatedAt, &v.UpdatedAt)
	if err != nil {
		return nil, err
	}
	v.QuorumMet = v.QuorumRequired == 0 || v.ParticipantCount >= v.QuorumRequired
	return &v, nil
}

// ---- Leitura ----

func (r *Repo) ListVotes(ctx context.Context, tx pgx.Tx, status string) ([]Vote, error) {
	rows, err := tx.Query(ctx, `
		SELECT `+voteCols+`
		FROM votes v
		LEFT JOIN minutes mi ON mi.id = v.minute_id
		WHERE ($1 = '' OR v.status = $1)
		ORDER BY v.created_at DESC LIMIT 300`, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Vote{}
	for rows.Next() {
		v, err := scanVote(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *v)
	}
	return out, rows.Err()
}

func (r *Repo) GetVote(ctx context.Context, tx pgx.Tx, id string) (*Vote, error) {
	v, err := scanVote(tx.QueryRow(ctx, `
		SELECT `+voteCols+`
		FROM votes v
		LEFT JOIN minutes mi ON mi.id = v.minute_id
		WHERE v.id = $1::uuid`, id))
	if err != nil {
		return nil, err
	}
	if v.Options, err = r.listOptions(ctx, tx, id); err != nil {
		return nil, err
	}
	return v, nil
}

func (r *Repo) listOptions(ctx context.Context, tx pgx.Tx, voteID string) ([]Option, error) {
	rows, err := tx.Query(ctx, `
		SELECT o.id::text, o.label, o.sort_order,
		       (SELECT count(*) FROM vote_ballots b WHERE b.option_id = o.id)::int
		FROM vote_options o WHERE o.vote_id = $1::uuid ORDER BY o.sort_order, o.label`, voteID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Option{}
	for rows.Next() {
		var o Option
		if err := rows.Scan(&o.ID, &o.Label, &o.Order, &o.Votes); err != nil {
			return nil, err
		}
		out = append(out, o)
	}
	return out, rows.Err()
}

// ---- Escrita ----

func (r *Repo) CreateVote(ctx context.Context, tx pgx.Tx, tenantID, branchID, actorID string, in VoteInput) (*Vote, error) {
	kind := "assembleia"
	if in.Kind != nil && *in.Kind != "" {
		kind = *in.Kind
	}
	secret := true
	if in.Secret != nil {
		secret = *in.Secret
	}
	opens, err := parseOptionalTime(in.OpensAt)
	if err != nil {
		return nil, err
	}
	closes, err := parseOptionalTime(in.ClosesAt)
	if err != nil {
		return nil, err
	}
	var newID string
	err = tx.QueryRow(ctx, `
		INSERT INTO votes (tenant_id, branch_id, minute_id, title, description, kind, secret, quorum_required, min_attendance, opens_at, closes_at, created_by)
		VALUES ($1, NULLIF($2,'')::uuid, NULLIF($3,'')::uuid, $4, $5, $6, $7, COALESCE($8,0), COALESCE($9,0), $10, $11, $12::uuid)
		RETURNING id::text`,
		tenantID, branchID, str(in.MinuteID), in.Title, in.Description, kind, secret,
		in.QuorumRequired, in.MinAttendance, opens, closes, actorID).Scan(&newID)
	if err != nil {
		return nil, err
	}
	if err := r.replaceOptions(ctx, tx, newID, in.Options); err != nil {
		return nil, err
	}
	return r.GetVote(ctx, tx, newID)
}

func (r *Repo) UpdateVote(ctx context.Context, tx pgx.Tx, id string, in VoteInput) (*Vote, error) {
	var status string
	if err := tx.QueryRow(ctx, `SELECT status FROM votes WHERE id = $1::uuid`, id).Scan(&status); err != nil {
		return nil, err
	}
	if status == "encerrada" || status == "cancelada" {
		return nil, ErrVoteClosed
	}
	opens, err := parseOptionalTime(in.OpensAt)
	if err != nil {
		return nil, err
	}
	closes, err := parseOptionalTime(in.ClosesAt)
	if err != nil {
		return nil, err
	}
	var updatedID string
	err = tx.QueryRow(ctx, `
		UPDATE votes SET
			minute_id = CASE WHEN $2::boolean THEN NULLIF($3,'')::uuid ELSE minute_id END,
			title = COALESCE(NULLIF($4,''), title),
			description = COALESCE($5, description),
			kind = COALESCE($6, kind),
			secret = COALESCE($7::boolean, secret),
			quorum_required = COALESCE($8, quorum_required),
			min_attendance = COALESCE($9, min_attendance),
			opens_at = COALESCE($10::timestamptz, opens_at),
			closes_at = COALESCE($11::timestamptz, closes_at),
			updated_at = now()
		WHERE id = $1::uuid
		RETURNING id::text`,
		id, in.MinuteID != nil, str(in.MinuteID), in.Title, in.Description, in.Kind, in.Secret,
		in.QuorumRequired, in.MinAttendance, opens, closes).Scan(&updatedID)
	if err != nil {
		return nil, err
	}
	if in.Options != nil {
		if err := r.replaceOptions(ctx, tx, updatedID, in.Options); err != nil {
			return nil, err
		}
	}
	return r.GetVote(ctx, tx, updatedID)
}

// replaceOptions troca as opções de uma votação (só permitido antes de haver
// votos).
func (r *Repo) replaceOptions(ctx context.Context, tx pgx.Tx, voteID string, labels []string) error {
	var ballots int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM vote_ballots WHERE vote_id = $1::uuid`, voteID).Scan(&ballots); err != nil {
		return err
	}
	if ballots > 0 {
		return errors.New("não é possível alterar as opções de uma votação que já recebeu votos")
	}
	if _, err := tx.Exec(ctx, `DELETE FROM vote_options WHERE vote_id = $1::uuid`, voteID); err != nil {
		return err
	}
	for i, label := range labels {
		label = strings.TrimSpace(label)
		if label == "" {
			continue
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO vote_options (tenant_id, branch_id, vote_id, label, sort_order)
			SELECT v.tenant_id, v.branch_id, v.id, $2, $3 FROM votes v WHERE v.id = $1::uuid`,
			voteID, label, i); err != nil {
			return err
		}
	}
	return nil
}

// OpenVote coloca a votação em andamento.
func (r *Repo) OpenVote(ctx context.Context, tx pgx.Tx, id string) (*Vote, error) {
	tag, err := tx.Exec(ctx, `
		UPDATE votes SET status = 'aberta', opens_at = COALESCE(opens_at, now()), updated_at = now()
		WHERE id = $1::uuid AND status = 'rascunho'`, id)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrVoteClosed
	}
	return r.GetVote(ctx, tx, id)
}

// CastBallot registra a participação (para o quórum) e o voto secreto. Recusa
// voto repetido e votação fora da janela.
func (r *Repo) CastBallot(ctx context.Context, tx pgx.Tx, id, voterID, voterName, optionID string) (*Result, error) {
	v, err := r.GetVote(ctx, tx, id)
	if err != nil {
		return nil, err
	}
	if v.Status != "aberta" {
		return nil, ErrVoteNotOpen
	}
	now := time.Now()
	if v.OpensAt != nil && now.Before(*v.OpensAt) {
		return nil, ErrVoteNotOpen
	}
	if v.ClosesAt != nil && now.After(*v.ClosesAt) {
		return nil, ErrVoteNotOpen
	}
	// A opção precisa pertencer a esta votação.
	var ok bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM vote_options WHERE id = $1::uuid AND vote_id = $2::uuid)`, optionID, id).Scan(&ok); err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrOptionInvalid
	}
	// Participação: uma por usuário por votação.
	tag, err := tx.Exec(ctx, `
		INSERT INTO vote_registrations (tenant_id, branch_id, vote_id, voter_id, voter_name)
		SELECT v.tenant_id, v.branch_id, v.id, $2::uuid, $3 FROM votes v WHERE v.id = $1::uuid
		ON CONFLICT (vote_id, voter_id) DO NOTHING`, id, voterID, voterName)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrAlreadyVoted
	}
	// Escolha: sem vínculo com o eleitor (voto secreto).
	if _, err := tx.Exec(ctx, `
		INSERT INTO vote_ballots (tenant_id, branch_id, vote_id, option_id)
		SELECT v.tenant_id, v.branch_id, v.id, $2::uuid FROM votes v WHERE v.id = $1::uuid`,
		id, optionID); err != nil {
		return nil, err
	}
	return r.Tally(ctx, tx, id)
}

// CloseVote encerra a votação, grava a apuração em result_summary e, se houver
// ata vinculada, anexa o resultado a ela (ata automática — G3).
func (r *Repo) CloseVote(ctx context.Context, tx pgx.Tx, id string) (*Vote, error) {
	var status, minuteID string
	var minutePtr *string
	if err := tx.QueryRow(ctx, `SELECT status, minute_id::text FROM votes WHERE id = $1::uuid`, id).Scan(&status, &minutePtr); err != nil {
		return nil, err
	}
	if status == "encerrada" {
		return r.GetVote(ctx, tx, id)
	}
	if status == "cancelada" {
		return nil, ErrVoteClosed
	}
	result, err := r.Tally(ctx, tx, id)
	if err != nil {
		return nil, err
	}
	summary := tallySummary(result)
	if _, err := tx.Exec(ctx, `
		UPDATE votes SET status = 'encerrada', closes_at = COALESCE(closes_at, now()),
			result_summary = $2, updated_at = now()
		WHERE id = $1::uuid`, id, summary); err != nil {
		return nil, err
	}
	if minutePtr != nil {
		minuteID = *minutePtr
		block := "\n\n---\n\n### Apuração — " + result.Winner + "\n"
		for _, o := range result.Options {
			block += fmt.Sprintf("- %s: %d voto(s)\n", o.Label, o.Votes)
		}
		block += fmt.Sprintf("\nTotal: %d voto(s) · Participantes: %d · Quórum exigido: %d · Quórum atingido: %s\n",
			result.Total, result.Participants, result.QuorumRequired, yesNo(result.QuorumMet))
		if err := r.appendMinuteResult(ctx, tx, minuteID, block); err != nil {
			return nil, err
		}
	}
	return r.GetVote(ctx, tx, id)
}

// Tally apura a votação sem alterar o estado (usado também na prévia).
func (r *Repo) Tally(ctx context.Context, tx pgx.Tx, id string) (*Result, error) {
	v, err := r.GetVote(ctx, tx, id)
	if err != nil {
		return nil, err
	}
	opts := make([]Option, len(v.Options))
	copy(opts, v.Options)
	sort.SliceStable(opts, func(i, j int) bool { return opts[i].Votes > opts[j].Votes })
	res := &Result{
		Total:          v.BallotCount,
		Participants:   v.ParticipantCount,
		QuorumRequired: v.QuorumRequired,
		QuorumMet:      v.QuorumRequired == 0 || v.ParticipantCount >= v.QuorumRequired,
		Options:        opts,
		GeneratedAt:    time.Now().Format(time.RFC3339),
	}
	if len(opts) > 0 && opts[0].Votes > 0 {
		res.Winner = opts[0].Label
	}
	return res, nil
}

// DeleteVote remove uma votação em rascunho (com opções por cascade).
func (r *Repo) DeleteVote(ctx context.Context, tx pgx.Tx, id string) error {
	tag, err := tx.Exec(ctx, `DELETE FROM votes WHERE id = $1::uuid AND status = 'rascunho'`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrVoteClosed
	}
	return nil
}

func tallySummary(res *Result) map[string]any {
	byLabel := make(map[string]int, len(res.Options))
	for _, o := range res.Options {
		byLabel[o.Label] = o.Votes
	}
	return map[string]any{
		"total":           res.Total,
		"participants":    res.Participants,
		"quorum_required": res.QuorumRequired,
		"quorum_met":      res.QuorumMet,
		"winner":          res.Winner,
		"options":         byLabel,
		"generated_at":    res.GeneratedAt,
	}
}

func yesNo(b bool) string {
	if b {
		return "sim"
	}
	return "não"
}

func parseOptionalTime(s *string) (*time.Time, error) {
	if s == nil || *s == "" {
		return nil, nil
	}
	t, err := parseTime(*s)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func str(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
