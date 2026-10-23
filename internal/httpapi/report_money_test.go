package httpapi

import "testing"

func TestMoneyBrazilianFormat(t *testing.T) {
	cases := map[float64]string{
		0:        "0,00",
		1:        "1,00",
		999.9:    "999,90",
		1000:     "1.000,00",
		1234.56:  "1.234,56",
		1234567:  "1.234.567,00",
		-1234.56: "-1.234,56",
	}
	for v, want := range cases {
		if got := money(v); got != want {
			t.Errorf("money(%v) = %q, queria %q", v, got, want)
		}
	}
}
