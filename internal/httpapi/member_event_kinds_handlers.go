package httpapi

import (
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"chosenerp/internal/memberevents"
	"chosenerp/internal/store"
)

// errEventKindDuplicado sinaliza 409 quando nome/slug ja existem no tenant.
var errEventKindDuplicado = errors.New("ja existe um evento com este nome")

// errEventKindEmUso sinaliza 409: o evento tem lancamentos no historico.
var errEventKindEmUso = errors.New("evento em uso")

// validaEventKind confere os dominios aceitos nos campos de efeito.
func validaEventKind(category string, setsStatus string, setsExit string, dateField string, tone string) string {
	if category != "" && !memberevents.Categories[category] {
		return "categoria invalida"
	}
	if setsStatus != "" && !memberevents.Statuses[setsStatus] {
		return "situacao invalida"
	}
	if setsExit != "" && !memberevents.ExitReasons[setsExit] {
		return "motivo de saida invalido"
	}
	if dateField != "" && !memberevents.DateFields[dateField] {
		return "campo de data invalido"
	}
	if tone != "" && !memberevents.Tones[tone] {
		return "tom invalido"
	}
	return ""
}

func (a *App) handleListMemberEventKinds(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	b := boundsFromClaims(claims)
	var out []memberevents.EventKind
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		var err error
		out, err = a.MemberEvents.List(r.Context(), tx)
		return err
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"event_kinds": out})
}

func (a *App) handleCreateMemberEventKind(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	var in memberevents.CreateInput
	if err := readJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	if in.Name == "" {
		writeErr(w, http.StatusBadRequest, "name required")
		return
	}
	if msg := validaEventKind(in.Category, in.SetsStatus, in.SetsExitReason, in.SetsDateField, in.Tone); msg != "" {
		writeErr(w, http.StatusBadRequest, msg)
		return
	}
	b := boundsFromClaims(claims)
	var k *memberevents.EventKind
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		var err error
		k, err = a.MemberEvents.Create(r.Context(), tx, claims.TenantID, in)
		return err
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			writeErr(w, http.StatusConflict, errEventKindDuplicado.Error())
			return
		}
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, k)
}

func (a *App) handleUpdateMemberEventKind(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	id := r.PathValue("id")
	var in memberevents.UpdateInput
	if err := readJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	if msg := validaEventKind(deref(in.Category), deref(in.SetsStatus), deref(in.SetsExitReason), deref(in.SetsDateField), deref(in.Tone)); msg != "" {
		writeErr(w, http.StatusBadRequest, msg)
		return
	}
	b := boundsFromClaims(claims)
	var k *memberevents.EventKind
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		var err error
		k, err = a.MemberEvents.Update(r.Context(), tx, id, in)
		return err
	})
	if err != nil {
		if store.IsNotFound(err) {
			writeErr(w, http.StatusNotFound, "evento not found")
			return
		}
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			writeErr(w, http.StatusConflict, errEventKindDuplicado.Error())
			return
		}
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, k)
}

// handleDeleteMemberEventKind responde 409 quando ja ha lancamentos no
// historico: apagar levaria o vinculo embora; desative-o em vez de excluir.
func (a *App) handleDeleteMemberEventKind(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	id := r.PathValue("id")
	b := boundsFromClaims(claims)
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		n, err := a.MemberEvents.InUse(r.Context(), tx, id)
		if err != nil {
			return err
		}
		if n > 0 {
			return errEventKindEmUso
		}
		return a.MemberEvents.Delete(r.Context(), tx, id)
	})
	if err != nil {
		switch {
		case err == errEventKindEmUso:
			writeErr(w, http.StatusConflict, "evento em uso no historico; desative-o em vez de excluir")
		case store.IsNotFound(err):
			writeErr(w, http.StatusNotFound, "evento not found")
		default:
			writeErr(w, http.StatusInternalServerError, err.Error())
		}
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
