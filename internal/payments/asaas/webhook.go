package asaas

import (
	"encoding/json"

	"chosenerp/internal/payments"
)

// webhookPayload e o corpo enviado pelo Asaas em um webhook.
type webhookPayload struct {
	ID     string `json:"id"`
	Event  string `json:"event"`
	Payment *struct {
		ID                string  `json:"id"`
		Status            string  `json:"status"`
		BillingType       string  `json:"billingType"`
		Value             float64 `json:"value"`
		NetValue          float64 `json:"netValue"`
		DueDate           string  `json:"dueDate"`
		PaymentDate       string  `json:"paymentDate"`
		InvoiceURL        string  `json:"invoiceUrl"`
		BankSlipURL       string  `json:"bankSlipUrl"`
		ExternalReference string  `json:"externalReference"`
	} `json:"payment"`
	Subscription *struct {
		ID          string `json:"id"`
		Status      string `json:"status"`
		NextDueDate string `json:"nextDueDate"`
	} `json:"subscription"`
}

// ParseWebhook normaliza um payload de webhook do Asaas.
func ParseWebhook(body []byte) (*payments.WebhookEvent, error) {
	var p webhookPayload
	if err := json.Unmarshal(body, &p); err != nil {
		return nil, err
	}
	ev := &payments.WebhookEvent{EventID: p.ID, EventType: p.Event, Raw: body}
	switch {
	case p.Payment != nil:
		ev.Kind = "payment"
		ev.ExternalID = p.Payment.ID
		ev.RawStatus = p.Payment.Status
		ev.Status = payments.NormalizeStatus("asaas", p.Payment.Status)
		ev.ValueCents = int64(p.Payment.Value*100 + 0.5)
		ev.NetValueCents = int64(p.Payment.NetValue*100 + 0.5)
		ev.DueDate = p.Payment.DueDate
		ev.PaidAt = p.Payment.PaymentDate
		ev.InvoiceURL = p.Payment.InvoiceURL
		ev.BankSlipURL = p.Payment.BankSlipURL
		ev.ExternalReference = p.Payment.ExternalReference
	case p.Subscription != nil:
		ev.Kind = "subscription"
		ev.ExternalID = p.Subscription.ID
		ev.RawStatus = p.Subscription.Status
		ev.Status = payments.NormalizeStatus("asaas", p.Subscription.Status)
	}
	if ev.EventID == "" {
		ev.EventID = p.Event + ":" + ev.ExternalID + ":" + ev.RawStatus
	}
	return ev, nil
}
