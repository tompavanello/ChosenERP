package payments

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"

	"chosenerp/internal/store"
)

// ApplyWebhook processa um evento normalizado de gateway. E idempotente: um
// evento ja processado (mesmo provider+event_id) nao e reaplicado.
//
// O registro do evento e confirmado ANTES do processamento, de modo que falhas
// no processamento possam ser persistidas na coluna `error` (a transacao de
// processamento e revertida e o motivo fica registrado).
func (s *Service) ApplyWebhook(ctx context.Context, provider string, ev *WebhookEvent) error {
	tenantID := TenantFromReference(ev.ExternalReference)

	var alreadyProcessed bool
	if err := s.Store.WithSystem(ctx, func(tx pgx.Tx) error {
		inserted, processed, err := s.registerEvent(ctx, tx, provider, tenantID, ev)
		if err != nil {
			return err
		}
		alreadyProcessed = !inserted && processed
		return nil
	}); err != nil {
		return err
	}
	if alreadyProcessed {
		return nil
	}

	if err := s.Store.WithSystem(ctx, func(tx pgx.Tx) error {
		switch ev.Kind {
		case "payment":
			return s.applyPayment(ctx, tx, provider, tenantID, ev)
		case "subscription":
			if ev.ExternalID != "" {
				return s.applySubscription(ctx, tx, provider, tenantID, ev)
			}
		}
		return nil
	}); err != nil {
		s.markEventError(ctx, provider, ev.EventID, err)
		return err
	}

	return s.Store.WithSystem(ctx, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx,
			`UPDATE billing_webhook_events SET processed_at=now(), error=NULL WHERE provider=$1 AND event_id=$2`,
			provider, ev.EventID)
		return err
	})
}

// markEventError registra o motivo da falha (em transacao propria, para nao ser
// revertido junto com o processamento).
func (s *Service) markEventError(ctx context.Context, provider, eventID string, cause error) {
	_ = s.Store.WithSystem(ctx, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx,
			`UPDATE billing_webhook_events SET error=$3 WHERE provider=$1 AND event_id=$2`,
			provider, eventID, cause.Error())
		return err
	})
}

// registerEvent garante a unicidade do evento. Devolve (inserido, jaProcessado).
func (s *Service) registerEvent(ctx context.Context, tx pgx.Tx, provider, tenantID string, ev *WebhookEvent) (bool, bool, error) {
	scope := "platform"
	var tenant any
	if tenantID != "" {
		scope = "tenant"
		tenant = tenantID
	}
	var id string
	err := tx.QueryRow(ctx, `
		INSERT INTO billing_webhook_events (provider, event_id, event_type, payload, scope, tenant_id)
		VALUES ($1,$2,$3,$4::jsonb,$5,$6::uuid)
		ON CONFLICT (provider, event_id) DO NOTHING
		RETURNING id`, provider, ev.EventID, ev.EventType, rawJSONB(ev.Raw), scope, tenant).Scan(&id)
	if err == nil {
		return true, false, nil
	}
	if !store.IsNotFound(err) {
		return false, false, err
	}
	var processed *string
	if err := tx.QueryRow(ctx, `
		SELECT processed_at::text FROM billing_webhook_events WHERE provider=$1 AND event_id=$2`,
		provider, ev.EventID).Scan(&processed); err != nil {
		return false, false, err
	}
	return false, processed != nil, nil
}

// applyPayment aplica um evento de pagamento na fatura (Fluxo A) ou na cobranca
// (Fluxo B) correspondente. Quando o tenant e conhecido, restringe a busca a ele
// (evita sobrescrita cross-tenant por colisao de external_id).
func (s *Service) applyPayment(ctx context.Context, tx pgx.Tx, provider, tenantID string, ev *WebhookEvent) error {
	if ev.ExternalID == "" {
		return nil
	}
	// Fluxo A: fatura da assinatura SaaS.
	var invTenantID string
	err := tx.QueryRow(ctx, `
		UPDATE billing_invoices SET
			status=$3,
			paid_at=COALESCE(NULLIF($4,'')::timestamptz, paid_at),
			value_cents=CASE WHEN $5>0 THEN $5 ELSE value_cents END,
			net_value_cents=CASE WHEN $6>0 THEN $6 ELSE net_value_cents END,
			raw=$7::jsonb, updated_at=now()
		WHERE provider=$1 AND external_id=$2 AND ($8='' OR tenant_id=$8::uuid)
		RETURNING tenant_id`, provider, ev.ExternalID, ev.Status, ev.PaidAt,
		ev.ValueCents, ev.NetValueCents, string(ev.Raw), tenantID).Scan(&invTenantID)
	if err == nil {
		return s.recomputeTenantStatus(ctx, tx, invTenantID)
	}
	if !store.IsNotFound(err) {
		return err
	}

	// Fluxo B: cobranca da igreja.
	var chargeID, chargeStatus string
	err = tx.QueryRow(ctx, `
		UPDATE payment_charges SET
			status=$3,
			paid_at=COALESCE(NULLIF($4,'')::timestamptz, paid_at),
			value_cents=CASE WHEN $5>0 THEN $5 ELSE value_cents END,
			raw=$6::jsonb, updated_at=now()
		WHERE provider=$1 AND external_id=$2 AND ($7='' OR tenant_id=$7::uuid)
		RETURNING id, status`, provider, ev.ExternalID, ev.Status, ev.PaidAt,
		ev.ValueCents, string(ev.Raw), tenantID).Scan(&chargeID, &chargeStatus)
	if err != nil {
		if store.IsNotFound(err) {
			return nil
		}
		return err
	}
	if chargeStatus == StatusPaid {
		return s.reconcileCharge(ctx, tx, chargeID)
	}
	return nil
}

// applySubscription atualiza as assinaturas (Fluxo A e Fluxo B) do evento.
func (s *Service) applySubscription(ctx context.Context, tx pgx.Tx, provider, tenantID string, ev *WebhookEvent) error {
	if _, err := tx.Exec(ctx, `
		UPDATE billing_subscriptions SET status=$3, next_due_date=COALESCE(NULLIF($4,'')::date, next_due_date),
			updated_at=now()
		WHERE provider=$1 AND external_id=$2 AND ($5='' OR tenant_id=$5::uuid)`,
		provider, ev.ExternalID, subStatus(ev.Status), ev.DueDate, tenantID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE payment_subscriptions SET status=$3, next_due_date=COALESCE(NULLIF($4,'')::date, next_due_date),
			updated_at=now()
		WHERE provider=$1 AND external_id=$2 AND ($5='' OR tenant_id=$5::uuid)`,
		provider, ev.ExternalID, subStatus(ev.Status), ev.DueDate, tenantID); err != nil {
		return err
	}
	return nil
}

func rawJSONB(b []byte) string {
	if len(b) == 0 {
		return "{}"
	}
	return string(b)
}

// reconcileCharge gera, uma unica vez, o lancamento financeiro de uma cobranca
// paga, usando a categoria/valor definidos no produto (parametrizavel).
func (s *Service) reconcileCharge(ctx context.Context, tx pgx.Tx, chargeID string) error {
	var tenantID, branchID, productID, memberID, billingType, description string
	var amountCents int64
	var existingTxID *string
	err := tx.QueryRow(ctx, `
		SELECT tenant_id::text, COALESCE(branch_id::text,''), COALESCE(product_id::text,''),
		       COALESCE(member_id::text,''), billing_type, amount_cents, COALESCE(description,''),
		       financial_transaction_id::text
		FROM payment_charges WHERE id=$1::uuid FOR UPDATE`, chargeID).
		Scan(&tenantID, &branchID, &productID, &memberID, &billingType, &amountCents, &description, &existingTxID)
	if err != nil {
		return err
	}
	if existingTxID != nil {
		return nil // ja conciliado
	}

	// O produto define a categoria contabil (e a conta, opcional).
	var categoryID, accountID *string
	if productID != "" {
		if err := tx.QueryRow(ctx, `
			SELECT category_id::text, account_id::text FROM payment_products WHERE id=$1::uuid`,
			productID).Scan(&categoryID, &accountID); err != nil && !store.IsNotFound(err) {
			return err
		}
	}
	if amountCents <= 0 {
		return errors.New("cobranca com valor zerado nao gera lancamento")
	}
	if accountID == nil || *accountID == "" {
		return errors.New("produto da cobranca sem conta bancaria vinculada")
	}
	if branchID == "" {
		if b, e := store.DefaultBranchID(ctx, tx, tenantID); e == nil {
			branchID = b
		}
	}
	if branchID == "" {
		return errors.New("igreja sem filial para lancar a contribuicao")
	}

	var txID string
	var donor any
	if memberID != "" {
		donor = memberID
	}
	if err := tx.QueryRow(ctx, `
		INSERT INTO financial_transactions
			(tenant_id, branch_id, category_id, account_id, type, amount, currency,
			 donor_member_id, payment_method, description, occurred_at)
		VALUES ($1::uuid,$2::uuid,$3,$4,'income',$5::numeric/100.0,'BRL',
		        $6,$7,$8, COALESCE((SELECT paid_at FROM payment_charges WHERE id=$9::uuid), now()))
		RETURNING id`,
		tenantID, branchID, categoryID, accountID, amountCents,
		donor, paymentMethod(billingType), description, chargeID).Scan(&txID); err != nil {
		return err
	}
	_, err = tx.Exec(ctx,
		`UPDATE payment_charges SET financial_transaction_id=$1::uuid, updated_at=now() WHERE id=$2::uuid`,
		txID, chargeID)
	return err
}

func subStatus(normalized string) string {
	switch normalized {
	case StatusPaid:
		return "active"
	case StatusCanceled:
		return "canceled"
	case StatusOverdue:
		return "active"
	default:
		return "active"
	}
}

func paymentMethod(billingType string) string {
	switch strings.ToUpper(billingType) {
	case BillingPix:
		return "pix"
	case BillingCreditCard:
		return "card"
	case BillingBoleto:
		return "boleto"
	default:
		return "pix"
	}
}
