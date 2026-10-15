package httpapi

import (
	"encoding/base64"
	"net/http"

	"github.com/jackc/pgx/v5"

	"chosenerp/internal/members"
)

// handlePreviewMembers le a planilha e devolve as primeiras linhas para o
// usuario mapear as colunas antes de importar.
func (a *App) handlePreviewMembers(w http.ResponseWriter, r *http.Request) {
	if _, ok := claimsFrom(r.Context()); !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	var in struct {
		Data     string `json:"data"`
		Filename string `json:"filename"`
	}
	if err := readJSONMax(r, &in, 8<<20); err != nil || in.Data == "" {
		writeErr(w, http.StatusBadRequest, "envie o arquivo")
		return
	}
	raw, err := base64.StdEncoding.DecodeString(in.Data)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "arquivo em base64 invalido")
		return
	}
	rows, err := members.ParseSheet(raw, in.Filename)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "planilha invalida ou formato nao suportado")
		return
	}
	if len(rows) > 50 {
		rows = rows[:50]
	}
	writeJSON(w, http.StatusOK, map[string]any{"rows": rows})
}

// handleImportMembers importa membros em lote via CSV (base64) ou planilha XLSX,
// com mapeamento de colunas e linha inicial. Linhas invalidas sao ignoradas e
// devolvidas em `errors`; a importacao respeita o limite de membros do plano.
func (a *App) handleImportMembers(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	var in struct {
		Data     string         `json:"data"`
		Filename string         `json:"filename"`
		StartRow int            `json:"start_row"`
		Mapping  map[string]int `json:"mapping"`
	}
	if err := readJSONMax(r, &in, 8<<20); err != nil {
		writeErr(w, http.StatusBadRequest, "arquivo invalido ou muito grande (max 8MB)")
		return
	}
	if in.Data == "" {
		writeErr(w, http.StatusBadRequest, "envie o arquivo")
		return
	}
	raw, err := base64.StdEncoding.DecodeString(in.Data)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "arquivo em base64 invalido")
		return
	}
	rows, err := members.ParseSheet(raw, in.Filename)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "planilha invalida ou formato nao suportado")
		return
	}
	if len(rows) == 0 {
		writeErr(w, http.StatusBadRequest, "planilha sem linhas")
		return
	}

	b := boundsFromClaims(claims)
	var res members.ImportResult
	err = a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		branchID, err := a.writeBranchID(r.Context(), tx, claims)
		if err != nil {
			return err
		}
		quota := func() error {
			return a.enforceQuota(r.Context(), tx, claims.TenantID, "members", 0)
		}
		res, err = a.Members.ImportRecords(r.Context(), tx, claims.TenantID, branchID, claims.UserID, rows, in.StartRow, in.Mapping, quota)
		if err != nil {
			return err
		}
		if res.Imported > 0 {
			_, err = tx.Exec(r.Context(), `
				INSERT INTO audit_log (tenant_id, actor_id, action, entity, payload)
				SELECT $1, $2::uuid, 'members.imported', 'members', $3::jsonb
				FROM users WHERE id = $2`,
				claims.TenantID, claims.UserID, "{}")
		}
		return err
	})
	if err != nil {
		if writeQuotaErr(w, err) {
			return
		}
		writeServerError(w, err)
		return
	}
	if res.Errors == nil {
		res.Errors = []members.ImportError{}
	}
	writeJSON(w, http.StatusOK, res)
}
