package org

import (
	"errors"
	"testing"
)

func TestValidateBranchKind(t *testing.T) {
	cases := []struct {
		name     string
		kind     string
		parentID string
		wantErr  error
	}{
		{"matriz raiz ok", BranchKindMatriz, "", nil},
		{"matriz com superior", BranchKindMatriz, "b1", ErrMatrizComSuperior},
		{"filial raiz ok", BranchKindFilial, "", nil},
		{"filial com superior ok", BranchKindFilial, "b1", nil},
		{"pae sem superior", BranchKindPAE, "", ErrPaeSemSuperior},
		{"pae com superior ok", BranchKindPAE, "b1", nil},
		{"tipo desconhecido", "branch", "b1", ErrBranchKindInvalido},
		{"tipo vazio", "", "", ErrBranchKindInvalido},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateBranchKind(tc.kind, tc.parentID)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("validateBranchKind(%q,%q)=%v, queria %v", tc.kind, tc.parentID, err, tc.wantErr)
			}
		})
	}
}
