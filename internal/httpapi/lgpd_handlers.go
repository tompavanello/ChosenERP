package httpapi

import (
	"net/http"

	"github.com/jackc/pgx/v5"

	"chosenerp/internal/lgpd"
	"chosenerp/internal/store"
)

// ---- Termos de consentimento ----

func (a *App) handleListConsentTerms(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	b := boundsFromClaims(claims)
	var out []lgpd.Term
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		var err error
		out, err = a.LGPD.ListTerms(r.Context(), tx)
		return err
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"terms": out})
}

func (a *App) handleCreateConsentTerm(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	if !isAdmin(claims.Role) {
		writeErr(w, http.StatusForbidden, "forbidden")
		return
	}
	var in struct {
		Title string `json:"title"`
		Body  string `json:"body"`
	}
	if err := readJSON(r, &in); err != nil || in.Title == "" || in.Body == "" {
		writeErr(w, http.StatusBadRequest, "title e body são obrigatórios")
		return
	}
	b := boundsFromClaims(claims)
	var t *lgpd.Term
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		var err error
		t, err = a.LGPD.CreateTerm(r.Context(), tx, claims.TenantID, in.Title, in.Body)
		return err
	})
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, t)
}

// ---- Consentimentos do membro ----

func (a *App) handleListMemberConsents(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	b := boundsFromClaims(claims)
	var out []lgpd.Consent
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		var err error
		out, err = a.LGPD.ListConsents(r.Context(), tx, "member", r.PathValue("id"))
		return err
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"consents": out})
}

func (a *App) handleRecordConsent(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	var in struct {
		TermID    string `json:"term_id"`
		Consented bool   `json:"consented"`
	}
	if err := readJSON(r, &in); err != nil || in.TermID == "" {
		writeErr(w, http.StatusBadRequest, "term_id é obrigatório")
		return
	}
	b := boundsFromClaims(claims)
	var c *lgpd.Consent
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		if _, err := a.Members.Get(r.Context(), tx, r.PathValue("id")); err != nil {
			return err
		}
		branchID, err := a.writeBranchID(r.Context(), tx, claims)
		if err != nil {
			return err
		}
		c, err = a.LGPD.RecordConsent(r.Context(), tx, claims.TenantID, branchID,
			"member", r.PathValue("id"), in.TermID, in.Consented,
			clientIP(r), r.UserAgent())
		return err
	})
	if err != nil {
		if store.IsNotFound(err) {
			writeErr(w, http.StatusNotFound, "membro ou termo não encontrado")
			return
		}
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, c)
}

// ---- Portabilidade ----

func (a *App) handleExportMemberData(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	b := boundsFromClaims(claims)
	var data []byte
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		var err error
		data, err = a.LGPD.ExportMember(r.Context(), tx, r.PathValue("id"))
		return err
	})
	if err != nil {
		if store.IsNotFound(err) {
			writeErr(w, http.StatusNotFound, "membro não encontrado")
			return
		}
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="dados-membro-`+r.PathValue("id")+`.json"`)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

// ---- Anonimização ----

func (a *App) handleAnonymizeMember(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	if !isAdmin(claims.Role) {
		writeErr(w, http.StatusForbidden, "forbidden")
		return
	}
	b := boundsFromClaims(claims)
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		return a.LGPD.AnonymizeMember(r.Context(), tx, r.PathValue("id"), claims.UserID)
	})
	if err != nil {
		if store.IsNotFound(err) {
			writeErr(w, http.StatusNotFound, "membro não encontrado")
			return
		}
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// clientIP devolve o IP do cliente considerando proxies.
func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return xff
	}
	return r.RemoteAddr
}
