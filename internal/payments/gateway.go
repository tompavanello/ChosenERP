// Package payments e a camada de cobranca do ChosenERP.
//
// Ele abstrai os provedores de pagamento (Asaas, MercadoPago, Stone, PicPay)
// por tras da interface Gateway, de forma que o restante do sistema nao conheca
// o formato de nenhum provedor especifico. Existem dois fluxos:
//
//   - Fluxo A (SaaS): a plataforma cobra a IGREJA pelos planos. A conta do
//     gateway e a da plataforma (config de `payment_gateway_configs` com
//     tenant_id NULL, ou as variaveis ASAAS_* do ambiente).
//   - Fluxo B (arrecadacao): a igreja cobra os membros. Cada igreja configura o
//     proprio provider.
package payments

import (
	"context"
	"strings"
)

// Status normalizados (comuns a todos os provedores).
const (
	StatusPending  = "pending"
	StatusPaid     = "paid"
	StatusOverdue  = "overdue"
	StatusCanceled = "canceled"
	StatusRefunded = "refunded"
	StatusFailed   = "failed"
	StatusReceived = "received" // pagamento confirmado, aguardando liquidacao
)

// Formas de pagamento normalizadas.
const (
	BillingPix        = "PIX"
	BillingBoleto     = "BOLETO"
	BillingCreditCard = "CREDIT_CARD"
)

// Ciclos normalizados.
const (
	CycleMonthly = "MONTHLY"
	CycleYearly  = "YEARLY"
)

// Gateway e implementado por cada provedor de pagamento.
type Gateway interface {
	Provider() string

	CreateCustomer(ctx context.Context, in CustomerInput) (*Customer, error)
	UpdateCustomer(ctx context.Context, externalID string, in CustomerInput) (*Customer, error)

	CreateSubscription(ctx context.Context, in SubscriptionInput) (*Subscription, error)
	UpdateSubscription(ctx context.Context, externalID string, in SubscriptionInput) (*Subscription, error)
	CancelSubscription(ctx context.Context, externalID string) error
	GetSubscription(ctx context.Context, externalID string) (*Subscription, error)

	CreateCharge(ctx context.Context, in ChargeInput) (*Charge, error)
	GetCharge(ctx context.Context, externalID string) (*Charge, error)
	RefundCharge(ctx context.Context, externalID string, amountCents *int64) (*Charge, error)
	PixQRCode(ctx context.Context, chargeExternalID string) (*PixQRCode, error)
}

// CustomerInput e a entrada para criar/atualizar um cliente no gateway.
type CustomerInput struct {
	ExternalReference string
	Name              string
	Email             string
	Phone             string
	CPFCNPJ           string
}

// Customer e o cliente criado no gateway.
type Customer struct {
	ExternalID string
	Name       string
	Email      string
	Phone      string
	CPFCNPJ    string
}

// Card e o cartao de credito tokenizado/enviado ao gateway. NUNCA persistimos
// esses dados; apenas repassamos ao provedor.
type Card struct {
	HolderName    string `json:"holder_name"`
	Number        string `json:"number"`
	ExpiryMonth   string `json:"expiry_month"`
	ExpiryYear    string `json:"expiry_year"`
	CCV           string `json:"ccv"`
	HolderCPF     string `json:"holder_cpf"`
	PostalCode    string `json:"postal_code"`
	AddressNumber string `json:"address_number"`
}

// SubscriptionInput e a entrada para criar/atualizar uma assinatura (recorrente).
type SubscriptionInput struct {
	CustomerExternalID string
	ExternalReference  string
	Description        string
	ValueCents         int64
	Cycle              string // MONTHLY | YEARLY
	BillingType        string // PIX | BOLETO | CREDIT_CARD
	NextDueDate        string // YYYY-MM-DD
	Card               *Card
}

// Subscription e a assinatura criada no gateway.
type Subscription struct {
	ExternalID  string
	Status      string
	ValueCents  int64
	Cycle       string
	BillingType string
	NextDueDate string
}

// ChargeInput e a entrada para uma cobranca avulsa.
type ChargeInput struct {
	CustomerExternalID string
	ExternalReference  string
	Description        string
	ValueCents         int64
	DueDate            string // YYYY-MM-DD
	BillingType        string // PIX | BOLETO | CREDIT_CARD
	InstallmentCount   int
	Card               *Card
}

// Charge e uma cobranca no gateway.
type Charge struct {
	ExternalID         string
	Status             string // normalizado
	RawStatus          string // status cru do provedor (auditoria)
	BillingType        string
	ValueCents         int64
	NetValueCents      int64
	DueDate            string
	PaidAt             string
	InvoiceURL         string
	BankSlipURL        string
	TransactionReceipt string // URL do comprovante
}

// PixQRCode e o QR Code (imagem base64) + payload copia-e-cola de uma cobranca Pix.
type PixQRCode struct {
	EncodedImage   string // base64 (sem prefixo data:)
	Payload        string
	ExpirationDate string
}

// WebhookEvent e um evento de webhook ja normalizado (provider-agnostico).
type WebhookEvent struct {
	EventID           string
	EventType         string // ex.: PAYMENT_RECEIVED (cru, para auditoria)
	Kind              string // payment | subscription
	ExternalID        string // id do payment/subscription no gateway
	Status            string // normalizado
	RawStatus         string
	ValueCents        int64
	NetValueCents     int64
	DueDate           string
	PaidAt            string
	InvoiceURL        string
	BankSlipURL       string
	ExternalReference string
	Raw               []byte
}

// NormalizeStatus converte status crus de provedores em um dos status normalizados.
func NormalizeStatus(provider, s string) string {
	switch provider {
	case "asaas":
		return normalizeAsaas(s)
	default:
		return strings.ToLower(s)
	}
}

func normalizeAsaas(s string) string {
	switch strings.ToUpper(s) {
	case "PENDING", "AWAITING_RISK_ANALYSIS", "DUNNING_REQUESTED", "AWAITING_CHARGEBACK_REVERSAL":
		return StatusPending
	case "RECEIVED", "CONFIRMED", "RECEIVED_IN_CASH", "DUNNING_RECEIVED":
		return StatusPaid
	case "OVERDUE":
		return StatusOverdue
	case "REFUNDED", "REFUND_REQUESTED":
		return StatusRefunded
	case "CHARGEBACK_REQUESTED", "CHARGEBACK_DISPUTE":
		return StatusFailed
	case "DELETED":
		return StatusCanceled
	default:
		return StatusPending
	}
}

