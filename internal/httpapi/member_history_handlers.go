package httpapi

import (
	"net/http"

	"github.com/jackc/pgx/v5"

	"chosenerp/internal/members"
	"chosenerp/internal/store"
)

// ---- Historico eclesiastico do membro (requisito 1.8) ----

func (a *App) handleListMemberHistory(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	id := r.PathValue("id")
	b := boundsFromClaims(claims)
	var out []members.HistoryEntry
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		var err error
		out, err = a.Members.ListHistory(r.Context(), tx, id)
		return err
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"history": out})
}

func (a *App) handleAddMemberHistory(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	id := r.PathValue("id")
	var in struct {
		Kind       string `json:"kind"`
		Notes      string `json:"notes"`
		OccurredAt string `json:"occurred_at"`
	}
	if err := readJSON(r, &in); err != nil || in.Kind == "" {
		writeErr(w, http.StatusBadRequest, "kind required")
		return
	}
	b := boundsFromClaims(claims)
	var entry *members.HistoryEntry
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		// Confirma que o membro existe no escopo antes de gravar (mensagem 404).
		if _, err := a.Members.Get(r.Context(), tx, id); err != nil {
			return err
		}
		var err error
		entry, err = a.Members.AddHistory(r.Context(), tx, id, in.Kind, in.Notes, in.OccurredAt, claims.UserID)
		return err
	})
	if err != nil {
		if store.IsNotFound(err) {
			writeErr(w, http.StatusNotFound, "member not found")
			return
		}
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, entry)
}
