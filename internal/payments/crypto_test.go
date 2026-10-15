package payments

import (
	"strings"
	"testing"
)

func TestCredentialsRoundTrip(t *testing.T) {
	key := authKey("minha-chave", "jwt")
	raw := []byte(`{"api_key":"sk_live_123","base_url":"https://api.asaas.com/v3","webhook_token":"tok"}`)
	enc, err := EncryptCredentials(key, raw)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	if strings.Contains(string(enc), "sk_live_123") {
		t.Fatal("credencial em texto puro no ciphertext")
	}
	dec, err := DecryptCredentials(key, enc)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if string(dec) != string(raw) {
		t.Fatalf("round-trip divergente: %s", dec)
	}
}

func TestCredentialsWrongKeyFails(t *testing.T) {
	enc, _ := EncryptCredentials(authKey("a", ""), []byte(`{"api_key":"x"}`))
	if _, err := DecryptCredentials(authKey("b", ""), enc); err == nil {
		t.Fatal("esperava falha ao decifrar com chave errada")
	}
}

func TestDecryptLegacyPlaintext(t *testing.T) {
	// Valor antigo (sem envelope enc) e devolvido como esta.
	got, err := DecryptCredentials(authKey("a", ""), []byte(`{"api_key":"plain"}`))
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if string(got) != `{"api_key":"plain"}` {
		t.Fatalf("legado: %s", got)
	}
}
