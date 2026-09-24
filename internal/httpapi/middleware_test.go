package httpapi

import (
	"net/http"
	"testing"

	"chosenerp/internal/auth"
)

func TestApplyBranchContext(t *testing.T) {
	cases := []struct {
		name    string
		role    string
		header  string
		initial string
		want    string
	}{
		{"sem header mantem escopo do token", "super_admin", "", "b1", "b1"},
		{"all volta para a Sede", "super_admin", "all", "b1", ""},
		{"uuid troca a filial", "admin_sede", "11111111-1111-1111-1111-111111111111", "b1", "11111111-1111-1111-1111-111111111111"},
		{"valor invalido e ignorado", "super_admin", "nao-e-uuid", "b1", "b1"},
		{"papel de filial nao troca", "secretario", "all", "b1", "b1"},
		{"papel de filial ignora uuid", "tesoureiro", "11111111-1111-1111-1111-111111111111", "b1", "b1"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req, _ := http.NewRequest(http.MethodGet, "/", nil)
			if tc.header != "" {
				req.Header.Set("X-Branch-Id", tc.header)
			}
			c := &auth.Claims{TenantID: "t1", BranchID: tc.initial, Role: tc.role, Type: auth.AccessTokenType}
			applyBranchContext(req, c)
			if c.BranchID != tc.want {
				t.Fatalf("BranchID=%q, queria %q", c.BranchID, tc.want)
			}
		})
	}
}
