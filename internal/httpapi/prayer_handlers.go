package httpapi

import (
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5"

	"chosenerp/internal/prayer"
	"chosenerp/internal/store"
)

// handleListMyPrayers lista os pedidos enviados pela propria pessoa.
func (a *App) handleListMyPrayers(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	var out []prayer.Request
	err := a.Store.WithTenant(r.Context(), boundsFromClaims(claims), func(tx pgx.Tx) error {
		m, e := a.memberForClaims(r.Context(), tx, claims)
		if e != nil {
			return e
		}
		out, e = a.Prayers.ListMine(r.Context(), tx, m.ID, claims.UserID)
		return e
	})
	if err != nil {
		writeMemberErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"prayer_requests": out})
}

// handlePrayerWall lista o mural publico (visibilidade "igreja").
func (a *App) handlePrayerWall(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	var out []prayer.Request
	err := a.Store.WithTenant(r.Context(), boundsFromClaims(claims), func(tx pgx.Tx) error {
		if _, e := a.memberForClaims(r.Context(), tx, claims); e != nil {
			return e
		}
		var e error
		out, e = a.Prayers.Wall(r.Context(), tx, claims.UserID)
		return e
	})
	if err != nil {
		writeMemberErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"prayer_requests": out})
}

// handleCreatePrayer registra um pedido de oracao do membro.
func (a *App) handleCreatePrayer(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	var in prayer.CreateInput
	if err := readJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	var req *prayer.Request
	err := a.Store.WithTenant(r.Context(), boundsFromClaims(claims), func(tx pgx.Tx) error {
		m, e := a.memberForClaims(r.Context(), tx, claims)
		if e != nil {
			return e
		}
		req, e = a.Prayers.Create(r.Context(), tx, claims.TenantID, m.BranchID, m.ID, claims.UserID, m.FullName, in)
		return e
	})
	if err != nil {
		if errors.Is(err, prayer.ErrInvalid) {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		writeMemberErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, req)
}

// handleReactPrayer registra "estou orando" (idempotente) e devolve o pedido.
func (a *App) handleReactPrayer(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	id := r.PathValue("id")
	var req *prayer.Request
	err := a.Store.WithTenant(r.Context(), boundsFromClaims(claims), func(tx pgx.Tx) error {
		m, e := a.memberForClaims(r.Context(), tx, claims)
		if e != nil {
			return e
		}
		req, e = a.Prayers.React(r.Context(), tx, claims.TenantID, m.BranchID, id, claims.UserID, m.ID)
		return e
	})
	if err != nil {
		writeMemberErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, req)
}

// handleListPrayersModeration lista os pedidos visiveis a quem modera.
func (a *App) handleListPrayersModeration(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	visibility := r.URL.Query().Get("visibility")
	var out []prayer.Request
	err := a.Store.WithTenant(r.Context(), boundsFromClaims(claims), func(tx pgx.Tx) error {
		var e error
		out, e = a.Prayers.ListModeration(r.Context(), tx, claims.UserID, visibility)
		return e
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if out == nil {
		out = []prayer.Request{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"prayer_requests": out})
}

// handleUpdatePrayer modera um pedido (status + observacao pastoral).
func (a *App) handleUpdatePrayer(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	var in struct {
		Status string  `json:"status"`
		Note   *string `json:"answered_note"`
	}
	if err := readJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	note := ""
	if in.Note != nil {
		note = *in.Note
	}
	var req *prayer.Request
	err := a.Store.WithTenant(r.Context(), boundsFromClaims(claims), func(tx pgx.Tx) error {
		var e error
		req, e = a.Prayers.UpdateStatus(r.Context(), tx, claims.UserID, r.PathValue("id"), in.Status, note)
		return e
	})
	if err != nil {
		if errors.Is(err, prayer.ErrInvalid) {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		if store.IsNotFound(err) {
			writeErr(w, http.StatusNotFound, "pedido nao encontrado")
			return
		}
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, req)
}
