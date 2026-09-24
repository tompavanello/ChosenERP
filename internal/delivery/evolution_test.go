package delivery

import "testing"

func TestNormalizeEvolutionState(t *testing.T) {
	cases := map[string]string{
		"open":       "connected",
		"connected":  "connected",
		"CONNECTING": "connecting",
		"pairing":    "connecting",
		"close":      "disconnected",
		"":           "disconnected",
		"qualquer":   "disconnected",
	}
	for in, want := range cases {
		if got := normalizeEvolutionState(in); got != want {
			t.Errorf("normalizeEvolutionState(%q)=%q, queria %q", in, got, want)
		}
	}
}

func TestParseQRVariants(t *testing.T) {
	// v1: base64 no topo.
	if qr := parseQR([]byte(`{"code":200,"base64":"AAA"}`)); qr.Base64 != "AAA" {
		t.Errorf("v1 base64=%q", qr.Base64)
	}
	// v2: aninhado em qrcode.
	qr := parseQR([]byte(`{"qrcode":{"base64":"BBB","code":"2@abc"}}`))
	if qr.Base64 != "BBB" || qr.Code != "2@abc" {
		t.Errorf("v2 qr=%+v", qr)
	}
}
