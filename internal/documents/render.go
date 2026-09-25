package documents

import (
	"encoding/json"
	"fmt"
	"html"
	"strings"
	"time"
)

// receiptContent espelha o JSON gravado na criacao do recibo (internal/finance/receipt.go).
type receiptContent struct {
	Kind       string  `json:"kind"`
	Ref        string  `json:"ref"`
	TxID       string  `json:"tx_id"`
	BranchID   string  `json:"branch_id"`
	Type       string  `json:"type"`
	Amount     float64 `json:"amount"`
	Currency   string  `json:"currency"`
	OccurredAt string  `json:"occurred_at"`
	Href       string  `json:"href"`
	MemberID   string  `json:"member_id"`
	Member     string  `json:"member"`
}

var typeLabel = map[string]string{
	"income":  "Entrada",
	"expense": "Saida",
}

// RenderReceiptHTML devolve um recibo pronto para impressao/visualizacao (HTML).
func RenderReceiptHTML(doc *Document, tenantName string) (string, error) {
	var c receiptContent
	if len(doc.Content) > 0 {
		_ = json.Unmarshal(doc.Content, &c)
	}
	when := ""
	if t, err := time.Parse(time.RFC3339, c.OccurredAt); err == nil {
		when = t.Format("02/01/2006")
	} else if t, err := time.Parse("2006-01-02", c.OccurredAt); err == nil {
		when = t.Format("02/01/2006")
	} else if c.OccurredAt != "" {
		when = c.OccurredAt
	}
	brl := func(v float64) string {
		return fmt.Sprintf("R$ %.2f", v)
	}
	label := typeLabel[c.Type]
	if label == "" {
		label = html.EscapeString(c.Type)
	}
	if c.Kind == KindMembershipCard {
		return fmt.Sprintf(receiptTemplate,
			html.EscapeString(tenantName),
			html.EscapeString(doc.Title),
			"Carteirinha de Membro",
			html.EscapeString(c.Member),
			html.EscapeString(doc.DocumentRef),
			when, "-", strings.Repeat("-", 24),
			html.EscapeString(doc.QRToken),
		), nil
	}
	return fmt.Sprintf(receiptTemplate,
		html.EscapeString(tenantName),
		html.EscapeString(doc.Title),
		label,
		"Contribuicao/dizimo",
		html.EscapeString(doc.DocumentRef),
		when, brl(c.Amount), strings.Repeat("-", 24),
		html.EscapeString(doc.QRToken),
	), nil
}

const receiptTemplate = `<!doctype html>
<html lang="pt-BR">
<head>
<meta charset="utf-8" />
<meta name="viewport" content="width=device-width, initial-scale=1" />
<title>%[2]s</title>
<style>
  :root { color-scheme: light; }
  body { margin:0; padding:32px; background:#f7f8fb; font-family: ui-sans-serif, system-ui, -apple-system, "Segoe UI", sans-serif; color:#0b1020; }
  .sheet { max-width:420px; margin:0 auto; background:#fff; border:1px solid #e4e7ee; border-radius:12px; padding:28px; box-shadow:0 1px 3px rgba(0,0,0,.05); }
  .brand { display:flex; align-items:center; gap:10px; margin-bottom:18px; }
  .dot { width:34px; height:34px; border-radius:9px; background:#6d28d9; color:#fff; display:flex; align-items:center; justify-content:center; font-weight:700; }
  h1 { font-size:16px; margin:0; letter-spacing:.02em; }
  .sub { font-size:12px; color:#6b7280; margin:0; }
  hr { border:0; border-top:1px dashed #d1d5db; margin:16px 0; }
  .row { display:flex; justify-content:space-between; font-size:13px; padding:4px 0; }
  .label { color:#6b7280; }
  .amount { font-size:22px; font-weight:600; text-align:center; margin:14px 0 4px; }
  .kind { text-align:center; font-size:12px; color:#6b7280; }
  .sep { text-align:center; color:#cbd5e1; letter-spacing:2px; font-size:12px; }
  .foot { font-size:11px; color:#9ca3af; text-align:center; margin-top:12px; }
  @media print { body { background:#fff; padding:0; } .sheet { border:0; box-shadow:none; } }
</style>
</head>
<body>
  <div class="sheet">
    <div class="brand">
      <div class="dot">C</div>
      <div>
        <h1>%[1]s</h1>
        <p class="sub">%[2]s</p>
      </div>
    </div>
    <hr />
    <div class="row"><span class="label">Tipo</span><span>%[3]s</span></div>
    <div class="row"><span class="label">Descricao</span><span>%[4]s</span></div>
    <div class="row"><span class="label">Referencia</span><span>%[5]s</span></div>
    <div class="row"><span class="label">Data</span><span>%[6]s</span></div>
    <hr />
    <div class="amount">%[7]s</div>
    <div class="kind">%[3]s</div>
    <hr />
    <div class="sep">%[8]s</div>
    <p class="foot">Codigo de validacao: %[9]s<br/>Emitido via Chosen ERP</p>
  </div>
</body>
</html>`
