package httpapi

import (
	"net/http"

	"github.com/jackc/pgx/v5"

	"chosenerp/internal/programacao"
	"chosenerp/internal/store"
)

// ---- Programacao (grade de horarios recorrentes) ----

func (a *App) handleListProgramacao(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	b := boundsFromClaims(claims)
	var out []programacao.Programacao
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		var err error
		out, err = a.Programacao.List(r.Context(), tx)
		return err
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"programacoes": out})
}

func (a *App) handleCreateProgramacao(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	var in programacao.CreateInput
	if err := readJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "corpo invalido: "+err.Error())
		return
	}
	if in.Name == "" || in.StartTime == "" {
		writeErr(w, http.StatusBadRequest, "nome e horario sao obrigatorios")
		return
	}
	if in.Weekday < 0 || in.Weekday > 6 {
		writeErr(w, http.StatusBadRequest, "dia da semana invalido")
		return
	}
	if in.Kind != "" && !programacao.Kinds[in.Kind] {
		writeErr(w, http.StatusBadRequest, "tipo invalido")
		return
	}
	b := boundsFromClaims(claims)
	var p *programacao.Programacao
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		branchID, err := a.writeBranchID(r.Context(), tx, claims)
		if err != nil {
			return err
		}
		p, err = a.Programacao.Create(r.Context(), tx, claims.TenantID, branchID, claims.UserID, in)
		return err
	})
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, p)
}

func (a *App) handleUpdateProgramacao(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	var in programacao.UpdateInput
	if err := readJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "corpo invalido: "+err.Error())
		return
	}
	if in.Weekday != nil && (*in.Weekday < 0 || *in.Weekday > 6) {
		writeErr(w, http.StatusBadRequest, "dia da semana invalido")
		return
	}
	if in.Kind != nil && !programacao.Kinds[*in.Kind] {
		writeErr(w, http.StatusBadRequest, "tipo invalido")
		return
	}
	b := boundsFromClaims(claims)
	var p *programacao.Programacao
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		var err error
		p, err = a.Programacao.Update(r.Context(), tx, r.PathValue("id"), in)
		return err
	})
	if err != nil {
		if store.IsNotFound(err) {
			writeErr(w, http.StatusNotFound, "programacao nao encontrada")
			return
		}
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (a *App) handleDeleteProgramacao(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	b := boundsFromClaims(claims)
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		return a.Programacao.Delete(r.Context(), tx, r.PathValue("id"))
	})
	if err != nil {
		if store.IsNotFound(err) {
			writeErr(w, http.StatusNotFound, "programacao nao encontrada")
			return
		}
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// handleGenerateProgramacao publica as ocorrencias na agenda. Body:
// {"from":"YYYY-MM-DD","to":"YYYY-MM-DD","programacao_id":"opcional"}.
func (a *App) handleGenerateProgramacao(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	var in struct {
		From          string `json:"from"`
		To            string `json:"to"`
		ProgramacaoID string `json:"programacao_id"`
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
	var created, updated int64
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		var err error
		created, updated, err = a.Programacao.GenerateEvents(r.Context(), tx, in.ProgramacaoID, in.From, in.To, claims.UserID)
		return err
	})
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"created": created, "updated": updated})
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
