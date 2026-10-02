// Package push envia notificacoes Web Push (VAPID) para as inscricoes do app do
// membro. O payload e cifrado pelo protocolo Web Push (RFC 8291) via
// webpush-go; o service worker (public/sw.js) exibe a notificacao.
package push

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	webpush "github.com/SherClockHolmes/webpush-go"
	"github.com/jackc/pgx/v5"
)

// Config sao as chaves VAPID (base64url) e o contato (sub do JWT VAPID).
type Config struct {
	PublicKey  string
	PrivateKey string
	Subject    string
}

// Message e o conteudo exibido na notificacao.
type Message struct {
	Title string `json:"title"`
	Body  string `json:"body"`
	URL   string `json:"url,omitempty"`
}

// Subscription e uma inscricao de um dispositivo.
type Subscription struct {
	ID       string
	Endpoint string
	P256dh   string
	Auth     string
}

type Repo struct {
	Config Config
	Client *http.Client
}

// Enabled informa se as chaves VAPID estao configuradas.
func (r *Repo) Enabled() bool {
	return r != nil && r.Config.PublicKey != "" && r.Config.PrivateKey != ""
}

// Subscribe cria/atualiza a inscricao do usuario (idempotente pelo endpoint).
func (r *Repo) Subscribe(ctx context.Context, tx pgx.Tx, tenantID, branchID, userID, endpoint, p256dh, auth, userAgent string) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO push_subscriptions (tenant_id, branch_id, user_id, endpoint, p256dh, auth, user_agent)
		VALUES ($1::uuid, NULLIF($2,'')::uuid, $3::uuid, $4, $5, $6, $7)
		ON CONFLICT (endpoint) DO UPDATE SET
			tenant_id = EXCLUDED.tenant_id,
			branch_id = EXCLUDED.branch_id,
			user_id = EXCLUDED.user_id,
			p256dh = EXCLUDED.p256dh,
			auth = EXCLUDED.auth,
			user_agent = EXCLUDED.user_agent,
			last_seen_at = now()`,
		tenantID, branchID, userID, endpoint, p256dh, auth, userAgent)
	return err
}

// Unsubscribe remove a inscricao do usuario.
func (r *Repo) Unsubscribe(ctx context.Context, tx pgx.Tx, userID, endpoint string) error {
	_, err := tx.Exec(ctx, `DELETE FROM push_subscriptions WHERE user_id = $1::uuid AND endpoint = $2`, userID, endpoint)
	return err
}

// SendToTenant envia a mensagem a todas as inscricoes do tenant. Best-effort:
// falhas de rede nao interrompem as demais; inscricoes expiradas (404/410) sao
// removidas. Deve rodar em contexto de sistema (WithSystem), pois le inscricoes
// de outros usuarios.
func (r *Repo) SendToTenant(ctx context.Context, tx pgx.Tx, tenantID string, msg Message) error {
	if !r.Enabled() {
		return nil
	}
	rows, err := tx.Query(ctx, `
		SELECT id::text, endpoint, p256dh, auth
		FROM push_subscriptions
		WHERE tenant_id = $1::uuid`, tenantID)
	if err != nil {
		return err
	}
	var subs []Subscription
	for rows.Next() {
		var s Subscription
		if err := rows.Scan(&s.ID, &s.Endpoint, &s.P256dh, &s.Auth); err != nil {
			rows.Close()
			return err
		}
		subs = append(subs, s)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	if len(subs) == 0 {
		return nil
	}

	payload, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	client := r.Client
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	for _, s := range subs {
		resp, err := webpush.SendNotificationWithContext(ctx, payload, &webpush.Subscription{
			Endpoint: s.Endpoint,
			Keys:     webpush.Keys{P256dh: s.P256dh, Auth: s.Auth},
		}, &webpush.Options{
			Subscriber:      r.Config.Subject,
			TTL:             300,
			VAPIDPublicKey:  r.Config.PublicKey,
			VAPIDPrivateKey: r.Config.PrivateKey,
			HTTPClient:      client,
		})
		if err != nil {
			log.Printf("[push] envio falhou: %v", err)
			continue
		}
		if resp.Body != nil {
			resp.Body.Close()
		}
		if resp.StatusCode == http.StatusGone || resp.StatusCode == http.StatusNotFound {
			_, _ = tx.Exec(ctx, `DELETE FROM push_subscriptions WHERE id = $1::uuid`, s.ID)
		}
	}
	return nil
}
