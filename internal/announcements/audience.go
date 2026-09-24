package announcements

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
)

// AudienceFilter define a segmentacao do disparo em massa (#32).
//
// Membros e visitantes compartilham apenas o filtro de filial; sexo, estado
// civil, faixa etaria e situacao sao atributos exclusivos de `members`. Por
// isso, quando um desses filtros e informado, os visitantes deixam de entrar no
// publico "todos" - nao ha como segmenta-los e inclui-los daria um resultado
// silenciosamente errado.
type AudienceFilter struct {
	GroupIDs           []string `json:"group_ids,omitempty"`
	MinistryIDs        []string `json:"ministry_ids,omitempty"`
	BranchIDs          []string `json:"branch_ids,omitempty"`
	Genders            []string `json:"genders,omitempty"`
	MaritalStatuses    []string `json:"marital_statuses,omitempty"`
	MembershipStatuses []string `json:"membership_statuses,omitempty"`
	AgeMin             *int     `json:"age_min,omitempty"`
	AgeMax             *int     `json:"age_max,omitempty"`
}

// memberOnly indica que ha filtro que visitante nao possui.
func (f AudienceFilter) memberOnly() bool {
	return len(f.Genders) > 0 || len(f.MaritalStatuses) > 0 ||
		len(f.MembershipStatuses) > 0 || f.AgeMin != nil || f.AgeMax != nil
}

// argBuilder numera os placeholders ($1, $2, ...) conforme os argumentos.
type argBuilder struct{ args []any }

func (b *argBuilder) add(v any) string {
	b.args = append(b.args, v)
	return fmt.Sprintf("$%d", len(b.args))
}

func (b *argBuilder) placeholders(vals []string) string {
	ph := make([]string, len(vals))
	for i, v := range vals {
		ph[i] = b.add(v)
	}
	return "(" + strings.Join(ph, ", ") + ")"
}

func (b *argBuilder) in(column string, vals []string) string {
	if len(vals) == 0 {
		return ""
	}
	return column + " IN " + b.placeholders(vals)
}

// memberConditions devolve as condicoes comuns a qualquer publico de membros.
// `tenantPH` e o placeholder do tenant do contexto: quando a consulta roda com
// role 'system' (scheduler/automacoes), a RLS libera todos os tenants e o filtro
// explicito e o que garante que o disparo nao atravesse tenants.
func memberConditions(f AudienceFilter, tenantPH string, b *argBuilder) []string {
	conds := []string{"m.whatsapp IS NOT NULL", "m.whatsapp <> ''", "m.tenant_id = " + tenantPH}
	if c := b.in("m.branch_id", f.BranchIDs); c != "" {
		conds = append(conds, c)
	}
	if c := b.in("m.gender", f.Genders); c != "" {
		conds = append(conds, c)
	}
	if c := b.in("m.marital_status", f.MaritalStatuses); c != "" {
		conds = append(conds, c)
	}
	if c := b.in("m.membership_status", f.MembershipStatuses); c != "" {
		conds = append(conds, c)
	}
	if f.AgeMin != nil {
		conds = append(conds,
			"m.birth_date IS NOT NULL AND date_part('year', age(m.birth_date)) >= "+b.add(*f.AgeMin))
	}
	if f.AgeMax != nil {
		conds = append(conds,
			"m.birth_date IS NOT NULL AND date_part('year', age(m.birth_date)) <= "+b.add(*f.AgeMax))
	}
	return conds
}

// memberQuery monta o SELECT de membros para o publico pedido.
func memberQuery(audience string, f AudienceFilter, tenantPH string, b *argBuilder) (string, bool) {
	selectList := "m.whatsapp, m.full_name, 'member', m.id::text"
	base := "FROM members m"
	conds := memberConditions(f, tenantPH, b)
	distinct := false

	switch audience {
	case "leaders":
		base = `FROM members m
			LEFT JOIN ministry_members mm ON mm.member_id = m.id AND mm.role = 'leader'
			LEFT JOIN small_groups sg ON sg.leader_id = m.id`
		conds = append(conds, "(mm.member_id IS NOT NULL OR sg.id IS NOT NULL)")
		distinct = true
	case "groups":
		if len(f.GroupIDs) == 0 {
			return "", false
		}
		base = `FROM members m
			JOIN group_attendance ga ON ga.member_id = m.id`
		conds = append(conds, "ga.small_group_id IN "+b.placeholders(f.GroupIDs))
		distinct = true
	case "ministries":
		if len(f.MinistryIDs) == 0 {
			return "", false
		}
		base = `FROM members m
			JOIN ministry_members mm ON mm.member_id = m.id`
		conds = append(conds, "mm.ministry_id IN "+b.placeholders(f.MinistryIDs))
		distinct = true
	}

	sel := "SELECT "
	if distinct {
		sel += "DISTINCT "
	}
	return sel + selectList + " " + base + " WHERE " + strings.Join(conds, " AND "), true
}

// visitorQuery monta o SELECT de visitantes. So o filtro de filial se aplica -
// os demais campos nao existem em `visitors`.
func visitorQuery(f AudienceFilter, tenantPH string, b *argBuilder) string {
	conds := []string{"v.whatsapp IS NOT NULL", "v.whatsapp <> ''", "v.tenant_id = " + tenantPH}
	if c := b.in("v.branch_id", f.BranchIDs); c != "" {
		conds = append(conds, c)
	}
	return "SELECT v.whatsapp, concat_ws(' ', v.first_name, v.last_name), 'visitor', v.id::text " +
		"FROM visitors v WHERE " + strings.Join(conds, " AND ")
}

// buildAudienceQuery devolve o SQL completo de resolucao de destinatarios e os
// argumentos. Retorna SQL vazio quando o publico nao gera consulta (ex.: grupos
// sem nenhum id escolhido).
func buildAudienceQuery(in SendInput, tenantID string) (string, []any) {
	b := &argBuilder{}
	tenantPH := b.add(tenantID)
	audience := in.Audience
	if audience == "" {
		audience = "everyone"
	}
	f := in.AudienceFilter

	parts := []string{}
	switch audience {
	case "visitors":
		parts = append(parts, visitorQuery(f, tenantPH, b))
	case "members", "leaders", "groups", "ministries":
		q, ok := memberQuery(audience, f, tenantPH, b)
		if !ok {
			return "", nil
		}
		parts = append(parts, q)
	default: // everyone
		q, _ := memberQuery("members", f, tenantPH, b)
		parts = append(parts, q)
		// Segmentacao demografica exclui visitantes (nao tem esses atributos).
		if !f.memberOnly() {
			parts = append(parts, visitorQuery(f, tenantPH, b))
		}
	}
	return "SELECT * FROM (" + strings.Join(parts, " UNION ALL ") + ") recipients ORDER BY 2", b.args
}

// ResolveRecipients resolve os destinatarios no escopo RLS da sessao.
func (r *Repo) ResolveRecipients(ctx context.Context, tx pgx.Tx, tenantID string, in SendInput) ([]Recipient, error) {
	sql, args := buildAudienceQuery(in, tenantID)
	if sql == "" {
		return []Recipient{}, nil
	}
	rows, err := tx.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []Recipient{}
	for rows.Next() {
		var phone, fullName, entityType, entityID string
		if err := rows.Scan(&phone, &fullName, &entityType, &entityID); err != nil {
			return nil, err
		}
		if phone != "" {
			out = append(out, Recipient{Phone: phone, FullName: fullName, EntityType: entityType, EntityID: entityID})
		}
	}
	return out, rows.Err()
}

// CountRecipients devolve quantos destinatarios o filtro alcanca, sem carrega-los.
// E o "preview" do disparo em massa.
func (r *Repo) CountRecipients(ctx context.Context, tx pgx.Tx, tenantID string, in SendInput) (int, error) {
	sql, args := buildAudienceQuery(in, tenantID)
	if sql == "" {
		return 0, nil
	}
	var n int
	err := tx.QueryRow(ctx, "SELECT count(*) FROM ("+sql+") q", args...).Scan(&n)
	return n, err
}
