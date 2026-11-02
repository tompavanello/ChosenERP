package httpapi

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"chosenerp/internal/delivery"
	"chosenerp/internal/support"
)

func (a *App) handleCreateTicket(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	if claims.TenantID == "" {
		writeErr(w, http.StatusBadRequest, "chamados disponiveis apenas no contexto de uma igreja")
		return
	}
	if a.Limitador != nil && !a.Limitador.permitir("support:"+claims.UserID) {
		writeErr(w, http.StatusTooManyRequests, "muitos chamados em pouco tempo")
		return
	}
	var in support.CreateInput
	if err := readJSONMax(r, &in, 128<<10); err != nil {
		writeErr(w, http.StatusBadRequest, "payload invalido")
		return
	}
	var ticket *support.Ticket
	err := a.Store.WithTenant(r.Context(), boundsFromClaims(claims), func(tx pgx.Tx) error {
		var e error
		ticket, e = a.Support.Create(r.Context(), tx, claims.TenantID, claims.BranchID, claims.UserID, in)
		return e
	})
	if err != nil {
		if errors.Is(err, support.ErrInvalid) {
			writeErr(w, http.StatusBadRequest, "preencha tipo, assunto e descricao")
			return
		}
		writeServerError(w, err)
		return
	}
	a.notifySupportTicket(ticket)
	writeJSON(w, http.StatusCreated, ticket)
}

func (a *App) handleListMyTickets(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	app := r.URL.Query().Get("app")
	if app != "member" {
		app = "webadmin"
	}
	var out []support.Ticket
	err := a.Store.WithTenant(r.Context(), boundsFromClaims(claims), func(tx pgx.Tx) error {
		var e error
		if isAdmin(claims.Role) {
			out, e = a.Support.ListTenant(r.Context(), tx, r.URL.Query().Get("status"))
		} else {
			out, e = a.Support.ListMine(r.Context(), tx, claims.UserID, app)
		}
		return e
	})
	if err != nil {
		writeServerError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"tickets": out})
}

func (a *App) handleGetMyTicket(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	var t *support.Ticket
	err := a.Store.WithTenant(r.Context(), boundsFromClaims(claims), func(tx pgx.Tx) error {
		var e error
		t, e = a.Support.Get(r.Context(), tx, r.PathValue("id"))
		return e
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeErr(w, http.StatusNotFound, "chamado nao encontrado")
			return
		}
		writeServerError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, t)
}

func (a *App) handleAddTicketMessage(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	var in struct {
		Body string `json:"body"`
	}
	if err := readJSONMax(r, &in, 16<<10); err != nil {
		writeErr(w, http.StatusBadRequest, "payload invalido")
		return
	}
	var msg *support.Message
	err := a.Store.WithTenant(r.Context(), boundsFromClaims(claims), func(tx pgx.Tx) error {
		t, e := a.Support.Get(r.Context(), tx, r.PathValue("id"))
		if e != nil {
			return e
		}
		branch := ""
		if t.BranchID != nil {
			branch = *t.BranchID
		}
		msg, e = a.Support.AddMessage(r.Context(), tx, t.ID, t.TenantID, branch, &claims.UserID, "user", in.Body)
		return e
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeErr(w, http.StatusNotFound, "chamado nao encontrado")
			return
		}
		if errors.Is(err, support.ErrInvalid) {
			writeErr(w, http.StatusBadRequest, "mensagem vazia")
			return
		}
		writeServerError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, msg)
}

func (a *App) handleAdminListTickets(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	if !a.isPlatformAdmin(r.Context(), claims) {
		writeErr(w, http.StatusForbidden, "apenas administrador da plataforma")
		return
	}
	var out []support.Ticket
	err := a.Store.WithSystem(r.Context(), func(tx pgx.Tx) error {
		var e error
		out, e = a.Support.ListTenant(r.Context(), tx, r.URL.Query().Get("status"))
		return e
	})
	if err != nil {
		writeServerError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"tickets": out})
}

func (a *App) handleAdminGetTicket(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	if !a.isPlatformAdmin(r.Context(), claims) {
		writeErr(w, http.StatusForbidden, "apenas administrador da plataforma")
		return
	}
	var t *support.Ticket
	err := a.Store.WithSystem(r.Context(), func(tx pgx.Tx) error {
		var e error
		t, e = a.Support.Get(r.Context(), tx, r.PathValue("id"))
		return e
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeErr(w, http.StatusNotFound, "chamado nao encontrado")
			return
		}
		writeServerError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, t)
}

func (a *App) handleAdminUpdateTicket(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	if !a.isPlatformAdmin(r.Context(), claims) {
		writeErr(w, http.StatusForbidden, "apenas administrador da plataforma")
		return
	}
	var in support.UpdateInput
	if err := readJSONMax(r, &in, 16<<10); err != nil {
		writeErr(w, http.StatusBadRequest, "payload invalido")
		return
	}
	var t *support.Ticket
	err := a.Store.WithSystem(r.Context(), func(tx pgx.Tx) error {
		var e error
		t, e = a.Support.Update(r.Context(), tx, r.PathValue("id"), in)
		return e
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeErr(w, http.StatusNotFound, "chamado nao encontrado")
			return
		}
		if errors.Is(err, support.ErrInvalid) {
			writeErr(w, http.StatusBadRequest, "status/prioridade invalidos")
			return
		}
		writeServerError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, t)
}

func (a *App) handleAdminAddTicketMessage(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	if !a.isPlatformAdmin(r.Context(), claims) {
		writeErr(w, http.StatusForbidden, "apenas administrador da plataforma")
		return
	}
	var in struct {
		Body string `json:"body"`
	}
	if err := readJSONMax(r, &in, 16<<10); err != nil {
		writeErr(w, http.StatusBadRequest, "payload invalido")
		return
	}
	var msg *support.Message
	err := a.Store.WithSystem(r.Context(), func(tx pgx.Tx) error {
		t, e := a.Support.Get(r.Context(), tx, r.PathValue("id"))
		if e != nil {
			return e
		}
		branch := ""
		if t.BranchID != nil {
			branch = *t.BranchID
		}
		msg, e = a.Support.AddMessage(r.Context(), tx, t.ID, t.TenantID, branch, &claims.UserID, "platform", in.Body)
		return e
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeErr(w, http.StatusNotFound, "chamado nao encontrado")
			return
		}
		if errors.Is(err, support.ErrInvalid) {
			writeErr(w, http.StatusBadRequest, "mensagem vazia")
			return
		}
		writeServerError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, msg)
}

// notifySupportTicket envia a notificacao paralela ao inbox (e-mail/WhatsApp).
// Roda em background e nunca falha a abertura do chamado.
func (a *App) notifySupportTicket(t *support.Ticket) {
	if a.Dispatch == nil || t == nil {
		return
	}
	email := strings.TrimSpace(a.Config.SupportEmail)
	phone := strings.TrimSpace(a.Config.SupportWhatsApp)
	if email == "" && phone == "" {
		return
	}
	body := fmt.Sprintf("Novo chamado #%s no Chosen ERP\n\nIgreja: %s\nUsuario: %s\nTipo: %s (%s)\nTela: %s %s\nAssunto: %s\n\n%s",
		shortID(t.ID), t.TenantID, ptr(t.UserName), t.Kind, t.Priority, t.App, t.PagePath, t.Subject, truncateRunes(t.Description, 1200))
	link := strings.TrimRight(a.Config.AppBaseURL, "/") + "/dashboard/platform/support"
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		if email != "" {
			_ = a.Dispatch.Send(ctx, delivery.Message{
				Channel: delivery.ChannelEmail, Recipient: email,
				Subject: "[Chosen ERP] Chamado " + t.Subject,
				Text:    body + "\n\nAbrir inbox: " + link,
				Link:    link,
			})
		}
		if phone != "" {
			for _, p := range strings.Split(phone, ",") {
				p = strings.TrimSpace(p)
				if p == "" {
					continue
				}
				_ = a.Dispatch.Send(ctx, delivery.Message{
					Channel: delivery.ChannelWhatsApp, Recipient: p,
					Text: body + "\n\n" + link,
				})
			}
		}
	}()
}

func ptr(v *string) string {
	if v == nil {
		return "-"
	}
	return *v
}

func shortID(id string) string {
	if len(id) <= 8 {
		return id
	}
	return id[:8]
}



