package content

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// LiveSettings e a config de culto ao vivo da igreja (canal do YouTube).
type LiveSettings struct {
	ChannelID  string `json:"youtube_channel_id"`
	LiveURL    string `json:"live_stream_url"`
	EmbedURL   string `json:"embed_url"`
	WatchURL   string `json:"watch_url"`
	Configured bool   `json:"configured"`
}

// LiveSettings carrega a config de live do tenant do contexto.
func (r *Repo) LiveSettings(ctx context.Context, tx pgx.Tx) (*LiveSettings, error) {
	var channel, url *string
	err := tx.QueryRow(ctx, `
		SELECT youtube_channel_id, live_stream_url
		FROM tenants WHERE id = current_tenant()`).Scan(&channel, &url)
	if err != nil {
		return nil, err
	}
	return buildLive(deref(channel), deref(url)), nil
}

// SetLiveSettings grava o canal/URL do YouTube do tenant do contexto.
func (r *Repo) SetLiveSettings(ctx context.Context, tx pgx.Tx, channelID, liveURL string) (*LiveSettings, error) {
	_, err := tx.Exec(ctx, `
		UPDATE tenants SET
			youtube_channel_id = NULLIF(btrim($1),''),
			live_stream_url    = NULLIF(btrim($2),''),
			updated_at = now()
		WHERE id = current_tenant()`, channelID, liveURL)
	if err != nil {
		return nil, err
	}
	return r.LiveSettings(ctx, tx)
}

func buildLive(channelID, liveURL string) *LiveSettings {
	s := &LiveSettings{ChannelID: channelID, LiveURL: liveURL}
	if channelID != "" {
		s.EmbedURL = fmt.Sprintf("https://www.youtube.com/embed/live_stream?channel=%s", channelID)
		s.WatchURL = fmt.Sprintf("https://www.youtube.com/channel/%s/live", channelID)
	} else if liveURL != "" {
		s.WatchURL = liveURL
		// Sem channel id nao ha embed automatico confiavel; o app abre o link.
	}
	if liveURL != "" {
		s.WatchURL = liveURL
	}
	s.Configured = channelID != "" || liveURL != ""
	return s
}

// ---- Config da Biblia ----

// BibleSettings guarda a traducao padrao do leitor.
type BibleSettings struct {
	VersionID string `json:"bible_version_id"`
}

// BibleSettings carrega a versao padrao da Biblia do tenant.
func (r *Repo) BibleSettings(ctx context.Context, tx pgx.Tx) (*BibleSettings, error) {
	var id *string
	if err := tx.QueryRow(ctx, `SELECT bible_version_id FROM tenants WHERE id = current_tenant()`).Scan(&id); err != nil {
		return nil, err
	}
	return &BibleSettings{VersionID: deref(id)}, nil
}

// SetBibleVersion grava a versao padrao da Biblia do tenant.
func (r *Repo) SetBibleVersion(ctx context.Context, tx pgx.Tx, versionID string) (*BibleSettings, error) {
	if _, err := tx.Exec(ctx, `
		UPDATE tenants SET bible_version_id = NULLIF(btrim($1),''), updated_at = now()
		WHERE id = current_tenant()`, versionID); err != nil {
		return nil, err
	}
	return r.BibleSettings(ctx, tx)
}
