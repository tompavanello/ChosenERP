package httpapi

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/jackc/pgx/v5"

	"chosenerp/internal/announcements"
	"chosenerp/internal/delivery"
	"chosenerp/internal/store"
)

// handleSendAnnouncement resolve o publico-alvo do comunicado e enfileira
// os envios na outbox (announcement_deliveries). O worker de anuncios processa
// os envios de forma assincrona respeitando o escopo RLS de cada filial.
func (a *App) handleSendAnnouncement(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	id := r.PathValue("id")
	var in announcements.SendInput
	if err := readJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}

	b := boundsFromClaims(claims)

	var recipients []announcements.Recipient
	var ann *announcements.Announcement
	var created int
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		var err error
		ann, err = a.Announcements.GetByID(r.Context(), tx, id)
		if err != nil {
			return err
		}
		// Sem escolha explicita no disparo, usa a segmentacao salva no comunicado.
		if in.Audience == "" {
			in.Audience = ann.Audience
		}
		if isZeroFilter(in.AudienceFilter) {
			in.AudienceFilter = ann.AudienceFilter
		}
		if in.Channel == "" {
			in.Channel = ann.Channel
		}
		if in.Channel != delivery.ChannelEmail && in.Channel != delivery.ChannelWhatsApp {
			in.Channel = delivery.ChannelWhatsApp
		}

		recipients, err = a.Announcements.ResolveRecipients(r.Context(), tx, ann.TenantID, in)
		if err != nil {
			return err
		}
		if len(recipients) == 0 {
			return nil
		}
		provider := a.Dispatch.ProviderFor(in.Channel)
		for _, rec := range recipients {
			_, derr := a.Announcements.CreateDelivery(r.Context(), tx,
				ann.TenantID, ann.BranchID, id, in.Channel, provider, rec.Phone)
			if derr != nil {
				return derr
			}
			created++
		}
		return nil
	})
	if err != nil {
		if store.IsNotFound(err) {
			writeErr(w, http.StatusNotFound, "announcement not found")
			return
		}
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"announcement_id": id,
		"channel":         in.Channel,
		"recipient_count": created,
		"status":          "queued",
	})
}

// isZeroFilter indica que a segmentacao enviada nao tem nenhum criterio.
func isZeroFilter(f announcements.AudienceFilter) bool {
	return len(f.GroupIDs) == 0 && len(f.MinistryIDs) == 0 && len(f.BranchIDs) == 0 &&
		len(f.Genders) == 0 && len(f.MaritalStatuses) == 0 && len(f.MembershipStatuses) == 0 &&
		f.AgeMin == nil && f.AgeMax == nil
}

// handleUpdateAnnouncement edita o comunicado: texto, segmentacao e agendamento.
func (a *App) handleUpdateAnnouncement(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	id := r.PathValue("id")
	var in announcements.UpsertInput
	if err := readJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	b := boundsFromClaims(claims)
	var out *announcements.Announcement
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		var err error
		out, err = a.Announcements.Update(r.Context(), tx, id, in)
		return err
	})
	if err != nil {
		switch {
		case store.IsNotFound(err):
			writeErr(w, http.StatusNotFound, "announcement not found")
		case errors.Is(err, announcements.ErrInvalidInput):
			writeErr(w, http.StatusBadRequest, err.Error())
		default:
			writeErr(w, http.StatusInternalServerError, err.Error())
		}
		return
	}
	writeJSON(w, http.StatusOK, out)
}

// handleDeleteAnnouncement exclui o comunicado (entregas e execucoes caem em cascata).
func (a *App) handleDeleteAnnouncement(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	b := boundsFromClaims(claims)
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		return a.Announcements.Delete(r.Context(), tx, r.PathValue("id"))
	})
	if err != nil {
		if store.IsNotFound(err) {
			writeErr(w, http.StatusNotFound, "announcement not found")
			return
		}
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// handleListNotificationRuns lista o historico de execucoes dos agendamentos.
func (a *App) handleListNotificationRuns(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	b := boundsFromClaims(claims)
	var out []announcements.Run
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		var err error
		out, err = a.Announcements.ListRuns(r.Context(), tx, limit)
		return err
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"runs": out})
}

// handleListAnnouncementDeliveries lista o status de envio de um comunicado.
func (a *App) handleListAnnouncementDeliveries(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	id := r.PathValue("id")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 {
		limit = 100
	}
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	b := boundsFromClaims(claims)
	var out []announcements.Delivery
	var stats announcements.Stats
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		var err error
		// Verify the announcement belongs to the session scope
		_, err = a.Announcements.GetByID(r.Context(), tx, id)
		if err != nil {
			return err
		}
		out, err = a.Announcements.ListDeliveries(r.Context(), tx, id, limit, offset)
		if err != nil {
			return err
		}
		stats, err = a.Announcements.Stats(r.Context(), tx, id)
		return err
	})
	if err != nil {
		if store.IsNotFound(err) {
			writeErr(w, http.StatusNotFound, "announcement not found")
			return
		}
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"deliveries": out,
		"stats":      stats,
	})
}

// handleSendTestMessage envia uma mensagem de teste para um numero especifico
// de forma sincrona (para validar a configuracao do provedor).
func (a *App) handleSendTestMessage(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	_ = claims
	var in struct {
		Phone   string `json:"phone"`
		Message string `json:"message"`
	}
	if err := readJSON(r, &in); err != nil || in.Phone == "" {
		writeErr(w, http.StatusBadRequest, "phone and message required")
		return
	}
	if in.Message == "" {
		in.Message = "Teste de configuracao do Chosen ERP - sua conexao WhatsApp esta funcionando!"
	}

	provider := a.Dispatch.ProviderFor(delivery.ChannelWhatsApp)
	msg := delivery.Message{
		Channel:   delivery.ChannelWhatsApp,
		Recipient: in.Phone,
		Text:      in.Message,
		Subject:   "Comunicado de teste",
		TenantID:  claims.TenantID,
		BranchID:  claims.BranchID,
	}
	if err := a.Dispatch.Send(r.Context(), msg); err != nil {
		writeErr(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"provider": provider,
		"status":   "sent",
		"to":       in.Phone,
	})
}
