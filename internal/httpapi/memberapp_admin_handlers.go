package httpapi

import (
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
)

// memberAppUser e o acesso do membro ao app (visao de gestao da Sede/secretaria).
type memberAppUser struct {
	MemberID           string     `json:"member_id"`
	MemberName         string     `json:"member_name"`
	BranchID           string     `json:"branch_id"`
	UserID             string     `json:"user_id"`
	Email              *string    `json:"email,omitempty"`
	Phone              *string    `json:"phone,omitempty"`
	IsActive           bool       `json:"is_active"`
	MustChangePassword bool       `json:"must_change_password"`
	LastLoginAt        *time.Time `json:"last_login_at,omitempty"`
	Devices            int        `json:"devices"`
	LastDeviceAt       *time.Time `json:"last_device_at,omitempty"`
}

// handleMemberAppOverview lista os acessos ao app do membro com dados de uso
// (ultimo login, dispositivos/Web Push). Roda no escopo RLS do usuario para a
// listagem e em contexto de sistema apenas para contar as inscricoes de push
// (a policy de push_subscriptions e por dono).
func (a *App) handleMemberAppOverview(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	var out []memberAppUser
	var ids []string
	err := a.Store.WithTenant(r.Context(), boundsFromClaims(claims), func(tx pgx.Tx) error {
		rows, err := tx.Query(r.Context(), `
			SELECT m.id::text, m.full_name, m.branch_id::text, u.id::text, u.email, u.phone,
			       u.is_active, u.must_change_password, u.last_login_at
			FROM memberships me
			JOIN members m ON m.id = me.member_id
			JOIN users u ON u.id = me.user_id
			WHERE me.member_id IS NOT NULL
			ORDER BY m.full_name`)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var x memberAppUser
			if err := rows.Scan(&x.MemberID, &x.MemberName, &x.BranchID, &x.UserID, &x.Email, &x.Phone,
				&x.IsActive, &x.MustChangePassword, &x.LastLoginAt); err != nil {
				return err
			}
			out = append(out, x)
			ids = append(ids, x.UserID)
		}
		return rows.Err()
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if out == nil {
		out = []memberAppUser{}
	}

	if len(ids) > 0 {
		_ = a.Store.WithSystem(r.Context(), func(tx pgx.Tx) error {
			rows, err := tx.Query(r.Context(), `
				SELECT user_id::text, count(*)::int, max(last_seen_at)
				FROM push_subscriptions WHERE user_id = ANY($1::uuid[])
				GROUP BY user_id`, ids)
			if err != nil {
				return err
			}
			defer rows.Close()
			idx := map[string]int{}
			for i := range out {
				idx[out[i].UserID] = i
			}
			for rows.Next() {
				var uid string
				var cnt int
				var last *time.Time
				if err := rows.Scan(&uid, &cnt, &last); err != nil {
					return err
				}
				if i, ok := idx[uid]; ok {
					out[i].Devices = cnt
					out[i].LastDeviceAt = last
				}
			}
			return rows.Err()
		})
	}

	active, never := 0, 0
	for _, u := range out {
		if u.IsActive {
			active++
		}
		if u.LastLoginAt == nil {
			never++
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"users": out,
		"totals": map[string]any{
			"total":       len(out),
			"active":      active,
			"never_login": never,
		},
	})
}
