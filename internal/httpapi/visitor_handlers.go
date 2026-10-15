package httpapi

import (
	"crypto/rand"
	"errors"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"chosenerp/internal/announcements"
	"chosenerp/internal/auth"
	"chosenerp/internal/delivery"
	"chosenerp/internal/events"
	"chosenerp/internal/members"
	"chosenerp/internal/prayer"
	"chosenerp/internal/requests"
	"chosenerp/internal/store"
	"chosenerp/internal/visitors"
)

// errNotMembershipRequest indica que o pedido nao e do tipo "quero ser membro".
var errNotMembershipRequest = errors.New("solicitacao nao e de conversao em membro")

// ---- Acesso publico por codigo (WhatsApp) ----

// visitorCodeTTL e a validade do codigo enviado por WhatsApp.
const visitorCodeTTL = 5 * time.Minute

// visitorDigits normaliza um telefone para apenas digitos.
func visitorDigits(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// randomCode gera um codigo numerico de n digitos (cripto-seguro).
func randomCode(n int) string {
	const digits = "0123456789"
	b := make([]byte, n)
	for i := range b {
		v, err := rand.Int(rand.Reader, big.NewInt(int64(len(digits))))
		if err != nil {
			b[i] = '0'
			continue
		}
		b[i] = digits[v.Int64()]
	}
	return string(b)
}

// handleVisitorRequestCode envia um codigo de acesso ao telefone informado,
// pelo WhatsApp da igreja. Resposta sempre generica para nao enumerar numeros.
func (a *App) handleVisitorRequestCode(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	var in struct {
		Name  string `json:"name"`
		Phone string `json:"phone"`
	}
	if err := readJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	phone := visitorDigits(in.Phone)
	if len(phone) < 8 {
		writeErr(w, http.StatusBadRequest, "telefone invalido")
		return
	}

	var tenantID, branchID, tenantName string
	err := a.Store.Pool().QueryRow(r.Context(),
		`SELECT tenant_id, branch_id, tenant_name FROM visitor_tenant_lookup($1)`, slug).
		Scan(&tenantID, &branchID, &tenantName)
	if err != nil {
		writeErr(w, http.StatusNotFound, "not found")
		return
	}

	code := randomCode(6)
	hash, err := auth.HashPassword(code)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "erro ao gerar codigo")
		return
	}
	if _, err := a.Store.Pool().Exec(r.Context(),
		`SELECT visitor_code_issue($1::uuid, $2, $3, $4, $5)`,
		tenantID, phone, in.Name, hash, int(visitorCodeTTL.Seconds())); err != nil {
		// Anti-abuso levantado pela funcao (P0001) -> 429. Qualquer outro erro e
		// falha real (schema/setup) e nao deve ser mascarado de "espere".
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "P0001" {
			writeErr(w, http.StatusTooManyRequests, "aguarde para pedir outro codigo")
			return
		}
		writeServerError(w, err)
		return
	}

	body := fmt.Sprintf("%s: seu codigo de acesso ao app e %s. Vale por 5 minutos.", tenantName, code)
	// Falha de envio nao revela existencia do numero: o codigo fica gravado e o
	// usuario pode tentar de novo. Mas registramos o erro (ex.: filial sem canal
	// de WhatsApp configurado) para diagnostico.
	if err := a.Dispatch.Send(r.Context(), delivery.Message{
		Channel:   delivery.ChannelWhatsApp,
		Recipient: phone,
		Text:      body,
		Subject:   "Codigo de acesso",
		TenantID:  tenantID,
		BranchID:  branchID,
	}); err != nil {
		log.Printf("visitor: falha ao enviar codigo (tenant=%s): %v", tenantID, err)
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// handleVisitorVerifyCode valida o codigo e cria/renova o acesso de visitante,
// emitindo os tokens do app.
func (a *App) handleVisitorVerifyCode(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	var in struct {
		Phone string `json:"phone"`
		Code  string `json:"code"`
	}
	if err := readJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	phone := visitorDigits(in.Phone)
	code := strings.TrimSpace(in.Code)
	if len(phone) < 8 || code == "" {
		writeErr(w, http.StatusBadRequest, "telefone e codigo obrigatorios")
		return
	}

	var tenantID string
	if err := a.Store.Pool().QueryRow(r.Context(),
		`SELECT tenant_id FROM visitor_tenant_lookup($1)`, slug).Scan(&tenantID); err != nil {
		writeErr(w, http.StatusNotFound, "not found")
		return
	}

	var id string
	var name *string
	var hash string
	var expires time.Time
	var attempts, maxAttempts int
	err := a.Store.Pool().QueryRow(r.Context(),
		`SELECT id, name, code_hash, expires_at, attempts, max_attempts FROM visitor_code_fetch($1::uuid, $2)`,
		tenantID, phone).Scan(&id, &name, &hash, &expires, &attempts, &maxAttempts)
	if err != nil {
		writeErr(w, http.StatusUnauthorized, "codigo invalido ou expirado")
		return
	}
	if attempts >= maxAttempts || time.Now().After(expires) {
		writeErr(w, http.StatusUnauthorized, "codigo invalido ou expirado")
		return
	}
	if !auth.CheckPassword(hash, code) {
		_, _ = a.Store.Pool().Exec(r.Context(), `SELECT visitor_code_use($1::uuid, false)`, id)
		writeErr(w, http.StatusUnauthorized, "codigo invalido ou expirado")
		return
	}
	_, _ = a.Store.Pool().Exec(r.Context(), `SELECT visitor_code_use($1::uuid, true)`, id)

	nameVal := ""
	if name != nil {
		nameVal = *name
	}
	var userID, visitorID, branchID string
	err = a.Store.Pool().QueryRow(r.Context(),
		`SELECT user_id, visitor_id, branch_id FROM user_attach_visitor($1, $2, $3::uuid)`,
		phone, nameVal, tenantID).Scan(&userID, &visitorID, &branchID)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			// O telefone ja tem acesso de membro/staff: use o login normal.
			writeErr(w, http.StatusForbidden, "use_password")
			return
		}
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}

	res, err := a.Auth.IssueVisitor(r.Context(), userID, tenantID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "erro ao emitir acesso")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"tokens": res.Tokens,
		"user":   profilePayload(res.Profile),
	})
}

// ---- App do visitante (escopo self) ----

func (a *App) handleMeVisitor(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	var v *visitors.Visitor
	err := a.Store.WithTenant(r.Context(), boundsFromClaims(claims), func(tx pgx.Tx) error {
		var e error
		v, e = a.visitorForClaims(r.Context(), tx, claims)
		return e
	})
	if err != nil {
		writeVisitorErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"visitor": v})
}

func (a *App) handleUpdateMeVisitor(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	var in visitors.ContactInput
	if err := readJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	var v *visitors.Visitor
	err := a.Store.WithTenant(r.Context(), boundsFromClaims(claims), func(tx pgx.Tx) error {
		cur, e := a.visitorForClaims(r.Context(), tx, claims)
		if e != nil {
			return e
		}
		v, e = a.Visitors.UpdateContact(r.Context(), tx, cur.ID, in)
		return e
	})
	if err != nil {
		writeVisitorErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"visitor": v})
}

func (a *App) handleVisitorAnnouncements(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	var out []announcements.Announcement
	err := a.Store.WithTenant(r.Context(), boundsFromClaims(claims), func(tx pgx.Tx) error {
		if _, e := a.visitorForClaims(r.Context(), tx, claims); e != nil {
			return e
		}
		var e error
		out, e = a.Announcements.List(r.Context(), tx)
		return e
	})
	if err != nil {
		writeVisitorErr(w, err)
		return
	}
	if out == nil {
		out = []announcements.Announcement{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"announcements": out})
}

func (a *App) handleVisitorEvents(w http.ResponseWriter, r *http.Request) {
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
		if _, e := a.visitorForClaims(r.Context(), tx, claims); e != nil {
			return e
		}
		var e error
		out, e = a.Events.ListEvents(r.Context(), tx, from, to, kind)
		return e
	})
	if err != nil {
		writeVisitorErr(w, err)
		return
	}
	if out == nil {
		out = []events.Event{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"events": out})
}

func (a *App) handleVisitorPrayerWall(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	var out []prayer.Request
	err := a.Store.WithTenant(r.Context(), boundsFromClaims(claims), func(tx pgx.Tx) error {
		if _, e := a.visitorForClaims(r.Context(), tx, claims); e != nil {
			return e
		}
		var e error
		out, e = a.Prayers.Wall(r.Context(), tx, claims.UserID)
		return e
	})
	if err != nil {
		writeVisitorErr(w, err)
		return
	}
	if out == nil {
		out = []prayer.Request{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"prayer_requests": out})
}

func (a *App) handleVisitorListMyPrayers(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	var out []prayer.Request
	err := a.Store.WithTenant(r.Context(), boundsFromClaims(claims), func(tx pgx.Tx) error {
		if _, e := a.visitorForClaims(r.Context(), tx, claims); e != nil {
			return e
		}
		var e error
		out, e = a.Prayers.ListByUser(r.Context(), tx, claims.UserID)
		return e
	})
	if err != nil {
		writeVisitorErr(w, err)
		return
	}
	if out == nil {
		out = []prayer.Request{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"prayer_requests": out})
}

func (a *App) handleVisitorCreatePrayer(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	var in prayer.CreateInput
	if err := readJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	// Visitante so publica no mural da igreja ou direciona ao pastorado.
	if in.Visibility == nil || (*in.Visibility != "igreja" && *in.Visibility != "pastor") {
		v := "igreja"
		in.Visibility = &v
	}
	var req *prayer.Request
	err := a.Store.WithTenant(r.Context(), boundsFromClaims(claims), func(tx pgx.Tx) error {
		v, e := a.visitorForClaims(r.Context(), tx, claims)
		if e != nil {
			return e
		}
		req, e = a.Prayers.Create(r.Context(), tx, claims.TenantID, v.BranchID, "", claims.UserID, v.FullName, in)
		return e
	})
	if err != nil {
		if errors.Is(err, prayer.ErrInvalid) {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		writeVisitorErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, req)
}

func (a *App) handleVisitorReactPrayer(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	id := r.PathValue("id")
	var req *prayer.Request
	err := a.Store.WithTenant(r.Context(), boundsFromClaims(claims), func(tx pgx.Tx) error {
		v, e := a.visitorForClaims(r.Context(), tx, claims)
		if e != nil {
			return e
		}
		req, e = a.Prayers.React(r.Context(), tx, claims.TenantID, v.BranchID, id, claims.UserID, "")
		return e
	})
	if err != nil {
		writeVisitorErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, req)
}

func (a *App) handleVisitorListMyRequests(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	var out []requests.Request
	err := a.Store.WithTenant(r.Context(), boundsFromClaims(claims), func(tx pgx.Tx) error {
		if _, e := a.visitorForClaims(r.Context(), tx, claims); e != nil {
			return e
		}
		var e error
		out, e = a.Requests.ListByUser(r.Context(), tx, claims.UserID)
		return e
	})
	if err != nil {
		writeVisitorErr(w, err)
		return
	}
	if out == nil {
		out = []requests.Request{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"requests": out})
}

func (a *App) handleVisitorCreateMyRequest(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	var in requests.CreateInput
	if err := readJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	var out *requests.Request
	err := a.Store.WithTenant(r.Context(), boundsFromClaims(claims), func(tx pgx.Tx) error {
		v, e := a.visitorForClaims(r.Context(), tx, claims)
		if e != nil {
			return e
		}
		out, e = a.Requests.Create(r.Context(), tx, claims.TenantID, v.BranchID, "", claims.UserID, in)
		return e
	})
	if err != nil {
		if errors.Is(err, requests.ErrInvalidKind) {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		writeVisitorErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"request": out})
}

// ---- Staff: trilha e aprovacao da conversao ----

// handleVisitorJourney devolve a trilha de etapas do visitante (secretaria).
func (a *App) handleVisitorJourney(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	id := r.PathValue("id")
	var steps []visitors.JourneyStep
	err := a.Store.WithTenant(r.Context(), boundsFromClaims(claims), func(tx pgx.Tx) error {
		if _, e := a.Visitors.Get(r.Context(), tx, id); e != nil {
			return e
		}
		var e error
		steps, e = a.Visitors.ListJourney(r.Context(), tx, id)
		return e
	})
	if err != nil {
		if store.IsNotFound(err) {
			writeErr(w, http.StatusNotFound, "visitor not found")
			return
		}
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if steps == nil {
		steps = []visitors.JourneyStep{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"journey": steps})
}

// handleApproveMembership aprova um pedido "quero ser membro": converte o
// visitante em membro e promove o acesso existente (preserva a identidade).
func (a *App) handleApproveMembership(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	id := r.PathValue("id")
	var out *requests.Request
	err := a.Store.WithTenant(r.Context(), boundsFromClaims(claims), func(tx pgx.Tx) error {
		req, e := a.Requests.Get(r.Context(), tx, id)
		if e != nil {
			return e
		}
		if req.Kind != "quero_ser_membro" {
			return errNotMembershipRequest
		}

		// Se veio de um visitante com acesso, converte e promove.
		if req.UserID != nil {
			var visitorID string
			_ = tx.QueryRow(r.Context(), `
				SELECT COALESCE(visitor_id::text,'')
				FROM memberships WHERE user_id = $1::uuid AND tenant_id = $2::uuid`,
				*req.UserID, claims.TenantID).Scan(&visitorID)
			if visitorID != "" {
				v, e := a.Visitors.Get(r.Context(), tx, visitorID)
				if e != nil {
					return e
				}
				in := members.CreateInput{
					FirstName: v.FirstName, LastName: v.LastName,
					Email: v.Email, Phone: v.Phone, Whatsapp: v.Whatsapp,
				}
				m, e := a.Visitors.Convert(r.Context(), tx, visitorID, claims.TenantID, v.BranchID, in, claims.UserID)
				if e != nil {
					return e
				}
				if e := a.Visitors.PromoteToMember(r.Context(), tx, *req.UserID, claims.TenantID, m.ID); e != nil {
					return e
				}
			}
		}

		out, e = a.Requests.Update(r.Context(), tx, id, requests.UpdateInput{Status: "concluida"}, claims.UserID)
		return e
	})
	if err != nil {
		if errors.Is(err, errNotMembershipRequest) {
			writeErr(w, http.StatusBadRequest, "solicitacao nao e de conversao em membro")
			return
		}
		if store.IsNotFound(err) {
			writeErr(w, http.StatusNotFound, "solicitacao nao encontrada")
			return
		}
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, out)
}
