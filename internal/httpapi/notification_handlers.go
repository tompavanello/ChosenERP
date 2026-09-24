package httpapi

import (
	"net/http"

	"github.com/jackc/pgx/v5"

	"chosenerp/internal/announcements"
	"chosenerp/internal/delivery"
)

// handlePreviewAudience devolve quantos destinatários um filtro de segmentação
// alcança (#32) — o "pré-visualizar público" antes de disparar.
func (a *App) handlePreviewAudience(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	var in announcements.SendInput
	if err := readJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	b := boundsFromClaims(claims)
	var count int
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		var err error
		count, err = a.Announcements.CountRecipients(r.Context(), tx, claims.TenantID, in)
		return err
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"count": count})
}

// handleGetNotificationSettings lê as automações de WhatsApp do tenant (#31).
func (a *App) handleGetNotificationSettings(w http.ResponseWriter, r *http.Request) {
	claims, ok := a.adminClaims(w, r)
	if !ok {
		return
	}
	b := boundsFromClaims(claims)
	var s *announcements.NotificationSettings
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		var err error
		s, err = a.Announcements.GetNotificationSettings(r.Context(), tx, claims.TenantID)
		return err
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, s)
}

// handleUpdateNotificationSettings atualiza as automações (PATCH parcial).
func (a *App) handleUpdateNotificationSettings(w http.ResponseWriter, r *http.Request) {
	claims, ok := a.adminClaims(w, r)
	if !ok {
		return
	}
	var in announcements.NotificationSettingsInput
	if err := readJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	b := boundsFromClaims(claims)
	var s *announcements.NotificationSettings
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		var err error
		s, err = a.Announcements.UpsertNotificationSettings(r.Context(), tx, claims.TenantID, in)
		return err
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, s)
}

// handleRunNotifications força uma varredura das automações agora (em vez de
// esperar o tick do worker) — útil para testar a configuração.
func (a *App) handleRunNotifications(w http.ResponseWriter, r *http.Request) {
	if _, ok := a.adminClaims(w, r); !ok {
		return
	}
	worker := &delivery.NotificationWorker{
		Store:         a.Store,
		Announcements: a.Announcements,
		Dispatcher:    a.Dispatch,
	}
	n, err := worker.ProcessOnce(r.Context())
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"queued": n})
}
