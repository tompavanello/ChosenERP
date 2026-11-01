package documents

import (
	"strings"
	"testing"
)

// TestRenderReversalReceipt garante que o recibo de estorno e renderizado com
// valor negativo e referencia ao recibo original.
func TestRenderReversalReceipt(t *testing.T) {
	doc := &Document{
		Kind:        KindReversalReceipt,
		Title:       "Recibo de Estorno REC-EST-ABC12345",
		DocumentRef: "REC-EST-ABC12345",
		QRToken:     "token-teste",
		Content: []byte(`{
			"kind":"reversal_receipt",
			"type":"income",
			"amount":300,
			"currency":"BRL",
			"occurred_at":"2031-01-15",
			"reversal_of_ref":"REC-12345678",
			"reversal_reason":"cheque devolvido",
			"donor_name":"Maria da Silva",
			"church_name":"Igreja Teste",
			"description":"Devolucao/estorno: dizimo"
		}`),
	}

	html, err := RenderReceiptHTML(doc, "Igreja Teste")
	if err != nil {
		t.Fatalf("RenderReceiptHTML: %v", err)
	}
	for _, want := range []string{
		"Estorno",
		"REC-EST-ABC12345",
		"REC-12345678",
		"- R$ 300.00",
		"Maria da Silva",
		"cheque devolvido",
	} {
		if !strings.Contains(html, want) {
			t.Errorf("recibo de estorno nao contem %q", want)
		}
	}
}
