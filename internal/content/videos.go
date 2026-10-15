package content

import (
	"context"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

// Repo agrega as operacoes de conteudo (devocionais, videos, live).
type Repo struct{}

// ---- Videos (YouTube) ----

// Video e um video publicado para a igreja/filial.
type Video struct {
	ID           string    `json:"id"`
	BranchID     *string   `json:"branch_id,omitempty"`
	Title        string    `json:"title"`
	Description  *string   `json:"description,omitempty"`
	YouTubeID    string    `json:"youtube_id"`
	Kind         string    `json:"kind"`
	Speaker      *string   `json:"speaker,omitempty"`
	RecordedAt   *string   `json:"recorded_at,omitempty"`
	ThumbnailURL *string   `json:"thumbnail_url,omitempty"`
	IsPublished  bool      `json:"is_published"`
	SortOrder    int       `json:"sort_order"`
	CreatedAt    time.Time `json:"created_at"`
}

// VideoInput e o corpo de criacao/edicao de um video.
type VideoInput struct {
	Title        string  `json:"title"`
	Description  *string `json:"description"`
	URL          string  `json:"url"`
	YouTubeID    *string `json:"youtube_id"`
	Kind         string  `json:"kind"`
	Speaker      *string `json:"speaker"`
	RecordedAt   *string `json:"recorded_at"`
	ThumbnailURL *string `json:"thumbnail_url"`
	IsPublished  *bool   `json:"is_published"`
	SortOrder    *int    `json:"sort_order"`
	BranchID     *string `json:"branch_id"`
}

const videoCols = `v.id::text, v.branch_id::text, v.title, v.description, v.youtube_id, v.kind,
	v.speaker, v.recorded_at::text, v.thumbnail_url, v.is_published, v.sort_order, v.created_at`

func scanVideo(row pgx.Row) (*Video, error) {
	var v Video
	err := row.Scan(&v.ID, &v.BranchID, &v.Title, &v.Description, &v.YouTubeID, &v.Kind,
		&v.Speaker, &v.RecordedAt, &v.ThumbnailURL, &v.IsPublished, &v.SortOrder, &v.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

// ListVideos lista os videos do escopo. publishedOnly restringe aos publicados.
func (r *Repo) ListVideos(ctx context.Context, tx pgx.Tx, publishedOnly bool) ([]Video, error) {
	rows, err := tx.Query(ctx, `
		SELECT `+videoCols+`
		FROM content_videos v
		WHERE (NOT $1 OR v.is_published)
		ORDER BY v.sort_order, v.recorded_at DESC NULLS LAST, v.created_at DESC`, publishedOnly)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Video{}
	for rows.Next() {
		v, err := scanVideo(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *v)
	}
	return out, rows.Err()
}

// GetVideo carrega um video pelo id.
func (r *Repo) GetVideo(ctx context.Context, tx pgx.Tx, id string) (*Video, error) {
	return scanVideo(tx.QueryRow(ctx, `SELECT `+videoCols+` FROM content_videos v WHERE v.id = $1::uuid`, id))
}

// CreateVideo insere um video.
func (r *Repo) CreateVideo(ctx context.Context, tx pgx.Tx, userID string, in VideoInput) (*Video, error) {
	kind := in.Kind
	if !validVideoKind(kind) {
		kind = "sermao"
	}
	published := true
	if in.IsPublished != nil {
		published = *in.IsPublished
	}
	sortOrder := 0
	if in.SortOrder != nil {
		sortOrder = *in.SortOrder
	}
	youtubeID := resolveYouTubeID(in)
	var id string
	err := tx.QueryRow(ctx, `
		INSERT INTO content_videos
			(tenant_id, branch_id, title, description, youtube_id, kind, speaker, recorded_at,
			 thumbnail_url, is_published, sort_order, created_by)
		VALUES (current_tenant(), NULLIF($1,'')::uuid, $2, $3, $4, $5, $6,
		        NULLIF($7,'')::date, $8, $9, $10, NULLIF($11,'')::uuid)
		RETURNING id::text`,
		deref(in.BranchID), in.Title, in.Description, youtubeID, kind, in.Speaker,
		deref(in.RecordedAt), thumbFor(in.ThumbnailURL, youtubeID), published, sortOrder, userID).Scan(&id)
	if err != nil {
		return nil, err
	}
	return r.GetVideo(ctx, tx, id)
}

// UpdateVideo edita um video.
func (r *Repo) UpdateVideo(ctx context.Context, tx pgx.Tx, id string, in VideoInput) (*Video, error) {
	// URL/id: se o cliente mandou algo, resolve; senao mantem.
	var ytArg *string
	if strings.TrimSpace(in.URL) != "" || deref(in.YouTubeID) != "" {
		y := resolveYouTubeID(in)
		if y != "" {
			ytArg = &y
		}
	}
	var kind *string
	if validVideoKind(in.Kind) {
		kind = &in.Kind
	}
	var thumb *string
	if ytArg != nil {
		thumb = thumbFor(in.ThumbnailURL, *ytArg)
	} else if in.ThumbnailURL != nil {
		thumb = in.ThumbnailURL
	}
	tag, err := tx.Exec(ctx, `
		UPDATE content_videos SET
			title = COALESCE(NULLIF(btrim($2),''), title),
			description = COALESCE($3, description),
			youtube_id = COALESCE($4, youtube_id),
			kind = COALESCE($5, kind),
			speaker = COALESCE($6, speaker),
			recorded_at = COALESCE(NULLIF($7,'')::date, recorded_at),
			thumbnail_url = COALESCE($8, thumbnail_url),
			is_published = COALESCE($9, is_published),
			sort_order = COALESCE($10, sort_order),
			updated_at = now()
		WHERE id = $1::uuid`,
		id, in.Title, in.Description, ytArg, kind, in.Speaker, deref(in.RecordedAt),
		thumb, in.IsPublished, in.SortOrder)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, pgx.ErrNoRows
	}
	return r.GetVideo(ctx, tx, id)
}

// DeleteVideo remove um video.
func (r *Repo) DeleteVideo(ctx context.Context, tx pgx.Tx, id string) error {
	tag, err := tx.Exec(ctx, `DELETE FROM content_videos WHERE id = $1::uuid`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func validVideoKind(k string) bool {
	switch k {
	case "sermao", "estudo", "musica", "testemunho", "outro":
		return true
	}
	return false
}

// resolveYouTubeID aceita o id direto ou uma URL do YouTube.
func resolveYouTubeID(in VideoInput) string {
	if id := strings.TrimSpace(deref(in.YouTubeID)); id != "" {
		return id
	}
	return ExtractYouTubeID(in.URL)
}

func thumbFor(custom *string, youtubeID string) *string {
	if custom != nil && strings.TrimSpace(*custom) != "" {
		return custom
	}
	if youtubeID == "" {
		return nil
	}
	t := "https://img.youtube.com/vi/" + youtubeID + "/hqdefault.jpg"
	return &t
}

var ytIDRe = regexp.MustCompile(`^[A-Za-z0-9_-]{6,20}$`)

// ExtractYouTubeID extrai o id de um video a partir de uma URL do YouTube
// (watch?v=, youtu.be/, /embed/, /shorts/) ou devolve o proprio valor se ja for
// um id valido.
func ExtractYouTubeID(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return ""
	}
	if ytIDRe.MatchString(s) {
		return s
	}
	if i := strings.Index(s, "youtu.be/"); i >= 0 {
		return ytIDRe.FindString(s[i+len("youtu.be/"):])
	}
	if i := strings.Index(s, "v="); i >= 0 {
		id := s[i+2:]
		if j := strings.IndexAny(id, "&#?"); j >= 0 {
			id = id[:j]
		}
		if ytIDRe.MatchString(id) {
			return id
		}
	}
	for _, seg := range []string{"/embed/", "/shorts/", "/v/"} {
		if i := strings.Index(s, seg); i >= 0 {
			return ytIDRe.FindString(s[i+len(seg):])
		}
	}
	return ""
}
