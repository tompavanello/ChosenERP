package httpapi

import (
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5"

	"chosenerp/internal/cargos"
	"chosenerp/internal/store"
)

// errCargoEmUso sinaliza 409 (conflito): o cargo tem historico de mandato.
var errCargoEmUso = errors.New("cargo em uso")

// ---- Catalogo de cargos (requisito CAD100 1.3) ----

func (a *App) handleListCargos(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	b := boundsFromClaims(claims)
	var out []cargos.Cargo
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		var err error
		out, err = a.Cargos.List(r.Context(), tx)
		return err
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"cargos": out})
}

func (a *App) handleCreateCargo(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	var in cargos.CreateInput
	if err := readJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	if in.Name == "" {
		writeErr(w, http.StatusBadRequest, "name required")
		return
	}
	if in.Kind != "" && !cargos.Kinds[in.Kind] {
		writeErr(w, http.StatusBadRequest, "kind invalido")
		return
	}
	b := boundsFromClaims(claims)
	var c *cargos.Cargo
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		var err error
		// branch_id vazio cria um cargo GLOBAL do tenant (catalogo da Sede),
		// enxergado por todas as filiais - e o caso comum: "criar o cargo uma
		// vez e usar em toda a igreja".
		c, err = a.Cargos.Create(r.Context(), tx, claims.TenantID, claims.BranchID, in)
		return err
	})
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, c)
}

func (a *App) handleUpdateCargo(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	id := r.PathValue("id")
	var in cargos.UpdateInput
	if err := readJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	if in.Kind != nil && !cargos.Kinds[*in.Kind] {
		writeErr(w, http.StatusBadRequest, "kind invalido")
		return
	}
	b := boundsFromClaims(claims)
	var c *cargos.Cargo
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		var err error
		c, err = a.Cargos.Update(r.Context(), tx, id, in)
		return err
	})
	if err != nil {
		if store.IsNotFound(err) {
			writeErr(w, http.StatusNotFound, "cargo not found")
			return
		}
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, c)
}

// handleDeleteCargo responde 409 quando o cargo ja tem mandatos.
// Apagar o cargo levaria junto o historico de quem o exerceu, e o requisito 1.4
// existe justamente para preservar esse historico. A saida correta para "nao uso
// mais este cargo" e desativa-lo (PATCH {"is_active": false}).
func (a *App) handleDeleteCargo(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	id := r.PathValue("id")
	b := boundsFromClaims(claims)
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		n, err := a.Cargos.InUse(r.Context(), tx, id)
		if err != nil {
			return err
		}
		if n > 0 {
			return errCargoEmUso
		}
		return a.Cargos.Delete(r.Context(), tx, id)
	})
	if err != nil {
		switch {
		case err == errCargoEmUso:
			writeErr(w, http.StatusConflict, "cargo em uso por mandatos; desative-o em vez de excluir")
		case store.IsNotFound(err):
			writeErr(w, http.StatusNotFound, "cargo not found")
		default:
			writeErr(w, http.StatusInternalServerError, err.Error())
		}
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// ---- Mandatos do membro (requisito CAD100 1.4) ----

func (a *App) handleListMemberCargos(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	memberID := r.PathValue("id")
	b := boundsFromClaims(claims)
	var out []cargos.MemberCargo
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		var err error
		out, err = a.Cargos.ListByMember(r.Context(), tx, memberID)
		return err
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"cargos": out})
}

func (a *App) handleAssignCargo(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	memberID := r.PathValue("id")
	var in cargos.AssignInput
	if err := readJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	if in.CargoID == "" {
		writeErr(w, http.StatusBadRequest, "cargo_id required")
		return
	}
	if in.Status != "" && !cargos.StatusValidos[in.Status] {
		writeErr(w, http.StatusBadRequest, "status invalido")
		return
	}
	b := boundsFromClaims(claims)
	var mc *cargos.MemberCargo
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		var err error
		mc, err = a.Cargos.Assign(r.Context(), tx, memberID, in)
		return err
	})
	if err != nil {
		if store.IsNotFound(err) {
			writeErr(w, http.StatusNotFound, "member or cargo not found")
			return
		}
		// ON CONFLICT DO NOTHING nao devolve linha: mandato identico ja existe.
		if err == pgx.ErrNoRows {
			writeErr(w, http.StatusConflict, "este membro ja tem este cargo no mesmo mandato")
			return
		}
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, mc)
}

func (a *App) handleUpdateMemberCargo(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	memberID := r.PathValue("id")
	linkID := r.PathValue("linkId")
	var in cargos.UpdateAssignmentInput
	if err := readJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	if in.Status != nil && !cargos.StatusValidos[*in.Status] {
		writeErr(w, http.StatusBadRequest, "status invalido")
		return
	}
	b := boundsFromClaims(claims)
	var mc *cargos.MemberCargo
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		var err error
		mc, err = a.Cargos.UpdateAssignment(r.Context(), tx, memberID, linkID, in)
		return err
	})
	if err != nil {
		if store.IsNotFound(err) {
			writeErr(w, http.StatusNotFound, "mandato not found")
			return
		}
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, mc)
}

func (a *App) handleUnassignCargo(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	memberID := r.PathValue("id")
	linkID := r.PathValue("linkId")
	b := boundsFromClaims(claims)
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		return a.Cargos.Unassign(r.Context(), tx, memberID, linkID)
	})
	if err != nil {
		if store.IsNotFound(err) {
			writeErr(w, http.StatusNotFound, "mandato not found")
			return
		}
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}
