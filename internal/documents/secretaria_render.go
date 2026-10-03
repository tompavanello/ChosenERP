package documents

import (
	"encoding/json"
	"fmt"
	"html"
)

// secretaryContent e o JSON gravado para certificados/cartas.
type secretaryContent struct {
	Kind         string `json:"kind"`
	Type         string `json:"type"`
	Member       string `json:"member"`
	MemberDoc    string `json:"member_doc"`
	BirthDate    string `json:"birth_date"`
	BaptismDate  string `json:"baptism_date"`
	MarriageDate string `json:"marriage_date"`
	Spouse       string `json:"spouse"`
	Church       string `json:"church"`
	ChurchDoc    string `json:"church_doc"`
	City         string `json:"city"`
	IssuedAt     string `json:"issued_at"`
	Body         string `json:"body"`
	Ref          string `json:"ref"`
}

var certTitles = map[string]string{
	"baptism":      "Certificado de Batismo",
	"marriage":     "Certificado de Casamento",
	"presentation": "Certificado de Apresentação de Criança",
}

var letterTitles = map[string]string{
	"transfer":       "Carta de Transferência",
	"recommendation": "Carta de Recomendação",
}

func decodeContent(doc *Document) secretaryContent {
	var c secretaryContent
	if len(doc.Content) > 0 {
		_ = json.Unmarshal(doc.Content, &c)
	}
	return c
}

// DocTitle devolve o titulo legivel de um documento de secretaria.
func DocTitle(kind, docType string) string {
	switch kind {
	case KindCertificate:
		if t := certTitles[docType]; t != "" {
			return t
		}
		return "Certificado"
	case KindLetter:
		if t := letterTitles[docType]; t != "" {
			return t
		}
		return "Carta"
	}
	return "Documento"
}

func docFooter(c secretaryContent, token string) string {
	ref := html.EscapeString(c.Ref)
	return fmt.Sprintf(`<p class="foot">Documento emitido eletronicamente via Chosen ERP<br/>Referência %s · Código de validação: %s</p>`, ref, html.EscapeString(token))
}

// RenderCertificateHTML devolve o certificado (batismo/casamento/apresentacao).
func RenderCertificateHTML(doc *Document, tenantName string) (string, error) {
	c := decodeContent(doc)
	title := certTitles[c.Type]
	if title == "" {
		title = "Certificado"
	}
	church := c.Church
	if church == "" {
		church = tenantName
	}
	cnpj := ""
	if c.ChurchDoc != "" {
		cnpj = " - CNPJ " + c.ChurchDoc
	}
	body := c.Body
	if body == "" {
		body = fmt.Sprintf("Certificamos, para os devidos fins, que <strong>%s</strong> faz parte desta igreja.", html.EscapeString(c.Member))
	}
	if c.MemberDoc != "" {
		body += fmt.Sprintf("<br/><span class=\"doc\">Documento: %s</span>", html.EscapeString(c.MemberDoc))
	}
	if c.Spouse != "" {
		body += fmt.Sprintf("<br/><span class=\"doc\">Cônjuge: %s</span>", html.EscapeString(c.Spouse))
	}
	return fmt.Sprintf(certTemplate,
		html.EscapeString(church),
		html.EscapeString(city(c)),
		html.EscapeString(title),
		html.EscapeString(c.Member),
		body,
		html.EscapeString(c.IssuedAt),
		html.EscapeString(church+cnpj),
		docFooter(c, doc.QRToken),
	), nil
}

// RenderLetterHTML devolve a carta (transferencia/recomendacao).
func RenderLetterHTML(doc *Document, tenantName string) (string, error) {
	c := decodeContent(doc)
	title := letterTitles[c.Type]
	if title == "" {
		title = "Carta"
	}
	church := c.Church
	if church == "" {
		church = tenantName
	}
	if c.ChurchDoc != "" {
		church += " - CNPJ " + c.ChurchDoc
	}
	body := c.Body
	if body == "" {
		body = "A paz do Senhor.<br/><br/>" +
			fmt.Sprintf("Encaminhamos <strong>%s</strong>, membro desta igreja, a quem esperamos receber com a mesma comunhão que nos une em Cristo.", html.EscapeString(c.Member))
	}
	return fmt.Sprintf(letterTemplate,
		html.EscapeString(church),
		html.EscapeString(city(c)),
		html.EscapeString(c.IssuedAt),
		html.EscapeString(title),
		html.EscapeString(c.Member),
		body,
		docFooter(c, doc.QRToken),
	), nil
}

func city(c secretaryContent) string {
	if c.City != "" {
		return c.City
	}
	return ""
}

const certTemplate = `<!doctype html>
<html lang="pt-BR"><head><meta charset="utf-8"/><meta name="viewport" content="width=device-width, initial-scale=1"/>
<title>%[3]s</title>
<style>
  body{margin:0;padding:32px;background:#f7f8fb;font-family:ui-serif,Georgia,"Times New Roman",serif;color:#0b1020}
  .sheet{max-width:720px;margin:0 auto;background:#fff;border:1px solid #e4e7ee;border-radius:12px;padding:44px 48px;box-shadow:0 1px 3px rgba(0,0,0,.06)}
  .church{font-family:ui-sans-serif,system-ui,sans-serif;font-size:12px;letter-spacing:.14em;text-transform:uppercase;color:#6b7280;text-align:center}
  h1{font-size:26px;text-align:center;margin:6px 0 2px}
  .city{text-align:center;color:#6b7280;font-size:13px;margin-bottom:26px}
  .body{font-size:15px;line-height:1.9;text-align:justify}
  .body .doc{color:#6b7280;font-size:13px}
  .sig{margin-top:56px;display:flex;justify-content:space-around;gap:32px}
  .sig div{border-top:1px solid #9ca3af;padding-top:6px;font-size:12px;color:#6b7280;min-width:220px;text-align:center}
  .foot{font-family:ui-sans-serif,system-ui,sans-serif;font-size:11px;color:#9ca3af;text-align:center;margin-top:34px}
  @media print{body{background:#fff;padding:0}.sheet{border:0;box-shadow:none}}
</style></head>
<body><div class="sheet">
  <div class="church">%[1]s</div>
  <h1>%[3]s</h1>
  <div class="city">%[2]s</div>
  <div class="body">%[5]s</div>
  <div class="sig"><div>Secretaria</div><div>Pastor / Presidente</div></div>
  <div class="city" style="margin-top:28px">%[2]s, %[6]s · %[7]s</div>
  %[8]s
</div></body></html>`

const letterTemplate = `<!doctype html>
<html lang="pt-BR"><head><meta charset="utf-8"/><meta name="viewport" content="width=device-width, initial-scale=1"/>
<title>%[4]s</title>
<style>
  body{margin:0;padding:32px;background:#f7f8fb;font-family:ui-serif,Georgia,"Times New Roman",serif;color:#0b1020}
  .sheet{max-width:720px;margin:0 auto;background:#fff;border:1px solid #e4e7ee;border-radius:12px;padding:44px 48px;box-shadow:0 1px 3px rgba(0,0,0,.06)}
  .head{text-align:center;border-bottom:2px solid #0b1020;padding-bottom:10px;margin-bottom:24px}
  .church{font-family:ui-sans-serif,system-ui,sans-serif;font-size:14px;font-weight:700;letter-spacing:.04em}
  .local{font-size:13px;color:#6b7280;text-align:right;margin:0 0 22px}
  h1{font-size:18px;text-align:center;margin:0 0 22px}
  .body{font-size:15px;line-height:1.9;text-align:justify}
  .sig{margin-top:56px}
  .sig div{border-top:1px solid #9ca3af;padding-top:6px;font-size:12px;color:#6b7280;max-width:280px}
  .foot{font-family:ui-sans-serif,system-ui,sans-serif;font-size:11px;color:#9ca3af;text-align:center;margin-top:34px}
  @media print{body{background:#fff;padding:0}.sheet{border:0;box-shadow:none}}
</style></head>
<body><div class="sheet">
  <div class="head"><div class="church">%[1]s</div></div>
  <div class="local">%[2]s, %[3]s</div>
  <h1>%[4]s</h1>
  <div class="body">%[6]s</div>
  <div class="sig"><div>Secretaria / Pastor</div></div>
  %[7]s
</div></body></html>`
