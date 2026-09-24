package auth

import (
	"strings"
	"testing"
	"time"
)

func TestTOTP_RoundTrip(t *testing.T) {
	secret, err := GenerateTOTPSecret()
	if err != nil {
		t.Fatalf("gerar segredo: %v", err)
	}
	code := totpAt(secret, time.Now().Unix())
	if !ValidateTOTP(secret, code) {
		t.Errorf("codigo atual deveria validar: %s", code)
	}
	if ValidateTOTP(secret, "000000") && code != "000000" {
		t.Errorf("codigo errado nao deveria validar")
	}
	if ValidateTOTP(secret, "abc") {
		t.Errorf("codigo com tamanho errado nao deveria validar")
	}
}

func TestTOTP_AcceptPreviousWindow(t *testing.T) {
	secret, _ := GenerateTOTPSecret()
	prev := totpAt(secret, time.Now().Unix()-int64(totpPeriod))
	if !ValidateTOTP(secret, prev) {
		t.Errorf("janela anterior deveria ser aceita (tolerancia de relogio)")
	}
}

func TestTOTPURL(t *testing.T) {
	u := TOTPURL("ABC234", "user@igreja.local", "Chosen ERP")
	if !strings.HasPrefix(u, "otpauth://totp/") {
		t.Fatalf("URL inesperada: %s", u)
	}
	if !strings.Contains(u, "secret=ABC234") || !strings.Contains(u, "issuer=Chosen") {
		t.Fatalf("URL sem os parametros esperados: %s", u)
	}
}
