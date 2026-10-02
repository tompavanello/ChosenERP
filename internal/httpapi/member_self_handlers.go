package httpapi

import (
	"net/http"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"

	"chosenerp/internal/announcements"
	"chosenerp/internal/events"
	"chosenerp/internal/families"
	"chosenerp/internal/finance"
	"chosenerp/internal/groups"
	"chosenerp/internal/members"
	"chosenerp/internal/ministries"
)

// handleMeMember devolve o cadastro de membro vinculado a identidade.
func (a *App) handleMeMember(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	var m *members.Member
	err := a.Store.WithTenant(r.Context(), boundsFromClaims(claims), func(tx pgx.Tx) error {
		var e error
		m, e = a.memberForClaims(r.Context(), tx, claims)
		return e
	})
	if err != nil {
		writeMemberErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"member": m})
}

// handleUpdateMeMember permite ao membro editar os proprios dados de contato e
// endereco. Nome, CPF/RG e a vida eclesiastica NAO sao editaveis pelo membro -
// so pela secretaria (ver members.UpdateInput).
func (a *App) handleUpdateMeMember(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	var in struct {
		Email    *string          `json:"email"`
		Phone    *string          `json:"phone"`
		Whatsapp *string          `json:"whatsapp"`
		Nickname *string          `json:"nickname"`
		Address  *members.Address `json:"address"`
	}
	if err := readJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	upd := members.UpdateInput{
		Email: in.Email, Phone: in.Phone, Whatsapp: in.Whatsapp,
		Nickname: in.Nickname, Address: in.Address,
	}
	var m *members.Member
	err := a.Store.WithTenant(r.Context(), boundsFromClaims(claims), func(tx pgx.Tx) error {
		mem, e := a.memberForClaims(r.Context(), tx, claims)
		if e != nil {
			return e
		}
		m, e = a.Members.Update(r.Context(), tx, mem.ID, upd, claims.UserID)
		return e
	})
	if err != nil {
		writeMemberErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"member": m})
}

// handleMeFamily lista as familias das quais o membro participa.
func (a *App) handleMeFamily(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	var out []families.Family
	err := a.Store.WithTenant(r.Context(), boundsFromClaims(claims), func(tx pgx.Tx) error {
		m, e := a.memberForClaims(r.Context(), tx, claims)
		if e != nil {
			return e
		}
		out, e = a.Families.ListByMember(r.Context(), tx, m.ID)
		return e
	})
	if err != nil {
		writeMemberErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"families": out})
}

// handleMeEvents lista a agenda da igreja no escopo do membro.
func (a *App) handleMeEvents(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")
	kind := r.URL.Query().Get("kind")
	var out []events.Event
	err := a.Store.WithTenant(r.Context(), boundsFromClaims(claims), func(tx pgx.Tx) error {
		if _, e := a.memberForClaims(r.Context(), tx, claims); e != nil {
			return e
		}
		var e error
		out, e = a.Events.ListEvents(r.Context(), tx, from, to, kind)
		return e
	})
	if err != nil {
		writeMemberErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"events": out})
}

// handleMeAnnouncements lista os avisos ativos da igreja/filial do membro.
func (a *App) handleMeAnnouncements(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	var out []announcements.Announcement
	err := a.Store.WithTenant(r.Context(), boundsFromClaims(claims), func(tx pgx.Tx) error {
		if _, e := a.memberForClaims(r.Context(), tx, claims); e != nil {
			return e
		}
		var e error
		out, e = a.Announcements.List(r.Context(), tx)
		return e
	})
	if err != nil {
		writeMemberErr(w, err)
		return
	}
	if out == nil {
		out = []announcements.Announcement{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"announcements": out})
}

// handleMeBirthdays lista os aniversariantes (nascimento e casamento) do mes.
// O mes vem de ?month= (1-12); o padrao e o mes corrente.
func (a *App) handleMeBirthdays(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	month := monthParam(r)
	var birthdays []members.Birthday
	var marriages []members.MarriageAnniversary
	err := a.Store.WithTenant(r.Context(), boundsFromClaims(claims), func(tx pgx.Tx) error {
		if _, e := a.memberForClaims(r.Context(), tx, claims); e != nil {
			return e
		}
		var e error
		if birthdays, e = a.Members.Birthdays(r.Context(), tx, month); e != nil {
			return e
		}
		marriages, e = a.Members.Marriages(r.Context(), tx, month)
		return e
	})
	if err != nil {
		writeMemberErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"month": month, "birthdays": birthdays, "marriages": marriages,
	})
}

// handleMeMinistries lista os ministerios dos quais o membro participa.
func (a *App) handleMeMinistries(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	var out []ministries.MemberMinistry
	err := a.Store.WithTenant(r.Context(), boundsFromClaims(claims), func(tx pgx.Tx) error {
		m, e := a.memberForClaims(r.Context(), tx, claims)
		if e != nil {
			return e
		}
		out, e = a.Ministries.ListByMember(r.Context(), tx, m.ID)
		return e
	})
	if err != nil {
		writeMemberErr(w, err)
		return
	}
	if out == nil {
		out = []ministries.MemberMinistry{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"ministries": out})
}

// handleMeContributions lista as contribuicoes do proprio membro em um ano.
// O ano vem de ?year=; o padrao e o ano corrente.
func (a *App) handleMeContributions(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	year := time.Now().Year()
	if y := r.URL.Query().Get("year"); y != "" {
		if parsed, err := strconv.Atoi(y); err == nil {
			year = parsed
		}
	}
	var out []finance.Contribution
	err := a.Store.WithTenant(r.Context(), boundsFromClaims(claims), func(tx pgx.Tx) error {
		m, e := a.memberForClaims(r.Context(), tx, claims)
		if e != nil {
			return e
		}
		out, e = a.Finance.ListContributions(r.Context(), tx, m.ID, year)
		return e
	})
	if err != nil {
		writeMemberErr(w, err)
		return
	}
	if out == nil {
		out = []finance.Contribution{}
	}
	var total float64
	for _, c := range out {
		total += c.Amount
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"year": year, "contributions": out, "total": total,
	})
}

// handleMeGroups lista os grupos/celulas dos quais o membro participa.
func (a *App) handleMeGroups(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	var out []groups.MyGroup
	err := a.Store.WithTenant(r.Context(), boundsFromClaims(claims), func(tx pgx.Tx) error {
		m, e := a.memberForClaims(r.Context(), tx, claims)
		if e != nil {
			return e
		}
		out, e = a.Groups.ListForMember(r.Context(), tx, m.ID)
		return e
	})
	if err != nil {
		writeMemberErr(w, err)
		return
	}
	if out == nil {
		out = []groups.MyGroup{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"groups": out})
}
