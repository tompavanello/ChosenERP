package httpapi

import (
	"errors"
	"net/http"

	"chosenerp/internal/payments"
	"chosenerp/internal/store"
)

// ---- Fluxo B: meios de pagamento e cobrancas da igreja ----

func (a *App) handleListPaymentGateways(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	if !a.paymentsAvailable(w) {
		return
	}
	out, err := a.Payments.ListGatewayConfigs(r.Context(), boundsFromClaims(claims))
	if err != nil {
		writeServerError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"gateways": out})
}

func (a *App) handleSavePaymentGateway(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	if !a.paymentsAvailable(w) {
		return
	}
	var in payments.GatewayConfigInput
	if err := readJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	in.Provider = r.PathValue("provider")
	out, err := a.Payments.SaveGatewayConfig(r.Context(), boundsFromClaims(claims), in)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (a *App) handleListPaymentProducts(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	if !a.paymentsAvailable(w) {
		return
	}
	out, err := a.Payments.ListProducts(r.Context(), boundsFromClaims(claims))
	if err != nil {
		writeServerError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"products": out})
}

func (a *App) handleCreatePaymentProduct(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	if !a.paymentsAvailable(w) {
		return
	}
	var in payments.ProductInput
	if err := readJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	out, err := a.Payments.CreateProduct(r.Context(), boundsFromClaims(claims), in)
	if err != nil {
		if errors.Is(err, payments.ErrProductInvalido) {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, out)
}

func (a *App) handleUpdatePaymentProduct(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	if !a.paymentsAvailable(w) {
		return
	}
	var in payments.ProductInput
	if err := readJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	out, err := a.Payments.UpdateProduct(r.Context(), boundsFromClaims(claims), r.PathValue("id"), in)
	if err != nil {
		if store.IsNotFound(err) {
			writeErr(w, http.StatusNotFound, "produto nao encontrado")
			return
		}
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (a *App) handleDeletePaymentProduct(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	if !a.paymentsAvailable(w) {
		return
	}
	err := a.Payments.DeleteProduct(r.Context(), boundsFromClaims(claims), r.PathValue("id"))
	if err != nil {
		if store.IsNotFound(err) {
			writeErr(w, http.StatusNotFound, "produto nao encontrado")
			return
		}
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (a *App) handleListPaymentCharges(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	if !a.paymentsAvailable(w) {
		return
	}
	out, err := a.Payments.ListCharges(r.Context(), boundsFromClaims(claims), r.URL.Query().Get("status"))
	if err != nil {
		writeServerError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"charges": out})
}

func (a *App) handleCreatePaymentCharge(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	if !a.paymentsAvailable(w) {
		return
	}
	var in payments.ChargeBInput
	if err := readJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	out, err := a.Payments.CreateChargeB(r.Context(), boundsFromClaims(claims), in)
	if err != nil {
		writeBillingErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, out)
}

func (a *App) handleRefundPaymentCharge(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	if !a.paymentsAvailable(w) {
		return
	}
	if err := a.Payments.RefundChargeB(r.Context(), boundsFromClaims(claims), r.PathValue("id")); err != nil {
		writeBillingErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}
