package httpapi

import (
	"errors"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5"

	"chosenerp/internal/requests"
	"chosenerp/internal/store"
)

// handleListMyRequests lista as solicitacoes do proprio membro.
func (a *App) handleListMyRequests(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	var out []requests.Request
	err := a.Store.WithTenant(r.Context(), boundsFromClaims(claims), func(tx pgx.Tx) error {
		m, e := a.memberForClaims(r.Context(), tx, claims)
		if e != nil {
			return e
		}
		out, e = a.Requests.ListForMember(r.Context(), tx, m.ID)
		return e
	})
	if err != nil {
		writeMemberErr(w, err)
		return
	}
	if out == nil {
		out = []requests.Request{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"requests": out})
}

// handleCreateMyRequest cria uma solicitacao do proprio membro.
func (a *App) handleCreateMyRequest(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	var in requests.CreateInput
	if err := readJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	in.Subject = strings.TrimSpace(in.Subject)
	if in.Subject == "" {
		writeErr(w, http.StatusBadRequest, "assunto obrigatorio")
		return
	}
	if in.Kind == "" {
		in.Kind = "outro"
	}
	var created *requests.Request
	err := a.Store.WithTenant(r.Context(), boundsFromClaims(claims), func(tx pgx.Tx) error {
		m, e := a.memberForClaims(r.Context(), tx, claims)
		if e != nil {
			return e
		}
		branchID, e := a.writeBranchID(r.Context(), tx, claims)
		if e != nil {
			return e
		}
		created, e = a.Requests.Create(r.Context(), tx, claims.TenantID, branchID, m.ID, claims.UserID, in)
		return e
	})
	if err != nil {
		if errors.Is(err, requests.ErrInvalidKind) {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		writeMemberErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"request": created})
}

// handleListRequests lista as solicitacoes do escopo (secretaria).
func (a *App) handleListRequests(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	status := r.URL.Query().Get("status")
	var out []requests.Request
	err := a.Store.WithTenant(r.Context(), boundsFromClaims(claims), func(tx pgx.Tx) error {
		var e error
		out, e = a.Requests.List(r.Context(), tx, status)
		return e
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if out == nil {
		out = []requests.Request{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"requests": out})
}

// handleUpdateRequest responde a solicitacao (status + observacao).
func (a *App) handleUpdateRequest(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	var in requests.UpdateInput
	if err := readJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	var out *requests.Request
	err := a.Store.WithTenant(r.Context(), boundsFromClaims(claims), func(tx pgx.Tx) error {
		var e error
		out, e = a.Requests.Update(r.Context(), tx, r.PathValue("id"), in, claims.UserID)
		return e
	})
	if err != nil {
		if errors.Is(err, requests.ErrInvalidStatus) {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		if store.IsNotFound(err) {
			writeErr(w, http.StatusNotFound, "solicitacao nao encontrada")
			return
		}
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"request": out})
}
