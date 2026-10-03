package httpapi

import (
	"context"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"chosenerp/internal/announcements"
	"chosenerp/internal/push"
)

// handlePushPublicKey devolve a chave publica VAPID para o navegador assinar.
func (a *App) handlePushPublicKey(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"public_key": a.Push.Config.PublicKey,
		"enabled":    a.Push.Enabled(),
	})
}

// handleMePushSubscribe registra a inscricao do dispositivo do usuario.
func (a *App) handleMePushSubscribe(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	var in struct {
		Endpoint string `json:"endpoint"`
		Keys     struct {
			P256dh string `json:"p256dh"`
			Auth   string `json:"auth"`
		} `json:"keys"`
	}
	if err := readJSON(r, &in); err != nil || in.Endpoint == "" || in.Keys.P256dh == "" || in.Keys.Auth == "" {
		writeErr(w, http.StatusBadRequest, "inscricao invalida")
		return
	}
	err := a.Store.WithTenant(r.Context(), boundsFromClaims(claims), func(tx pgx.Tx) error {
		return a.Push.Subscribe(r.Context(), tx, claims.TenantID, claims.BranchID, claims.UserID,
			in.Endpoint, in.Keys.P256dh, in.Keys.Auth, r.UserAgent())
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// handleMePushUnsubscribe remove a inscricao do dispositivo.
func (a *App) handleMePushUnsubscribe(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	var in struct {
		Endpoint string `json:"endpoint"`
	}
	if err := readJSON(r, &in); err != nil || in.Endpoint == "" {
		writeErr(w, http.StatusBadRequest, "endpoint required")
		return
	}
	err := a.Store.WithTenant(r.Context(), boundsFromClaims(claims), func(tx pgx.Tx) error {
		return a.Push.Unsubscribe(r.Context(), tx, claims.UserID, in.Endpoint)
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// pushToAudienceAsync dispara o push (best-effort) apenas para os membros
// alcancados pelo publico/filtro do comunicado, sem bloquear a resposta.
func (a *App) pushToAudienceAsync(tenantID, title, body, audience string, filter announcements.AudienceFilter) {
	if !a.Push.Enabled() || tenantID == "" || strings.TrimSpace(title) == "" {
		return
	}
	msg := push.Message{Title: title, Body: truncateText(body, 140), URL: "/avisos"}
	in := announcements.SendInput{Audience: audience, AudienceFilter: filter}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := a.Store.WithSystem(ctx, func(tx pgx.Tx) error {
			ids, err := a.Announcements.ResolveMemberIDs(ctx, tx, tenantID, in)
			if err != nil {
				return err
			}
			return a.Push.SendToMembers(ctx, tx, tenantID, ids, msg)
		}); err != nil {
			log.Printf("[push] tenant %s: %v", tenantID, err)
		}
	}()
}

func truncateText(s string, max int) string {
	s = strings.TrimSpace(s)
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return strings.TrimSpace(string(r[:max])) + "..."
}
