package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"chosenerp/internal/auth"
)

// TestAdminOnly cobre o middleware que restringe rotas sensiveis (ex.: acesso do
// membro) a Sede, usada junto com o gate de permissao por rota.
func TestAdminOnly(t *testing.T) {
	app := &App{}
	called := false
	h := app.adminOnly(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	cases := []struct {
		name       string
		role       string
		withClaims bool
		want       int
		wantCalled bool
	}{
		{"sem claims", "", false, http.StatusUnauthorized, false},
		{"super_admin", "super_admin", true, http.StatusOK, true},
		{"admin_sede", "admin_sede", true, http.StatusOK, true},
		{"secretario", "secretario", true, http.StatusForbidden, false},
		{"membro", "membro", true, http.StatusForbidden, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			called = false
			req := httptest.NewRequest(http.MethodGet, "/x", nil)
			if tc.withClaims {
				req = req.WithContext(withClaims(req.Context(), &auth.Claims{Role: tc.role}))
			}
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)
			if rec.Code != tc.want {
				t.Errorf("status = %d, queria %d", rec.Code, tc.want)
			}
			if called != tc.wantCalled {
				t.Errorf("handler chamado = %v, queria %v", called, tc.wantCalled)
			}
		})
	}
}

// TestHasPerm_HQRoleBypass garante que a Sede (super_admin/admin_sede) acessa
// qualquer permissao sem depender de concessao explicita no banco. O short-circuit
// ocorre antes de tocar na transacao, entao um tx nil e suficiente.
func TestHasPerm_HQRoleBypass(t *testing.T) {
	app := &App{}
	for _, role := range []string{"super_admin", "admin_sede"} {
		ok, err := app.hasPerm(context.Background(), nil, &auth.Claims{Role: role}, "modulo.inexistente")
		if err != nil {
			t.Fatalf("role %s: erro inesperado: %v", role, err)
		}
		if !ok {
			t.Errorf("role %s deveria ter a permissao (bypass da Sede)", role)
		}
	}
}
