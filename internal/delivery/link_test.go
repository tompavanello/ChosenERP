package delivery

import "testing"

// TestPublicLink cobre a montagem do link que vai no corpo do e-mail/WhatsApp.
//
// O bug que originou este teste: os dois chamadores montavam
// `AppBaseURL + "/" + token` por conta propria, formato que nao corresponde a
// rota nenhuma do webadmin (as rotas sao `/`, `/dashboard/*` e `/member/[token]`)
// - quem clicasse caia em 404. So a carteirinha tem pagina publica; para os
// demais documentos o link tem de sair vazio, porque os senders omitem a linha
// quando e vazia (senders.go:229,368,425).
func TestPublicLink(t *testing.T) {
	casos := []struct {
		nome    string
		baseURL string
		kind    string
		token   string
		quero   string
	}{
		{"carteirinha monta /member/{token}", "https://chosenerp.mgmconsultoria.com", "membership_card", "abc123", "https://chosenerp.mgmconsultoria.com/member/abc123"},
		{"barra final da base nao duplica", "https://chosenerp.mgmconsultoria.com/", "membership_card", "abc123", "https://chosenerp.mgmconsultoria.com/member/abc123"},
		{"base local tambem funciona", "http://localhost:38080", "membership_card", "abc123", "http://localhost:38080/member/abc123"},
		{"recibo nao tem pagina publica", "https://chosenerp.mgmconsultoria.com", "receipt", "abc123", ""},
		{"kind vazio nao gera link", "https://chosenerp.mgmconsultoria.com", "", "abc123", ""},
		{"token vazio nao gera link", "https://chosenerp.mgmconsultoria.com", "membership_card", "", ""},
	}
	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			if got := PublicLink(c.baseURL, c.kind, c.token); got != c.quero {
				t.Errorf("PublicLink(%q, %q, %q) = %q; queria %q",
					c.baseURL, c.kind, c.token, got, c.quero)
			}
		})
	}
}
