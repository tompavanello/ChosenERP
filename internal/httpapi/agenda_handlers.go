package httpapi

import (
	"net/http"

	"chosenerp/internal/events"
	"github.com/jackc/pgx/v5"
)

// handleListAgenda devolve as ocorrencias do calendario com filtros por tipo,
// ministerio, origem e situacao (a "Agenda" do scheduler).
func (a *App) handleListAgenda(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	q := r.URL.Query()
	b := boundsFromClaims(claims)
	var out []events.Event
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		var err error
		out, err = a.Events.ListAgenda(r.Context(), tx,
			first(q["from"]), first(q["to"]), first(q["kind"]),
			first(q["ministry"]), first(q["origin"]), first(q["status"]))
		return err
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"events": out})
}

type moveOccurrenceInput struct {
	StartsAt string  `json:"starts_at"`
	EndsAt   *string `json:"ends_at"`
	Scope    string  `json:"scope"`
}

func (a *App) handleMoveAgendaOccurrence(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	var in moveOccurrenceInput
	if err := readJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "corpo invalido: "+err.Error())
		return
	}
	if in.StartsAt == "" {
		writeErr(w, http.StatusBadRequest, "starts_at e obrigatorio")
		return
	}
	if in.Scope == "" {
		in.Scope = "this"
	}
	b := boundsFromClaims(claims)
	var ev *events.Event
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		var err error
		ev, err = a.Events.MoveOccurrence(r.Context(), tx, r.PathValue("id"), in.StartsAt, in.EndsAt, in.Scope, claims.UserID)
		return err
	})
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, ev)
}

func (a *App) handleDeleteAgendaOccurrence(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	scope := first(r.URL.Query()["scope"])
	if scope == "" {
		scope = "this"
	}
	b := boundsFromClaims(claims)
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		return a.Events.DeleteOccurrence(r.Context(), tx, r.PathValue("id"), scope)
	})
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (a *App) handleSetAgendaStatus(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	var in struct {
		Status string `json:"status"`
	}
	if err := readJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "corpo invalido: "+err.Error())
		return
	}
	b := boundsFromClaims(claims)
	var ev *events.Event
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		var err error
		ev, err = a.Events.SetOccurrenceStatus(r.Context(), tx, r.PathValue("id"), in.Status)
		return err
	})
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, ev)
}
