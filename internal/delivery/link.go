package delivery

import (
	"strings"

	"chosenerp/internal/documents"
)

// PublicLink monta o link que vai no corpo do e-mail/WhatsApp.
//
// Só a carteirinha tem página pública no webadmin: /member/{token}. Os demais
// documentos não têm rota pública nenhuma, e o formato anterior
// (base + "/" + token) não correspondia a rota alguma do Next — quem clicasse
// caía em 404. Para esses casos devolvemos vazio de propósito: os senders já
// omitem a linha de link quando ela é vazia (senders.go), então o destinatário
// recebe só o código de validação, sem promessa de link quebrado.
//
// Vive aqui, e não duplicado em cada chamador, porque worker.go e
// reports_handlers.go montavam a mesma URL e já haviam divergido do formato
// correto uma vez.
func PublicLink(baseURL, kind, token string) string {
	if kind != documents.KindMembershipCard || token == "" {
		return ""
	}
	return strings.TrimRight(baseURL, "/") + "/member/" + token
}
