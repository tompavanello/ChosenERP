package payments

import (
	"context"
	"testing"
)

func TestVerifyWebhookToken_FailsClosed(t *testing.T) {
	ctx := context.Background()

	// Sem token na requisicao: rejeita.
	s := &Service{Cfg: Config{AsaasWebhookToken: "segredo"}}
	if s.VerifyWebhookToken(ctx, "", "asaas", "") {
		t.Fatal("token vazio deveria ser rejeitado")
	}

	// Sem token esperado (plataforma vazio e sem igreja): rejeita mesmo com token.
	s = &Service{}
	if s.VerifyWebhookToken(ctx, "qualquer", "asaas", "") {
		t.Fatal("sem token esperado deveria ser rejeitado (falha fechado)")
	}

	// Token da plataforma confere.
	s = &Service{Cfg: Config{AsaasWebhookToken: "segredo"}}
	if !s.VerifyWebhookToken(ctx, "segredo", "asaas", "") {
		t.Fatal("token correto deveria ser aceito")
	}
	if s.VerifyWebhookToken(ctx, "errado", "asaas", "") {
		t.Fatal("token incorreto deveria ser rejeitado")
	}
}

func TestTenantFromReference(t *testing.T) {
	cases := map[string]string{
		"tenant:11111111-1111-1111-1111-111111111111":      "11111111-1111-1111-1111-111111111111",
		"tenant:11111111-1111-1111-1111-111111111111:payer": "11111111-1111-1111-1111-111111111111",
		"":           "",
		"tenant:abc": "",
		"outro":      "",
	}
	for in, want := range cases {
		if got := TenantFromReference(in); got != want {
			t.Errorf("TenantFromReference(%q) = %q, want %q", in, got, want)
		}
	}
}
