package httpapi

import (
	"net/http"

	"github.com/jackc/pgx/v5"

	"chosenerp/internal/cultos"
	"chosenerp/internal/store"
)

// ---- Cultos (grade de horarios) ----

func (a *App) handleListCultos(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	b := boundsFromClaims(claims)
	var out []cultos.Culto
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		var err error
		out, err = a.Cultos.List(r.Context(), tx)
		return err
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"cultos": out})
}

func validWeekday(d int) bool { return d >= 0 && d <= 6 }

func (a *App) handleCreateCulto(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	var in cultos.CreateInput
	if err := readJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "corpo invalido: "+err.Error())
		return
	}
	if in.Name == "" || in.StartTime == "" {
		writeErr(w, http.StatusBadRequest, "nome e horario sao obrigatorios")
		return
	}
	if !validWeekday(in.Weekday) {
		writeErr(w, http.StatusBadRequest, "dia da semana invalido")
		return
	}
	b := boundsFromClaims(claims)
	var c *cultos.Culto
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		branchID, err := a.writeBranchID(r.Context(), tx, claims)
		if err != nil {
			return err
		}
		c, err = a.Cultos.Create(r.Context(), tx, claims.TenantID, branchID, claims.UserID, in)
		return err
	})
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, c)
}

func (a *App) handleUpdateCulto(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	var in cultos.UpdateInput
	if err := readJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "corpo invalido: "+err.Error())
		return
	}
	if in.Weekday != nil && !validWeekday(*in.Weekday) {
		writeErr(w, http.StatusBadRequest, "dia da semana invalido")
		return
	}
	b := boundsFromClaims(claims)
	var c *cultos.Culto
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		var err error
		c, err = a.Cultos.Update(r.Context(), tx, r.PathValue("id"), in)
		return err
	})
	if err != nil {
		if store.IsNotFound(err) {
			writeErr(w, http.StatusNotFound, "culto nao encontrado")
			return
		}
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, c)
}

func (a *App) handleDeleteCulto(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	b := boundsFromClaims(claims)
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		return a.Cultos.Delete(r.Context(), tx, r.PathValue("id"))
	})
	if err != nil {
		if store.IsNotFound(err) {
			writeErr(w, http.StatusNotFound, "culto nao encontrado")
			return
		}
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// handleGenerateCultos publica as ocorrencias dos cultos na agenda no periodo
// informado. Body: {"from":"YYYY-MM-DD","to":"YYYY-MM-DD","culto_id":"opcional"}.
func (a *App) handleGenerateCultos(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	var in struct {
		From    string `json:"from"`
		To      string `json:"to"`
		CultoID string `json:"culto_id"`
	}
	if err := readJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "corpo invalido: "+err.Error())
		return
	}
	if !isISODate(in.From) || !isISODate(in.To) {
		writeErr(w, http.StatusBadRequest, "informe o periodo (from e to no formato YYYY-MM-DD)")
		return
	}
	if in.From > in.To {
		writeErr(w, http.StatusBadRequest, "periodo invalido: from maior que to")
		return
	}
	b := boundsFromClaims(claims)
	var created int64
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		var err error
		created, err = a.Cultos.GenerateEvents(r.Context(), tx, in.CultoID, in.From, in.To, claims.UserID)
		return err
	})
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"created": created})
}

// isISODate confere o formato YYYY-MM-DD sem depender do parser do Postgres.
func isISODate(s string) bool {
	if len(s) != 10 || s[4] != '-' || s[7] != '-' {
		return false
	}
	for i, ch := range s {
		if i == 4 || i == 7 {
			continue
		}
		if ch < '0' || ch > '9' {
			return false
		}
	}
	return true
}
