package org

import (
	"errors"
	"testing"
)

func TestValidTenantSlug(t *testing.T) {
	cases := []struct {
		in      string
		want    string
		wantErr error
	}{
		{"demo", "demo", nil},
		{"Igreja-Central", "igreja-central", nil},
		{"  matriz  ", "matriz", nil},
		{"a", "", ErrTenantSlugInvalido},          // curto demais
		{"-abc", "", ErrTenantSlugInvalido},       // comeca com hifen
		{"abc-", "", ErrTenantSlugInvalido},       // termina com hifen
		{"com espaco", "", ErrTenantSlugInvalido}, // espaco
		{"caf\u00e9", "", ErrTenantSlugInvalido},   // acento
		{"www", "", ErrTenantSlugReservado},
		{"app", "", ErrTenantSlugReservado},
	}
	for _, tc := range cases {
		got, err := validTenantSlug(tc.in)
		if !errors.Is(err, tc.wantErr) {
			t.Errorf("validTenantSlug(%q) err=%v, queria %v", tc.in, err, tc.wantErr)
			continue
		}
		if tc.wantErr == nil && got != tc.want {
			t.Errorf("validTenantSlug(%q)=%q, queria %q", tc.in, got, tc.want)
		}
	}
}
