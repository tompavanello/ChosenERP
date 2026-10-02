package httpapi

import (
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"chosenerp/internal/org"
	"chosenerp/internal/store"
)

// ---- Console da plataforma: igrejas ----

func (a *App) handleAdminGetTenant(w http.ResponseWriter, r *http.Request) {
	if _, ok := a.platformAdminClaims(w, r); !ok {
		return
	}
	id := r.PathValue("id")
	var t *org.Tenant
	err := a.Store.WithSystem(r.Context(), func(tx pgx.Tx) error {
		var e error
		t, e = a.Org.GetTenantAdmin(r.Context(), tx, id)
		return e
	})
	if err != nil {
		if store.IsNotFound(err) {
			writeErr(w, http.StatusNotFound, "igreja nao encontrada")
			return
		}
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, t)
}

func (a *App) handleAdminUpdateTenant(w http.ResponseWriter, r *http.Request) {
	if _, ok := a.platformAdminClaims(w, r); !ok {
		return
	}
	id := r.PathValue("id")
	var in org.AdminTenantInput
	if err := readJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	var t *org.Tenant
	err := a.Store.WithSystem(r.Context(), func(tx pgx.Tx) error {
		var e error
		t, e = a.Org.UpdateTenantAdmin(r.Context(), tx, id, in)
		return e
	})
	if err != nil {
		switch {
		case errors.Is(err, org.ErrTenantSlugInvalido) || errors.Is(err, org.ErrTenantSlugReservado):
			writeErr(w, http.StatusBadRequest, err.Error())
		case store.IsNotFound(err):
			writeErr(w, http.StatusNotFound, "igreja nao encontrada")
		default:
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "23505" {
				writeErr(w, http.StatusConflict, "slug ja esta em uso")
				return
			}
			writeErr(w, http.StatusBadRequest, err.Error())
		}
		return
	}
	writeJSON(w, http.StatusOK, t)
}

func (a *App) handleAdminTenantUsage(w http.ResponseWriter, r *http.Request) {
	if _, ok := a.platformAdminClaims(w, r); !ok {
		return
	}
	id := r.PathValue("id")
	var u *org.TenantUsage
	err := a.Store.WithSystem(r.Context(), func(tx pgx.Tx) error {
		var e error
		u, e = a.Org.TenantUsage(r.Context(), tx, id)
		return e
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, u)
}

// ---- Estatisticas gerais ----

func (a *App) handleAdminStats(w http.ResponseWriter, r *http.Request) {
	if _, ok := a.platformAdminClaims(w, r); !ok {
		return
	}
	var s *org.PlatformStats
	err := a.Store.WithSystem(r.Context(), func(tx pgx.Tx) error {
		var e error
		s, e = a.Org.Stats(r.Context(), tx)
		return e
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, s)
}

// ---- Catalogo de features (modulos gateaveis) ----

func (a *App) handleListFeatures(w http.ResponseWriter, r *http.Request) {
	if _, ok := a.platformAdminClaims(w, r); !ok {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"features": org.FeatureCatalog})
}

// ---- Catalogo de planos ----

func (a *App) handleListPlans(w http.ResponseWriter, r *http.Request) {
	if _, ok := a.platformAdminClaims(w, r); !ok {
		return
	}
	var out []org.Plan
	err := a.Store.WithSystem(r.Context(), func(tx pgx.Tx) error {
		var e error
		out, e = a.Org.ListPlans(r.Context(), tx)
		return e
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if out == nil {
		out = []org.Plan{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"plans": out})
}

func (a *App) handleCreatePlan(w http.ResponseWriter, r *http.Request) {
	if _, ok := a.platformAdminClaims(w, r); !ok {
		return
	}
	var in org.PlanInput
	if err := readJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	var p *org.Plan
	err := a.Store.WithSystem(r.Context(), func(tx pgx.Tx) error {
		var e error
		p, e = a.Org.CreatePlan(r.Context(), tx, in)
		return e
	})
	if err != nil {
		if errors.Is(err, org.ErrPlanInvalido) {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			writeErr(w, http.StatusConflict, "ja existe um plano com esta chave")
			return
		}
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, p)
}

func (a *App) handleUpdatePlan(w http.ResponseWriter, r *http.Request) {
	if _, ok := a.platformAdminClaims(w, r); !ok {
		return
	}
	key := r.PathValue("key")
	var in org.PlanInput
	if err := readJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	var p *org.Plan
	err := a.Store.WithSystem(r.Context(), func(tx pgx.Tx) error {
		var e error
		p, e = a.Org.UpdatePlan(r.Context(), tx, key, in)
		return e
	})
	if err != nil {
		if store.IsNotFound(err) {
			writeErr(w, http.StatusNotFound, "plano nao encontrado")
			return
		}
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, p)
}
