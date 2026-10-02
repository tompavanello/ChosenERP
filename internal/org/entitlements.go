package org

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5"
)

// Feature e um modulo gateavel por plano. O catalogo e fixo (validado em
// codigo); modulos core (people/admin/auth/app do membro) NAO entram aqui e
// nunca sao bloqueados.
type Feature struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	Group string `json:"group"`
}

// FeatureCatalog e a fonte unica das features gateaveis.
var FeatureCatalog = []Feature{
	{Key: "finance", Label: "Financeiro", Group: "Financeiro"},
	{Key: "reports", Label: "Relatorios", Group: "Financeiro"},
	{Key: "governance", Label: "Governanca", Group: "Organizacao"},
	{Key: "ministries", Label: "Ministerios e grupos", Group: "Organizacao"},
	{Key: "rosters", Label: "Escalas", Group: "Organizacao"},
	{Key: "events", Label: "Eventos e programacao", Group: "Organizacao"},
	{Key: "kids", Label: "Ministerio infantil (Kids)", Group: "Organizacao"},
	{Key: "prayer", Label: "Pedidos de oracao", Group: "Comunidade"},
	{Key: "whatsapp", Label: "WhatsApp e comunicados", Group: "Comunicacao"},
}

// Limits sao os tetos do plano/igreja (nil = ilimitado).
type Limits struct {
	MaxMembers   *int `json:"max_members"`
	MaxBranches  *int `json:"max_branches"`
	MaxUsers     *int `json:"max_users"`
	MaxStorageMB *int `json:"max_storage_mb"`
}

// Entitlements e o pacote efetivo de uma igreja: plano, features habilitadas,
// limites e uso atual.
type Entitlements struct {
	Plan     string      `json:"plan"`
	Features []string    `json:"features"`
	Limits   Limits      `json:"limits"`
	Usage    TenantUsage `json:"usage"`
}

// Has informa se a feature esta habilitada.
func (e *Entitlements) Has(key string) bool {
	for _, f := range e.Features {
		if f == key {
			return true
		}
	}
	return false
}

func applyFeatureMap(raw string, enabled map[string]bool) {
	if raw == "" {
		return
	}
	var m map[string]any
	if json.Unmarshal([]byte(raw), &m) != nil {
		return
	}
	for k, v := range m {
		if b, ok := v.(bool); ok {
			enabled[k] = b
		}
	}
}

// EffectiveForTenant resolve as features e limites efetivos: a matriz do plano
// sobreposta pelo override da igreja (`tenants.features`/`tenants.limits`).
// Chave ausente = habilitada.
func (r *Repo) EffectiveForTenant(ctx context.Context, tx pgx.Tx, tenantID string) (*Entitlements, error) {
	var plan, planRaw, tenantRaw string
	var l Limits
	var limitsRaw *string
	err := tx.QueryRow(ctx, `
		SELECT COALESCE(t.plan, ''), COALESCE(p.features::text, '{}'),
		       COALESCE(t.features::text, '{}'),
		       p.max_members, p.max_branches, p.max_users, p.max_storage_mb,
		       t.limits::text
		FROM tenants t
		LEFT JOIN plans p ON p.key = t.plan
		WHERE t.id = $1::uuid`, tenantID).
		Scan(&plan, &planRaw, &tenantRaw, &l.MaxMembers, &l.MaxBranches, &l.MaxUsers, &l.MaxStorageMB, &limitsRaw)
	if err != nil {
		return nil, err
	}

	enabled := make(map[string]bool, len(FeatureCatalog))
	for _, f := range FeatureCatalog {
		enabled[f.Key] = true // default: habilitada
	}
	applyFeatureMap(planRaw, enabled)
	applyFeatureMap(tenantRaw, enabled)
	feats := []string{}
	for _, f := range FeatureCatalog {
		if enabled[f.Key] {
			feats = append(feats, f.Key)
		}
	}

	if limitsRaw != nil && *limitsRaw != "" {
		var ov map[string]*int
		if json.Unmarshal([]byte(*limitsRaw), &ov) == nil {
			if v, ok := ov["max_members"]; ok {
				l.MaxMembers = v
			}
			if v, ok := ov["max_branches"]; ok {
				l.MaxBranches = v
			}
			if v, ok := ov["max_users"]; ok {
				l.MaxUsers = v
			}
			if v, ok := ov["max_storage_mb"]; ok {
				l.MaxStorageMB = v
			}
		}
	}

	usage, err := r.TenantUsage(ctx, tx, tenantID)
	if err != nil {
		return nil, err
	}
	return &Entitlements{Plan: plan, Features: feats, Limits: l, Usage: *usage}, nil
}

// EffectiveForTenant e o atalho sem estado (para pacotes que nao tem um Repo).
func EffectiveForTenant(ctx context.Context, tx pgx.Tx, tenantID string) (*Entitlements, error) {
	return (&Repo{}).EffectiveForTenant(ctx, tx, tenantID)
}
