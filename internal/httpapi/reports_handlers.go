package httpapi

import (
	"net/http"

	"github.com/jackc/pgx/v5"

	"chosenerp/internal/documents"
	"chosenerp/internal/finance"
	"chosenerp/internal/store"
)

// ---- Relatórios financeiros ----

func (a *App) handleMonthlyBalance(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	q := r.URL.Query()
	b := boundsFromClaims(claims)
	var points []finance.MonthlyPoint
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		var err error
		points, err = a.Finance.MonthlySeries(r.Context(), tx, q.Get("from"), q.Get("to"))
		return err
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"series": points})
}

func (a *App) handleDRE(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	q := r.URL.Query()
	b := boundsFromClaims(claims)
	var dre finance.DRE
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		var err error
		dre, err = a.Finance.BuildDRE(r.Context(), tx, q.Get("from"), q.Get("to"))
		return err
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, dre)
}

// ---- Recibos: renderização e envio ----

func (a *App) handleRenderReceipt(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	id := r.PathValue("id")
	b := boundsFromClaims(claims)
	var body string
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		doc, err := a.Documents.GetByID(r.Context(), tx, id)
		if err != nil {
			return err
		}
		tenant, _ := a.Documents.TenantName(r.Context(), tx)
		body, err = documents.RenderReceiptHTML(doc, tenant)
		return err
	})
	if err != nil {
		if store.IsNotFound(err) {
			writeErr(w, http.StatusNotFound, "document not found")
			return
		}
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(body))
}

func (a *App) handleSendDocument(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	id := r.PathValue("id")
	var in struct {
		Channel   string `json:"channel"`
		Recipient string `json:"recipient"`
	}
	if err := readJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	b := boundsFromClaims(claims)
	var d *documents.Delivery
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		if _, err := a.Documents.GetByID(r.Context(), tx, id); err != nil {
			return err
		}
		var err error
		d, err = a.Documents.QueueDelivery(r.Context(), tx, claims.TenantID, claims.BranchID, id, in.Channel, in.Recipient)
		if err != nil {
			return err
		}
		d, err = a.Documents.Dispatch(r.Context(), tx, d.ID)
		return err
	})
	if err != nil {
		if store.IsNotFound(err) {
			writeErr(w, http.StatusNotFound, "document not found")
			return
		}
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, d)
}

func (a *App) handleListDeliveries(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	id := r.PathValue("id")
	b := boundsFromClaims(claims)
	var out []documents.Delivery
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		var err error
		out, err = a.Documents.ListDeliveries(r.Context(), tx, id)
		return err
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"deliveries": out})
}
