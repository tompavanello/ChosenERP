package httpapi

import (
	"errors"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"chosenerp/internal/auth"
	"chosenerp/internal/members"
	"chosenerp/internal/store"
)

// handleGetMemberAccess devolve a situacao do acesso do membro ao app.
func (a *App) handleGetMemberAccess(w http.ResponseWriter, r *http.Request) {
	claims, ok := a.adminClaims(w, r)
	if !ok {
		return
	}
	memberID := r.PathValue("id")
	var acc *members.Access
	err := a.Store.WithTenant(r.Context(), boundsFromClaims(claims), func(tx pgx.Tx) error {
		var e error
		acc, e = a.Members.GetAccess(r.Context(), tx, memberID)
		return e
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, acc)
}

// handleCreateMemberAccess cria/anexa o acesso (identidade + senha provisoria).
func (a *App) handleCreateMemberAccess(w http.ResponseWriter, r *http.Request) {
	claims, ok := a.adminClaims(w, r)
	if !ok {
		return
	}
	memberID := r.PathValue("id")
	var in members.AccessInput
	if err := readJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	email := strings.TrimSpace(strDeref(in.Email))
	if email == "" {
		writeErr(w, http.StatusBadRequest, "email e obrigatorio")
		return
	}
	if len(in.Password) < 8 {
		writeErr(w, http.StatusBadRequest, "a senha deve ter ao menos 8 caracteres")
		return
	}
	phone := strDeref(in.Phone)
	hash, err := auth.HashPassword(in.Password)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}

	var acc *members.Access
	err = a.Store.WithTenant(r.Context(), boundsFromClaims(claims), func(tx pgx.Tx) error {
		m, e := a.Members.Get(r.Context(), tx, memberID)
		if e != nil {
			return e
		}
		// Quota so quando o acesso ainda nao existe (redefinir nao cria vinculo).
		if existing, e := a.Members.GetAccess(r.Context(), tx, memberID); e == nil && !existing.HasAccess {
			if e := a.enforceQuota(r.Context(), tx, claims.TenantID, "users", 0); e != nil {
				return e
			}
		}
		acc, e = a.Members.SetAccess(r.Context(), tx, claims.TenantID, memberID, email, phone, hash, m.FullName)
		return e
	})
	if err != nil {
		if writeQuotaErr(w, err) {
			return
		}
		if writeAccessErr(w, err) {
			return
		}
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, acc)
}

// handleUpdateMemberAccess altera e-mail/telefone e ativa/desativa o acesso.
func (a *App) handleUpdateMemberAccess(w http.ResponseWriter, r *http.Request) {
	claims, ok := a.adminClaims(w, r)
	if !ok {
		return
	}
	memberID := r.PathValue("id")
	var in members.AccessInput
	if err := readJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	var acc *members.Access
	err := a.Store.WithTenant(r.Context(), boundsFromClaims(claims), func(tx pgx.Tx) error {
		var e error
		acc, e = a.Members.UpdateAccess(r.Context(), tx, memberID, in)
		return e
	})
	if err != nil {
		if store.IsNotFound(err) {
			writeErr(w, http.StatusNotFound, "membro sem acesso cadastrado")
			return
		}
		if writeAccessErr(w, err) {
			return
		}
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, acc)
}

// handleResetMemberAccessPassword define uma nova senha provisoria para o membro.
func (a *App) handleResetMemberAccessPassword(w http.ResponseWriter, r *http.Request) {
	claims, ok := a.adminClaims(w, r)
	if !ok {
		return
	}
	memberID := r.PathValue("id")
	var in struct {
		Password string `json:"password"`
	}
	if err := readJSON(r, &in); err != nil || len(in.Password) < 8 {
		writeErr(w, http.StatusBadRequest, "a senha deve ter ao menos 8 caracteres")
		return
	}
	hash, err := auth.HashPassword(in.Password)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	err = a.Store.WithTenant(r.Context(), boundsFromClaims(claims), func(tx pgx.Tx) error {
		return a.Members.ResetAccessPassword(r.Context(), tx, memberID, hash, true)
	})
	if err != nil {
		if store.IsNotFound(err) {
			writeErr(w, http.StatusNotFound, "membro sem acesso cadastrado")
			return
		}
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// writeAccessErr traduz os erros de unicidade/chave do acesso. Devolve true se
// ja escreveu a resposta.
func writeAccessErr(w http.ResponseWriter, err error) bool {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return false
	}
	switch pgErr.Code {
	case "23505":
		writeErr(w, http.StatusConflict, "e-mail ou telefone ja em uso, ou membro ja vinculado a outro acesso")
	case "P0002", "22023":
		writeErr(w, http.StatusBadRequest, pgErr.Message)
	default:
		return false
	}
	return true
}

func strDeref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
