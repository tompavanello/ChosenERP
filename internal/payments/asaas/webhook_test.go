package asaas

import (
	"testing"

	"chosenerp/internal/payments"
)

func TestParseWebhookPayment(t *testing.T) {
	body := []byte(`{
		"id": "evt_1",
		"event": "PAYMENT_RECEIVED",
		"payment": {
			"id": "pay_9",
			"status": "RECEIVED",
			"billingType": "PIX",
			"value": 99.9,
			"netValue": 98.0,
			"dueDate": "2026-02-01",
			"paymentDate": "2026-01-30",
			"invoiceUrl": "https://x",
			"bankSlipUrl": "",
			"externalReference": "tenant:abc"
		}
	}`)
	ev, err := ParseWebhook(body)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if ev.EventID != "evt_1" || ev.Kind != "payment" || ev.ExternalID != "pay_9" {
		t.Fatalf("evento inesperado: %+v", ev)
	}
	if ev.Status != payments.StatusPaid || ev.ValueCents != 9990 || ev.NetValueCents != 9800 {
		t.Fatalf("valores inesperados: %+v", ev)
	}
	if ev.PaidAt != "2026-01-30" {
		t.Fatalf("paidAt: %q", ev.PaidAt)
	}
}

func TestParseWebhookFallbackEventID(t *testing.T) {
	body := []byte(`{"event":"PAYMENT_OVERDUE","payment":{"id":"pay_1","status":"OVERDUE","value":10}}`)
	ev, err := ParseWebhook(body)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if ev.EventID != "PAYMENT_OVERDUE:pay_1:OVERDUE" {
		t.Fatalf("event id fallback: %q", ev.EventID)
	}
	if ev.Status != payments.StatusOverdue {
		t.Fatalf("status: %q", ev.Status)
	}
}

func TestParseWebhookSubscription(t *testing.T) {
	body := []byte(`{"id":"evt_2","event":"SUBSCRIPTION_DELETED","subscription":{"id":"sub_1","status":"DELETED","nextDueDate":"2026-03-01"}}`)
	ev, err := ParseWebhook(body)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if ev.Kind != "subscription" || ev.ExternalID != "sub_1" || ev.Status != payments.StatusCanceled {
		t.Fatalf("evento inesperado: %+v", ev)
	}
}
