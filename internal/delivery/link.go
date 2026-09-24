package delivery

import (
	"strings"

	"chosenerp/internal/documents"
)

// PublicLink monta o link que vai no corpo do e-mail/WhatsApp.
//
// So a carteirinha tem pagina publica no webadmin: /member/{token}. Os demais
// documentos nao tem rota publica nenhuma, e o formato anterior
// (base + "/" + token) nao correspondia a rota alguma do Next - quem clicasse
// caia em 404. Para esses casos devolvemos vazio de proposito: os senders ja
// omitem a linha de link quando ela e vazia (senders.go), entao o destinatario
// recebe so o codigo de validacao, sem promessa de link quebrado.
//
// Vive aqui, e nao duplicado em cada chamador, porque worker.go e
// reports_handlers.go montavam a mesma URL e ja haviam divergido do formato
// correto uma vez.
func PublicLink(baseURL, kind, token string) string {
	if kind != documents.KindMembershipCard || token == "" {
		return ""
	}
	return strings.TrimRight(baseURL, "/") + "/member/" + token
}
