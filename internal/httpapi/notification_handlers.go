package httpapi

import (
	"net/http"

	"github.com/jackc/pgx/v5"

	"chosenerp/internal/announcements"
	"chosenerp/internal/delivery"
)

// handlePreviewAudience devolve quantos destinatarios um filtro de segmentacao
// alcanca (#32) - o "pre-visualizar publico" antes de disparar.
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

// handleGetNotificationSettings le as automacoes de WhatsApp do tenant (#31).
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

// handleUpdateNotificationSettings atualiza as automacoes (PATCH parcial).
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

// handleRunNotifications forca uma varredura das automacoes agora (em vez de
// esperar o tick do worker) - util para testar a configuracao.
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
