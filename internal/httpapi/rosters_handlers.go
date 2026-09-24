package httpapi

import (
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5"

	"chosenerp/internal/rosters"
	"chosenerp/internal/store"
)

func (a *App) handleListRosters(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	q := r.URL.Query()
	b := boundsFromClaims(claims)
	var out []rosters.Roster
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		var err error
		out, err = a.Rosters.List(r.Context(), tx, q.Get("from"), q.Get("to"), q.Get("ministry"))
		return err
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"rosters": out})
}

func (a *App) handleCreateRoster(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	var in rosters.CreateInput
	if err := readJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "corpo invalido: "+err.Error())
		return
	}
	if in.Title == "" || in.StartsAt == "" {
		writeErr(w, http.StatusBadRequest, "title e starts_at sao obrigatorios")
		return
	}
	if in.CreateEvent && (in.EventID == nil || *in.EventID == "") && (in.EventKindID == nil || *in.EventKindID == "") {
		writeErr(w, http.StatusBadRequest, "para gerar evento automatico, informe o tipo de evento")
		return
	}
	b := boundsFromClaims(claims)
	var out *rosters.Roster
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		branchID, err := a.writeBranchID(r.Context(), tx, claims)
		if err != nil {
			return err
		}
		out, err = a.Rosters.Create(r.Context(), tx, claims.TenantID, branchID, claims.UserID, in)
		return err
	})
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, out)
}

func (a *App) handleGetRoster(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	b := boundsFromClaims(claims)
	var out *rosters.Roster
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		var err error
		out, err = a.Rosters.Get(r.Context(), tx, r.PathValue("id"))
		return err
	})
	if err != nil {
		if store.IsNotFound(err) {
			writeErr(w, http.StatusNotFound, "escala nao encontrada")
			return
		}
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (a *App) handleUpdateRoster(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	var in rosters.UpdateInput
	if err := readJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "corpo invalido: "+err.Error())
		return
	}
	b := boundsFromClaims(claims)
	var out *rosters.Roster
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		var err error
		out, err = a.Rosters.Update(r.Context(), tx, r.PathValue("id"), in)
		return err
	})
	if err != nil {
		if store.IsNotFound(err) {
			writeErr(w, http.StatusNotFound, "escala nao encontrada")
			return
		}
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (a *App) handleDeleteRoster(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	b := boundsFromClaims(claims)
	deleteEvent := r.URL.Query().Get("delete_event") == "true"
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		return a.Rosters.Delete(r.Context(), tx, r.PathValue("id"), deleteEvent)
	})
	if err != nil {
		if store.IsNotFound(err) {
			writeErr(w, http.StatusNotFound, "escala nao encontrada")
			return
		}
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (a *App) handleSetRosterAssignments(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	var in struct {
		Assignments []rosters.AssignmentInput `json:"assignments"`
	}
	if err := readJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "corpo invalido: "+err.Error())
		return
	}
	b := boundsFromClaims(claims)
	var out *rosters.Roster
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		if err := a.Rosters.SetAssignments(r.Context(), tx, r.PathValue("id"), in.Assignments); err != nil {
			return err
		}
		var err error
		out, err = a.Rosters.Get(r.Context(), tx, r.PathValue("id"))
		return err
	})
	if err != nil {
		if store.IsNotFound(err) {
			writeErr(w, http.StatusNotFound, "escala nao encontrada")
			return
		}
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (a *App) handleRespondRosterAssignment(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	var in struct {
		Status string  `json:"status"`
		Notes  *string `json:"notes"`
	}
	if err := readJSON(r, &in); err != nil || in.Status == "" {
		writeErr(w, http.StatusBadRequest, "status e obrigatorio")
		return
	}
	b := boundsFromClaims(claims)
	var out *rosters.Assignment
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		var err error
		out, err = a.Rosters.Respond(r.Context(), tx, r.PathValue("id"), r.PathValue("assignmentId"), in.Status, in.Notes)
		return err
	})
	if err != nil {
		switch {
		case errors.Is(err, rosters.ErrInvalidStatus):
			writeErr(w, http.StatusBadRequest, err.Error())
		case store.IsNotFound(err):
			writeErr(w, http.StatusNotFound, "escalado nao encontrado")
		default:
			writeErr(w, http.StatusBadRequest, err.Error())
		}
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (a *App) handleRosterConflicts(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	b := boundsFromClaims(claims)
	var out []rosters.Conflict
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		var err error
		out, err = a.Rosters.Conflicts(r.Context(), tx, r.PathValue("id"))
		return err
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"conflicts": out})
}

func (a *App) handleRosterSuggestions(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	q := r.URL.Query()
	b := boundsFromClaims(claims)
	var out []rosters.Suggestion
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		var err error
		out, err = a.Rosters.Suggestions(r.Context(), tx, q.Get("ministry_id"), q.Get("starts_at"), q.Get("ends_at"), q.Get("roster_id"))
		return err
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"suggestions": out})
}
