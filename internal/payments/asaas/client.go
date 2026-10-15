// Package asaas implementa o provedor de pagamentos Asaas.
//
// Docs: https://docs.asaas.com/
// Autorizacao: header `access_token`.
// Sandbox: https://sandbox.asaas.com/api/v3
// Producao: https://api.asaas.com/v3
package asaas

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"chosenerp/internal/payments"
)

// ErrNotConfigured e devolvido quando a API key nao esta definida.
var ErrNotConfigured = fmt.Errorf("asaas: API key nao configurada")

// Client chama a API do Asaas.
type Client struct {
	BaseURL string
	APIKey  string
	HTTP    *http.Client
}

// New cria um cliente Asaas. baseURL vazio usa producao.
func New(baseURL, apiKey string) *Client {
	if baseURL == "" {
		baseURL = "https://api.asaas.com/v3"
	}
	return &Client{
		BaseURL: strings.TrimRight(baseURL, "/"),
		APIKey:  apiKey,
		HTTP:    &http.Client{Timeout: 30 * time.Second},
	}
}

// Provider identifica o provedor.
func (c *Client) Provider() string { return "asaas" }

// ErrAPI representa um erro retornado pelo Asaas.
type ErrAPI struct {
	StatusCode  int
	Code        string
	Description string
}

func (e *ErrAPI) Error() string {
	if e.Description != "" {
		return fmt.Sprintf("asaas (%d): %s", e.StatusCode, e.Description)
	}
	return fmt.Sprintf("asaas (%d)", e.StatusCode)
}

type apiError struct {
	Errors []struct {
		Code        string `json:"code"`
		Description string `json:"description"`
	} `json:"errors"`
}

func (c *Client) do(ctx context.Context, method, path string, body any, out any) error {
	if c.APIKey == "" {
		return ErrNotConfigured
	}
	var reader io.Reader
	if body != nil {
		buf, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(buf)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, reader)
	if err != nil {
		return err
	}
	req.Header.Set("access_token", c.APIKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "ChosenERP/1.0")

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var apiErr apiError
		_ = json.Unmarshal(raw, &apiErr)
		e := &ErrAPI{StatusCode: resp.StatusCode}
		if len(apiErr.Errors) > 0 {
			e.Code = apiErr.Errors[0].Code
			e.Description = apiErr.Errors[0].Description
		} else {
			e.Description = strings.TrimSpace(string(raw))
		}
		return e
	}
	if out != nil && len(raw) > 0 {
		return json.Unmarshal(raw, out)
	}
	return nil
}

// ---- Customers ----

type customerDTO struct {
	ID                string `json:"id"`
	Name              string `json:"name"`
	Email             string `json:"email"`
	Phone             string `json:"phone"`
	CPFCNPJ           string `json:"cpfCnpj"`
	ExternalReference string `json:"externalReference"`
}

// CreateCustomer cria um cliente no Asaas.
func (c *Client) CreateCustomer(ctx context.Context, in payments.CustomerInput) (*payments.Customer, error) {
	body := map[string]any{
		"name":              in.Name,
		"externalReference": in.ExternalReference,
	}
	if in.Email != "" {
		body["email"] = in.Email
	}
	if in.Phone != "" {
		body["phone"] = in.Phone
	}
	if in.CPFCNPJ != "" {
		body["cpfCnpj"] = onlyDigits(in.CPFCNPJ)
	}
	var out customerDTO
	if err := c.do(ctx, http.MethodPost, "/customers", body, &out); err != nil {
		return nil, err
	}
	return customerFromDTO(out), nil
}

// UpdateCustomer atualiza um cliente existente.
func (c *Client) UpdateCustomer(ctx context.Context, externalID string, in payments.CustomerInput) (*payments.Customer, error) {
	body := map[string]any{"name": in.Name}
	if in.Email != "" {
		body["email"] = in.Email
	}
	if in.Phone != "" {
		body["phone"] = in.Phone
	}
	if in.CPFCNPJ != "" {
		body["cpfCnpj"] = onlyDigits(in.CPFCNPJ)
	}
	var out customerDTO
	if err := c.do(ctx, http.MethodPut, "/customers/"+externalID, body, &out); err != nil {
		return nil, err
	}
	return customerFromDTO(out), nil
}

func customerFromDTO(d customerDTO) *payments.Customer {
	return &payments.Customer{
		ExternalID: d.ID, Name: d.Name, Email: d.Email, Phone: d.Phone, CPFCNPJ: d.CPFCNPJ,
	}
}

// ---- Subscriptions ----

type subscriptionDTO struct {
	ID          string `json:"id"`
	Status      string `json:"status"`
	Value       float64 `json:"value"`
	Cycle       string `json:"cycle"`
	BillingType string `json:"billingType"`
	NextDueDate string `json:"nextDueDate"`
}

func subscriptionFromDTO(d subscriptionDTO) *payments.Subscription {
	return &payments.Subscription{
		ExternalID: d.ID, Status: normStatus(d.Status),
		ValueCents: int64(d.Value*100 + 0.5), Cycle: d.Cycle,
		BillingType: d.BillingType, NextDueDate: d.NextDueDate,
	}
}

// CreateSubscription cria uma assinatura recorrente.
func (c *Client) CreateSubscription(ctx context.Context, in payments.SubscriptionInput) (*payments.Subscription, error) {
	body := map[string]any{
		"customer":          in.CustomerExternalID,
		"billingType":       in.BillingType,
		"value":             centsToFloat(in.ValueCents),
		"nextDueDate":       in.NextDueDate,
		"cycle":             in.Cycle,
		"description":       in.Description,
		"externalReference": in.ExternalReference,
	}
	if in.Card != nil {
		body["creditCard"] = cardBody(in.Card)
		body["creditCardHolderInfo"] = cardHolderBody(in.Card)
	}
	var out subscriptionDTO
	if err := c.do(ctx, http.MethodPost, "/subscriptions", body, &out); err != nil {
		return nil, err
	}
	return subscriptionFromDTO(out), nil
}

// UpdateSubscription atualiza uma assinatura existente.
func (c *Client) UpdateSubscription(ctx context.Context, externalID string, in payments.SubscriptionInput) (*payments.Subscription, error) {
	body := map[string]any{
		"billingType": in.BillingType,
		"value":       centsToFloat(in.ValueCents),
		"cycle":       in.Cycle,
		"description": in.Description,
	}
	if in.NextDueDate != "" {
		body["nextDueDate"] = in.NextDueDate
	}
	if in.Card != nil {
		body["creditCard"] = cardBody(in.Card)
		body["creditCardHolderInfo"] = cardHolderBody(in.Card)
	}
	var out subscriptionDTO
	if err := c.do(ctx, http.MethodPut, "/subscriptions/"+externalID, body, &out); err != nil {
		return nil, err
	}
	return subscriptionFromDTO(out), nil
}

// CancelSubscription remove a assinatura (nao gera novas cobrancas).
func (c *Client) CancelSubscription(ctx context.Context, externalID string) error {
	return c.do(ctx, http.MethodDelete, "/subscriptions/"+externalID, nil, nil)
}

// GetSubscription busca uma assinatura.
func (c *Client) GetSubscription(ctx context.Context, externalID string) (*payments.Subscription, error) {
	var out subscriptionDTO
	if err := c.do(ctx, http.MethodGet, "/subscriptions/"+externalID, nil, &out); err != nil {
		return nil, err
	}
	return subscriptionFromDTO(out), nil
}

// ---- Payments (cobrancas avulsas) ----

type paymentDTO struct {
	ID                 string  `json:"id"`
	Status             string  `json:"status"`
	BillingType        string  `json:"billingType"`
	Value              float64 `json:"value"`
	NetValue           float64 `json:"netValue"`
	DueDate            string  `json:"dueDate"`
	PaymentDate        string  `json:"paymentDate"`
	InvoiceURL         string  `json:"invoiceUrl"`
	BankSlipURL        string  `json:"bankSlipUrl"`
	TransactionReceipt string  `json:"transactionReceiptUrl"`
}

func paymentFromDTO(d paymentDTO) *payments.Charge {
	return &payments.Charge{
		ExternalID:         d.ID,
		Status:             normStatus(d.Status),
		RawStatus:          d.Status,
		BillingType:        d.BillingType,
		ValueCents:         int64(d.Value*100 + 0.5),
		NetValueCents:      int64(d.NetValue*100 + 0.5),
		DueDate:            d.DueDate,
		PaidAt:             d.PaymentDate,
		InvoiceURL:         d.InvoiceURL,
		BankSlipURL:        d.BankSlipURL,
		TransactionReceipt: d.TransactionReceipt,
	}
}

// CreateCharge cria uma cobranca avulsa.
func (c *Client) CreateCharge(ctx context.Context, in payments.ChargeInput) (*payments.Charge, error) {
	body := map[string]any{
		"customer":          in.CustomerExternalID,
		"billingType":       in.BillingType,
		"value":             centsToFloat(in.ValueCents),
		"dueDate":           in.DueDate,
		"description":       in.Description,
		"externalReference": in.ExternalReference,
	}
	if in.InstallmentCount > 1 {
		body["installmentCount"] = in.InstallmentCount
	}
	if in.Card != nil {
		body["creditCard"] = cardBody(in.Card)
		body["creditCardHolderInfo"] = cardHolderBody(in.Card)
	}
	var out paymentDTO
	if err := c.do(ctx, http.MethodPost, "/payments", body, &out); err != nil {
		return nil, err
	}
	return paymentFromDTO(out), nil
}

// GetCharge busca uma cobranca.
func (c *Client) GetCharge(ctx context.Context, externalID string) (*payments.Charge, error) {
	var out paymentDTO
	if err := c.do(ctx, http.MethodGet, "/payments/"+externalID, nil, &out); err != nil {
		return nil, err
	}
	return paymentFromDTO(out), nil
}

// RefundCharge estorna uma cobranca (total ou parcial).
func (c *Client) RefundCharge(ctx context.Context, externalID string, amountCents *int64) (*payments.Charge, error) {
	body := map[string]any{}
	if amountCents != nil {
		body["value"] = centsToFloat(*amountCents)
	}
	var out paymentDTO
	if err := c.do(ctx, http.MethodPost, "/payments/"+externalID+"/refund", body, &out); err != nil {
		return nil, err
	}
	return paymentFromDTO(out), nil
}

// PixQRCode devolve o QR Code de uma cobranca Pix.
func (c *Client) PixQRCode(ctx context.Context, chargeExternalID string) (*payments.PixQRCode, error) {
	var out struct {
		EncodedImage   string `json:"encodedImage"`
		Payload        string `json:"payload"`
		ExpirationDate string `json:"expirationDate"`
	}
	if err := c.do(ctx, http.MethodGet, "/payments/"+chargeExternalID+"/pixQrCode", nil, &out); err != nil {
		return nil, err
	}
	return &payments.PixQRCode{
		EncodedImage: out.EncodedImage, Payload: out.Payload, ExpirationDate: out.ExpirationDate,
	}, nil
}

// ---- helpers ----

func normStatus(s string) string {
	return payments.NormalizeStatus("asaas", s)
}

func cardBody(c *payments.Card) map[string]any {
	return map[string]any{
		"holderName":  c.HolderName,
		"number":      onlyDigits(c.Number),
		"expiryMonth": c.ExpiryMonth,
		"expiryYear":  c.ExpiryYear,
		"ccv":         c.CCV,
	}
}

func cardHolderBody(c *payments.Card) map[string]any {
	m := map[string]any{
		"name":    c.HolderName,
		"cpfCnpj": onlyDigits(c.HolderCPF),
	}
	if c.PostalCode != "" {
		m["postalCode"] = onlyDigits(c.PostalCode)
	}
	if c.AddressNumber != "" {
		m["addressNumber"] = c.AddressNumber
	}
	return m
}

func onlyDigits(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func centsToFloat(c int64) float64 { return float64(c) / 100.0 }
