package asaas

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"chosenerp/internal/payments"
)

func TestCreateCustomerAndCharge(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("access_token")
		switch r.URL.Path {
		case "/customers":
			_ = json.NewEncoder(w).Encode(map[string]any{"id": "cus_1", "name": "Igreja Demo", "email": "a@b.c"})
		case "/payments":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id": "pay_1", "status": "PENDING", "billingType": "PIX",
				"value": 150.0, "netValue": 148.5, "dueDate": "2026-01-10",
				"invoiceUrl": "https://x/inv", "bankSlipUrl": "",
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	c := New(srv.URL, "key-123")
	ctx := context.Background()

	cust, err := c.CreateCustomer(ctx, payments.CustomerInput{Name: "Igreja Demo", Email: "a@b.c", CPFCNPJ: "12.345.678/0001-99"})
	if err != nil {
		t.Fatalf("CreateCustomer: %v", err)
	}
	if cust.ExternalID != "cus_1" || cust.Name != "Igreja Demo" {
		t.Fatalf("customer inesperado: %+v", cust)
	}

	ch, err := c.CreateCharge(ctx, payments.ChargeInput{
		CustomerExternalID: cust.ExternalID, ValueCents: 15000,
		DueDate: "2026-01-10", BillingType: payments.BillingPix,
	})
	if err != nil {
		t.Fatalf("CreateCharge: %v", err)
	}
	if ch.ExternalID != "pay_1" || ch.Status != payments.StatusPending || ch.ValueCents != 15000 {
		t.Fatalf("charge inesperada: %+v", ch)
	}
	if gotAuth != "key-123" {
		t.Fatalf("header access_token: %q", gotAuth)
	}
}

func TestPixQRCodeAndRefund(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/payments/pay_1/pixQrCode":
			_ = json.NewEncoder(w).Encode(map[string]any{"encodedImage": "AAA", "payload": "000201", "expirationDate": "2026-01-10"})
		case "/payments/pay_1/refund":
			_ = json.NewEncoder(w).Encode(map[string]any{"id": "pay_1", "status": "REFUNDED", "value": 100.0})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	c := New(srv.URL, "k")
	qr, err := c.PixQRCode(context.Background(), "pay_1")
	if err != nil || qr.Payload != "000201" || qr.EncodedImage != "AAA" {
		t.Fatalf("pix: %+v err=%v", qr, err)
	}
	ch, err := c.RefundCharge(context.Background(), "pay_1", nil)
	if err != nil || ch.Status != payments.StatusRefunded {
		t.Fatalf("refund: %+v err=%v", ch, err)
	}
}

func TestAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"errors": []map[string]string{{"code": "invalid", "description": "CPF invalido"}},
		})
	}))
	defer srv.Close()

	c := New(srv.URL, "k")
	_, err := c.CreateCustomer(context.Background(), payments.CustomerInput{Name: "x"})
	if err == nil {
		t.Fatal("esperava erro")
	}
	apiErr, ok := err.(*ErrAPI)
	if !ok || apiErr.Description != "CPF invalido" {
		t.Fatalf("erro inesperado: %v", err)
	}
}

func TestNotConfigured(t *testing.T) {
	c := New("http://localhost", "")
	if _, err := c.CreateCustomer(context.Background(), payments.CustomerInput{}); err != ErrNotConfigured {
		t.Fatalf("esperava ErrNotConfigured, veio %v", err)
	}
}

func TestStatusNormalization(t *testing.T) {
	cases := map[string]string{
		"PENDING": payments.StatusPending, "RECEIVED": payments.StatusPaid,
		"CONFIRMED": payments.StatusPaid, "OVERDUE": payments.StatusOverdue,
		"REFUNDED": payments.StatusRefunded, "DELETED": payments.StatusCanceled,
	}
	for raw, want := range cases {
		if got := payments.NormalizeStatus("asaas", raw); got != want {
			t.Errorf("%s => %s, queria %s", raw, got, want)
		}
	}
}
