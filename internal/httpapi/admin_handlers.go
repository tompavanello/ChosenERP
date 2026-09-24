package httpapi

import (
	"net/http"
	"strings"
)

// handleResetData limpa todo o dado operacional, mantendo a base (tenant,
// filiais, papeis, permissoes) e apenas o usuario super_admin. Pensado para
// reiniciar testes operacionais. Exige confirmacao explicita no corpo.
func (a *App) handleResetData(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	if claims.Role != "super_admin" {
		writeErr(w, http.StatusForbidden, "apenas super_admin pode resetar os dados")
		return
	}
	var in struct {
		Confirm string `json:"confirm"`
	}
	if err := readJSON(r, &in); err != nil || strings.ToUpper(strings.TrimSpace(in.Confirm)) != "RESET" {
		writeErr(w, http.StatusBadRequest, `envie {"confirm":"RESET"} para confirmar`)
		return
	}
	if _, err := a.Store.Pool().Exec(r.Context(), `SELECT reset_operational_data()`); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}
