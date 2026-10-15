package httpapi

import (
	"errors"
	"io"
	"net/http"
	"strings"

	"chosenerp/internal/payments"
	"chosenerp/internal/payments/asaas"
	"chosenerp/internal/store"
)

// paymentsAvailable garante que o servico de pagamentos esta ativo.
func (a *App) paymentsAvailable(w http.ResponseWriter) bool {
	if a.Payments == nil {
		writeErr(w, http.StatusServiceUnavailable, "pagamentos indisponiveis")
		return false
	}
	return true
}

func writeBillingErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, payments.ErrDisabled):
		writeErr(w, http.StatusServiceUnavailable, "pagamentos nao configurados")
	case errors.Is(err, payments.ErrNoSubscription):
		writeErr(w, http.StatusNotFound, "igreja sem assinatura")
	default:
		writeErr(w, http.StatusBadRequest, err.Error())
	}
}

// ---- Console da plataforma: cobranca SaaS (Fluxo A) ----

func (a *App) handleAdminBillingOverview(w http.ResponseWriter, r *http.Request) {
	if _, ok := a.platformAdminClaims(w, r); !ok {
		return
	}
	if !a.paymentsAvailable(w) {
		return
	}
	o, err := a.Payments.GetOverview(r.Context())
	if err != nil {
		writeServerError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, o)
}

func (a *App) handleAdminListSubscriptions(w http.ResponseWriter, r *http.Request) {
	if _, ok := a.platformAdminClaims(w, r); !ok {
		return
	}
	if !a.paymentsAvailable(w) {
		return
	}
	subs, err := a.Payments.ListSubscriptions(r.Context())
	if err != nil {
		writeServerError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"subscriptions": subs})
}

func (a *App) handleAdminListInvoices(w http.ResponseWriter, r *http.Request) {
	if _, ok := a.platformAdminClaims(w, r); !ok {
		return
	}
	if !a.paymentsAvailable(w) {
		return
	}
	status := r.URL.Query().Get("status")
	invs, err := a.Payments.ListInvoices(r.Context(), status, 200)
	if err != nil {
		writeServerError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"invoices": invs})
}

func (a *App) handleAdminGetTenantBilling(w http.ResponseWriter, r *http.Request) {
	if _, ok := a.platformAdminClaims(w, r); !ok {
		return
	}
	if !a.paymentsAvailable(w) {
		return
	}
	tb, err := a.Payments.GetTenantBilling(r.Context(), r.PathValue("id"))
	if err != nil {
		if store.IsNotFound(err) {
			writeErr(w, http.StatusNotFound, "igreja nao encontrada")
			return
		}
		writeServerError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, tb)
}

func (a *App) handleAdminSubscribeTenant(w http.ResponseWriter, r *http.Request) {
	if _, ok := a.platformAdminClaims(w, r); !ok {
		return
	}
	if !a.paymentsAvailable(w) {
		return
	}
	var in payments.SubscribeInput
	if err := readJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	tb, err := a.Payments.Subscribe(r.Context(), r.PathValue("id"), in)
	if err != nil {
		writeBillingErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, tb)
}

func (a *App) handleAdminCancelSubscription(w http.ResponseWriter, r *http.Request) {
	if _, ok := a.platformAdminClaims(w, r); !ok {
		return
	}
	if !a.paymentsAvailable(w) {
		return
	}
	if err := a.Payments.CancelSubscription(r.Context(), r.PathValue("id")); err != nil {
		writeBillingErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (a *App) handleAdminChargeTenant(w http.ResponseWriter, r *http.Request) {
	if _, ok := a.platformAdminClaims(w, r); !ok {
		return
	}
	if !a.paymentsAvailable(w) {
		return
	}
	var in payments.ChargeInputView
	if err := readJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	inv, err := a.Payments.CreateCharge(r.Context(), r.PathValue("id"), in)
	if err != nil {
		writeBillingErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, inv)
}

func (a *App) handleAdminSyncTenantBilling(w http.ResponseWriter, r *http.Request) {
	if _, ok := a.platformAdminClaims(w, r); !ok {
		return
	}
	if !a.paymentsAvailable(w) {
		return
	}
	tb, err := a.Payments.SyncTenant(r.Context(), r.PathValue("id"))
	if err != nil {
		writeBillingErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, tb)
}

func (a *App) handleAdminRefundInvoice(w http.ResponseWriter, r *http.Request) {
	if _, ok := a.platformAdminClaims(w, r); !ok {
		return
	}
	if !a.paymentsAvailable(w) {
		return
	}
	tenantID := r.URL.Query().Get("tenant_id")
	if tenantID == "" || !uuidRe.MatchString(tenantID) {
		writeErr(w, http.StatusBadRequest, "tenant_id obrigatorio")
		return
	}
	if err := a.Payments.RefundInvoice(r.Context(), tenantID, r.PathValue("id")); err != nil {
		writeBillingErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// ---- Area da igreja: visao da propria assinatura (Fluxo A) ----

func (a *App) handleGetOwnBilling(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok || claims.TenantID == "" {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	if !a.paymentsAvailable(w) {
		return
	}
	tb, err := a.Payments.GetOwnBilling(r.Context(), claims.TenantID)
	if err != nil {
		if errors.Is(err, payments.ErrDisabled) {
			writeErr(w, http.StatusServiceUnavailable, "pagamentos nao configurados")
			return
		}
		if store.IsNotFound(err) {
			writeErr(w, http.StatusNotFound, "igreja sem assinatura")
			return
		}
		writeServerError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, tb)
}

// ---- Webhook dos gateways ----

func (a *App) handlePaymentsWebhook(w http.ResponseWriter, r *http.Request) {
	if !a.paymentsAvailable(w) {
		return
	}
	provider := strings.ToLower(r.PathValue("provider"))
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}

	var ev *payments.WebhookEvent
	switch provider {
	case "asaas":
		ev, err = asaas.ParseWebhook(body)
	default:
		writeErr(w, http.StatusBadRequest, "provedor nao suportado")
		return
	}
	if err != nil {
		writeErr(w, http.StatusBadRequest, "payload invalido")
		return
	}

	// O token identifica o remetente. A partir da externalReference resolvemos a
	// igreja dona do evento (Fluxo B) para validar contra o token dela; senao
	// valida contra o token da plataforma (Fluxo A). Sem token configurado,
	// rejeita (falha fechado).
	tenantID := payments.TenantFromReference(ev.ExternalReference)
	token := r.Header.Get("asaas-access-token")
	if token == "" {
		token = r.Header.Get("Asaas-Access-Token")
	}
	if !a.Payments.VerifyWebhookToken(r.Context(), token, provider, tenantID) {
		writeErr(w, http.StatusUnauthorized, "webhook token invalido")
		return
	}

	if err := a.Payments.ApplyWebhook(r.Context(), provider, ev); err != nil {
		writeErr(w, http.StatusInternalServerError, "falha ao processar webhook")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}
