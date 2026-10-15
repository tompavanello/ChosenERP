package payments

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"chosenerp/internal/store"
)

// ErrDisabled sinaliza que o gateway nao esta configurado (sem API key), de modo
// que as operacoes de cobranca ficam indisponiveis.
var ErrDisabled = errors.New("pagamentos nao configurados")

// ErrNoSubscription sinaliza que a igreja ainda nao tem assinatura.
var ErrNoSubscription = errors.New("igreja sem assinatura")

// Config reune os parametros da plataforma (Fluxo A) e da cifragem.
type Config struct {
	DefaultProvider   string
	AsaasBaseURL      string
	AsaasAPIKey       string
	AsaasWebhookToken string
	CredentialsKey    string
	JWTSecret         string
	SuspendEnabled    bool
	SuspendAfterDays  int
}

// Credentials sao as credenciais decifradas de um gateway.
type Credentials struct {
	APIKey       string `json:"api_key"`
	BaseURL      string `json:"base_url"`
	WebhookToken string `json:"webhook_token"`
}

// Factory constroi um Gateway para o provider (injetado em main.go para evitar
// ciclo de import entre payments e payments/asaas).
type Factory func(provider string, creds Credentials) (Gateway, error)

// Service orquestra os dois fluxos de cobranca.
type Service struct {
	Store   *store.Store
	Cfg     Config
	Factory Factory
}

// NewService cria o servico de pagamentos.
func NewService(st *store.Store, cfg Config) *Service {
	return &Service{Store: st, Cfg: cfg}
}

// credKey devolve a chave AES derivada da config.
func (s *Service) credKey() []byte {
	return authKey(s.Cfg.CredentialsKey, s.Cfg.JWTSecret)
}

// VerifyWebhookToken valida o token do webhook. Falha FECHADO: sem nenhum token
// esperado configurado (plataforma ou igreja) a requisicao e rejeitada. Aceita se
// o token bater com o segredo da plataforma OU com o token da igreja dona do
// evento (Fluxo B), sempre em comparacao de tempo constante.
func (s *Service) VerifyWebhookToken(ctx context.Context, token, provider, tenantID string) bool {
	if token == "" {
		return false
	}
	matched := false
	if s.Cfg.AsaasWebhookToken != "" {
		matched = subtle.ConstantTimeCompare([]byte(token), []byte(s.Cfg.AsaasWebhookToken)) == 1
	}
	if !matched && tenantID != "" {
		if t := s.tenantWebhookToken(ctx, tenantID, provider); t != "" {
			matched = subtle.ConstantTimeCompare([]byte(token), []byte(t)) == 1
		}
	}
	return matched
}

// tenantWebhookToken le (e decifra) o token de webhook configurado pela igreja.
func (s *Service) tenantWebhookToken(ctx context.Context, tenantID, provider string) string {
	if s.Store == nil {
		return ""
	}
	var raw []byte
	err := s.Store.WithSystem(ctx, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `
			SELECT credentials FROM payment_gateway_configs
			WHERE tenant_id=$1::uuid AND is_active=true AND ($2='' OR provider=$2)
			ORDER BY is_default DESC, created_at ASC LIMIT 1`, tenantID, provider).Scan(&raw)
	})
	if err != nil {
		return ""
	}
	dec, e := DecryptCredentials(s.credKey(), raw)
	if e != nil {
		return ""
	}
	var creds Credentials
	if json.Unmarshal(dec, &creds) != nil {
		return ""
	}
	return creds.WebhookToken
}

// TenantFromReference extrai o tenant de uma externalReference "tenant:<uuid>".
func TenantFromReference(ref string) string {
	const prefix = "tenant:"
	if !strings.HasPrefix(ref, prefix) {
		return ""
	}
	v := strings.TrimPrefix(ref, prefix)
	if i := strings.IndexByte(v, ':'); i >= 0 {
		v = v[:i]
	}
	v = strings.TrimSpace(v)
	if len(v) != 36 {
		return ""
	}
	return v
}

// platformCredentials devolve as credenciais da conta da plataforma (Fluxo A).
func (s *Service) platformCredentials() Credentials {
	return Credentials{APIKey: s.Cfg.AsaasAPIKey, BaseURL: s.Cfg.AsaasBaseURL}
}

// PlatformGateway constroi o gateway da plataforma.
func (s *Service) PlatformGateway() (Gateway, error) {
	if s.Factory == nil {
		return nil, ErrDisabled
	}
	provider := s.Cfg.DefaultProvider
	if provider == "" {
		provider = "asaas"
	}
	return s.Factory(provider, s.platformCredentials())
}

// Enabled indica se a plataforma tem gateway configurado (API key presente).
func (s *Service) Enabled() bool {
	if strings.EqualFold(s.Cfg.DefaultProvider, "mercadopago") ||
		strings.EqualFold(s.Cfg.DefaultProvider, "stone") ||
		strings.EqualFold(s.Cfg.DefaultProvider, "picpay") {
		return false
	}
	return s.Cfg.AsaasAPIKey != ""
}

// ---- View models ----

// BillingCustomer espelha a linha de billing_customers.
type BillingCustomer struct {
	ID         string `json:"id"`
	Provider   string `json:"provider"`
	ExternalID string `json:"external_id"`
	Name       string `json:"name"`
	Email      string `json:"email"`
	Phone      string `json:"phone"`
	CPFCNPJ    string `json:"cpf_cnpj"`
}

// BillingSubscription espelha a linha de billing_subscriptions.
type BillingSubscription struct {
	ID          string     `json:"id"`
	PlanKey     string     `json:"plan_key"`
	Provider    string     `json:"provider"`
	ExternalID  string     `json:"external_id"`
	BillingType string     `json:"billing_type"`
	Cycle       string     `json:"cycle"`
	ValueCents  int64      `json:"value_cents"`
	NextDueDate *string    `json:"next_due_date"`
	Status      string     `json:"status"`
	StartedAt   time.Time  `json:"started_at"`
	CanceledAt  *time.Time `json:"canceled_at"`
}

// BillingInvoice espelha a linha de billing_invoices.
type BillingInvoice struct {
	ID           string     `json:"id"`
	Provider     string     `json:"provider"`
	ExternalID   string     `json:"external_id"`
	BillingType  string     `json:"billing_type"`
	Status       string     `json:"status"`
	ValueCents   int64      `json:"value_cents"`
	NetValueCents *int64    `json:"net_value_cents"`
	DueDate      *string    `json:"due_date"`
	PaidAt       *time.Time `json:"paid_at"`
	InvoiceURL   string     `json:"invoice_url"`
	BankSlipURL  string     `json:"bank_slip_url"`
	PixPayload   string     `json:"pix_payload"`
	PixQRImage   string     `json:"pix_qr_image"`
	Description  string     `json:"description"`
	CreatedAt    time.Time  `json:"created_at"`
}

// TenantBilling agrega a cobranca de uma igreja (Fluxo A).
type TenantBilling struct {
	TenantID      string               `json:"tenant_id"`
	TenantName    string               `json:"tenant_name"`
	Plan          string               `json:"plan"`
	BillingStatus string               `json:"billing_status"`
	BillingExempt bool                 `json:"billing_exempt"`
	PastDueSince  *time.Time           `json:"past_due_since"`
	SuspendedAt   *time.Time           `json:"suspended_at"`
	Customer      *BillingCustomer     `json:"customer"`
	Subscription  *BillingSubscription `json:"subscription"`
	Invoices      []BillingInvoice     `json:"invoices"`
}

// SubscribeInput e a entrada para criar/atualizar a assinatura de uma igreja.
type SubscribeInput struct {
	PlanKey     string `json:"plan_key"`
	Provider    string `json:"provider"`
	BillingType string `json:"billing_type"`
	Cycle       string `json:"cycle"`
	CPFCNPJ     string `json:"cpf_cnpj"`
	Email       string `json:"email"`
	Phone       string `json:"phone"`
	NextDueDate string `json:"next_due_date"`
	ValueCents  *int64 `json:"value_cents"`
	Card        *Card  `json:"card"`
}

// ChargeInputView e a entrada para uma cobranca avulsa (Fluxo A).
type ChargeInputView struct {
	ValueCents  int64  `json:"value_cents"`
	BillingType string `json:"billing_type"`
	DueDate     string `json:"due_date"`
	Description string `json:"description"`
	CPFCNPJ     string `json:"cpf_cnpj"`
	Email       string `json:"email"`
	Phone       string `json:"phone"`
	Card        *Card  `json:"card"`
}

// ---- Fluxo A: leitura ----

func (s *Service) tenantBilling(ctx context.Context, tx pgx.Tx, tenantID string) (*TenantBilling, error) {
	tb := &TenantBilling{TenantID: tenantID, Invoices: []BillingInvoice{}}
	err := tx.QueryRow(ctx, `
		SELECT name, plan, billing_status, billing_exempt, past_due_since, suspended_at
		FROM tenants WHERE id = $1::uuid`, tenantID).
		Scan(&tb.TenantName, &tb.Plan, &tb.BillingStatus, &tb.BillingExempt, &tb.PastDueSince, &tb.SuspendedAt)
	if err != nil {
		return nil, err
	}

	// Cliente.
	var cust BillingCustomer
	err = tx.QueryRow(ctx, `
		SELECT id, provider, external_id, COALESCE(name,''), COALESCE(email,''), COALESCE(phone,''), COALESCE(cpf_cnpj,'')
		FROM billing_customers WHERE tenant_id = $1::uuid LIMIT 1`, tenantID).
		Scan(&cust.ID, &cust.Provider, &cust.ExternalID, &cust.Name, &cust.Email, &cust.Phone, &cust.CPFCNPJ)
	if err == nil {
		tb.Customer = &cust
	} else if !store.IsNotFound(err) {
		return nil, err
	}

	// Assinatura viva.
	sub, err := scanActiveSubscription(ctx, tx, tenantID)
	if err == nil {
		tb.Subscription = sub
	} else if !store.IsNotFound(err) {
		return nil, err
	}

	// Faturas.
	rows, err := tx.Query(ctx, `
		SELECT id, provider, COALESCE(external_id,''), COALESCE(billing_type,''), status,
		       value_cents, net_value_cents, to_char(due_date,'YYYY-MM-DD'), paid_at,
		       COALESCE(invoice_url,''), COALESCE(bank_slip_url,''), COALESCE(pix_payload,''),
		       COALESCE(pix_qr_image,''), COALESCE(description,''), created_at
		FROM billing_invoices WHERE tenant_id = $1::uuid
		ORDER BY created_at DESC LIMIT 24`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var inv BillingInvoice
		if err := rows.Scan(&inv.ID, &inv.Provider, &inv.ExternalID, &inv.BillingType, &inv.Status,
			&inv.ValueCents, &inv.NetValueCents, &inv.DueDate, &inv.PaidAt, &inv.InvoiceURL,
			&inv.BankSlipURL, &inv.PixPayload, &inv.PixQRImage, &inv.Description, &inv.CreatedAt); err != nil {
			return nil, err
		}
		tb.Invoices = append(tb.Invoices, inv)
	}
	return tb, rows.Err()
}

func scanActiveSubscription(ctx context.Context, tx pgx.Tx, tenantID string) (*BillingSubscription, error) {
	var sub BillingSubscription
	err := tx.QueryRow(ctx, `
		SELECT id, COALESCE(plan_key,''), provider, COALESCE(external_id,''), billing_type, cycle,
		       value_cents, to_char(next_due_date,'YYYY-MM-DD'), status, started_at, canceled_at
		FROM billing_subscriptions
		WHERE tenant_id = $1::uuid AND status IN ('active','inactive')
		ORDER BY started_at DESC LIMIT 1`, tenantID).
		Scan(&sub.ID, &sub.PlanKey, &sub.Provider, &sub.ExternalID, &sub.BillingType, &sub.Cycle,
			&sub.ValueCents, &sub.NextDueDate, &sub.Status, &sub.StartedAt, &sub.CanceledAt)
	if err != nil {
		return nil, err
	}
	return &sub, nil
}

// GetTenantBilling devolve a cobranca de uma igreja (platform/system).
func (s *Service) GetTenantBilling(ctx context.Context, tenantID string) (*TenantBilling, error) {
	var out *TenantBilling
	err := s.Store.WithSystem(ctx, func(tx pgx.Tx) error {
		var e error
		out, e = s.tenantBilling(ctx, tx, tenantID)
		return e
	})
	return out, err
}

// GetOwnBilling devolve a cobranca da propria igreja (area da igreja).
func (s *Service) GetOwnBilling(ctx context.Context, tenantID string) (*TenantBilling, error) {
	var out *TenantBilling
	err := s.Store.WithTenant(ctx, store.Bounds{TenantID: tenantID, Role: "super_admin"}, func(tx pgx.Tx) error {
		var e error
		out, e = s.tenantBilling(ctx, tx, tenantID)
		return e
	})
	return out, err
}

// SubscriptionRow e uma assinatura listada no console da plataforma.
type SubscriptionRow struct {
	TenantID      string  `json:"tenant_id"`
	TenantName    string  `json:"tenant_name"`
	Plan          string  `json:"plan"`
	BillingStatus string  `json:"billing_status"`
	ValueCents    int64   `json:"value_cents"`
	Cycle         string  `json:"cycle"`
	BillingType   string  `json:"billing_type"`
	Status        string  `json:"status"`
	NextDueDate   *string `json:"next_due_date"`
}

// ListSubscriptions lista as assinaturas das igrejas.
func (s *Service) ListSubscriptions(ctx context.Context) ([]SubscriptionRow, error) {
	out := []SubscriptionRow{}
	err := s.Store.WithSystem(ctx, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `
			SELECT t.id, t.name, t.plan, t.billing_status,
			       COALESCE(bs.value_cents,0), COALESCE(bs.cycle,''), COALESCE(bs.billing_type,''),
			       COALESCE(bs.status,'none'), to_char(bs.next_due_date,'YYYY-MM-DD')
			FROM tenants t
			LEFT JOIN billing_subscriptions bs
			  ON bs.tenant_id = t.id AND bs.status IN ('active','inactive')
			ORDER BY t.name`)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var r SubscriptionRow
			if err := rows.Scan(&r.TenantID, &r.TenantName, &r.Plan, &r.BillingStatus,
				&r.ValueCents, &r.Cycle, &r.BillingType, &r.Status, &r.NextDueDate); err != nil {
				return err
			}
			out = append(out, r)
		}
		return rows.Err()
	})
	return out, err
}

// InvoiceRow e uma fatura listada no console (com o nome da igreja).
type InvoiceRow struct {
	BillingInvoice
	TenantID   string `json:"tenant_id"`
	TenantName string `json:"tenant_name"`
}

// ListInvoices lista faturas, opcionalmente filtrando por status.
func (s *Service) ListInvoices(ctx context.Context, status string, limit int) ([]InvoiceRow, error) {
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	out := []InvoiceRow{}
	err := s.Store.WithSystem(ctx, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `
			SELECT bi.id, bi.provider, COALESCE(bi.external_id,''), COALESCE(bi.billing_type,''), bi.status,
			       bi.value_cents, bi.net_value_cents, to_char(bi.due_date,'YYYY-MM-DD'), bi.paid_at,
			       COALESCE(bi.invoice_url,''), COALESCE(bi.bank_slip_url,''), COALESCE(bi.pix_payload,''),
			       COALESCE(bi.pix_qr_image,''), COALESCE(bi.description,''), bi.created_at,
			       t.id, t.name
			FROM billing_invoices bi
			JOIN tenants t ON t.id = bi.tenant_id
			WHERE ($1 = '' OR bi.status = $1)
			ORDER BY bi.created_at DESC LIMIT $2`, status, limit)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var r InvoiceRow
			if err := rows.Scan(&r.ID, &r.Provider, &r.ExternalID, &r.BillingType, &r.Status,
				&r.ValueCents, &r.NetValueCents, &r.DueDate, &r.PaidAt, &r.InvoiceURL,
				&r.BankSlipURL, &r.PixPayload, &r.PixQRImage, &r.Description, &r.CreatedAt,
				&r.TenantID, &r.TenantName); err != nil {
				return err
			}
			out = append(out, r)
		}
		return rows.Err()
	})
	return out, err
}

// Overview reune indicadores de cobranca da plataforma.
type Overview struct {
	ActiveSubscriptions int   `json:"active_subscriptions"`
	OverdueInvoices     int   `json:"overdue_invoices"`
	SuspendedTenants    int   `json:"suspended_tenants"`
	MRRCents            int64 `json:"mrr_cents"`
	OverdueCents        int64 `json:"overdue_cents"`
}

// GetOverview calcula os indicadores do console.
func (s *Service) GetOverview(ctx context.Context) (*Overview, error) {
	o := &Overview{}
	err := s.Store.WithSystem(ctx, func(tx pgx.Tx) error {
		if err := tx.QueryRow(ctx, `
			SELECT COUNT(*), COALESCE(SUM(CASE WHEN cycle='YEARLY' THEN value_cents/12 ELSE value_cents END),0)
			FROM billing_subscriptions WHERE status = 'active'`).
			Scan(&o.ActiveSubscriptions, &o.MRRCents); err != nil {
			return err
		}
		if err := tx.QueryRow(ctx, `
			SELECT COUNT(*), COALESCE(SUM(value_cents),0)
			FROM billing_invoices WHERE status = 'overdue'`).
			Scan(&o.OverdueInvoices, &o.OverdueCents); err != nil {
			return err
		}
		return tx.QueryRow(ctx,
			`SELECT COUNT(*) FROM tenants WHERE billing_status = 'suspended'`).Scan(&o.SuspendedTenants)
	})
	return o, err
}

// ---- Fluxo A: escrita (plataforma) ----

// ensureCustomer garante o cliente no gateway e em billing_customers.
func (s *Service) ensureCustomer(ctx context.Context, tx pgx.Tx, tenantID, provider string, gw Gateway, in SubscribeInput, tenantName string) (string, error) {
	var externalID string
	err := tx.QueryRow(ctx,
		`SELECT external_id FROM billing_customers WHERE tenant_id=$1::uuid AND provider=$2`,
		tenantID, provider).Scan(&externalID)
	if err == nil {
		return externalID, nil
	}
	if !store.IsNotFound(err) {
		return "", err
	}
	cust, err := gw.CreateCustomer(ctx, CustomerInput{
		ExternalReference: "tenant:" + tenantID,
		Name:              tenantName,
		Email:             in.Email,
		Phone:             in.Phone,
		CPFCNPJ:           in.CPFCNPJ,
	})
	if err != nil {
		return "", err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO billing_customers (tenant_id, provider, external_id, name, email, phone, cpf_cnpj)
		VALUES ($1::uuid,$2,$3,$4,$5,$6,$7)
		ON CONFLICT (tenant_id, provider) DO UPDATE SET external_id=EXCLUDED.external_id,
			name=EXCLUDED.name, email=EXCLUDED.email, phone=EXCLUDED.phone, cpf_cnpj=EXCLUDED.cpf_cnpj,
			updated_at=now()`,
		tenantID, provider, cust.ExternalID, cust.Name, cust.Email, cust.Phone, cust.CPFCNPJ); err != nil {
		return "", err
	}
	return cust.ExternalID, nil
}

// Subscribe cria (ou substitui) a assinatura da igreja no gateway.
func (s *Service) Subscribe(ctx context.Context, tenantID string, in SubscribeInput) (*TenantBilling, error) {
	gw, err := s.PlatformGateway()
	if err != nil || !s.Enabled() {
		return nil, ErrDisabled
	}
	provider := gw.Provider()

	var tenantName, currentPlan string
	if err := s.Store.WithSystem(ctx, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `SELECT name, plan FROM tenants WHERE id=$1::uuid`, tenantID).
			Scan(&tenantName, &currentPlan)
	}); err != nil {
		return nil, err
	}
	if in.PlanKey == "" {
		in.PlanKey = currentPlan
	}

	// Plano define preco/ciclo quando nao sobrescritos.
	var planPrice int64
	var planCycle string
	if err := s.Store.WithSystem(ctx, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `SELECT price_cents, COALESCE(NULLIF(cycle,''),'MONTHLY') FROM plans WHERE key=$1`, in.PlanKey).
			Scan(&planPrice, &planCycle)
	}); err != nil {
		return nil, err
	}
	value := planPrice
	if in.ValueCents != nil {
		value = *in.ValueCents
	}
	cycle := in.Cycle
	if cycle == "" {
		cycle = planCycle
	}
	billingType := in.BillingType
	if billingType == "" {
		billingType = BillingPix
	}
	nextDue := in.NextDueDate
	if nextDue == "" {
		nextDue = time.Now().Format("2006-01-02")
	}

	var subscriptionID string
	err = s.Store.WithSystem(ctx, func(tx pgx.Tx) error {
		externalCustomer, e := s.ensureCustomer(ctx, tx, tenantID, provider, gw, in, tenantName)
		if e != nil {
			return e
		}

		// Cancela assinatura viva anterior, se houver.
		if prev, e := scanActiveSubscription(ctx, tx, tenantID); e == nil {
			if prev.ExternalID != "" {
				_ = gw.CancelSubscription(ctx, prev.ExternalID)
			}
			if _, e := tx.Exec(ctx, `UPDATE billing_subscriptions SET status='canceled', canceled_at=now(), updated_at=now() WHERE id=$1::uuid`, prev.ID); e != nil {
				return e
			}
		} else if !store.IsNotFound(e) {
			return e
		}

		sub, e := gw.CreateSubscription(ctx, SubscriptionInput{
			CustomerExternalID: externalCustomer,
			ExternalReference:  "tenant:" + tenantID,
			Description:        "Assinatura Chosen ERP - " + in.PlanKey,
			ValueCents:         value,
			Cycle:              cycle,
			BillingType:        billingType,
			NextDueDate:        nextDue,
			Card:               in.Card,
		})
		if e != nil {
			return e
		}

		if e := tx.QueryRow(ctx, `
			INSERT INTO billing_subscriptions
				(tenant_id, plan_key, provider, external_id, billing_type, cycle, value_cents, next_due_date, status)
			VALUES ($1::uuid,$2,$3,$4,$5,$6,$7,$8::date,$9)
			RETURNING id`,
			tenantID, in.PlanKey, provider, sub.ExternalID, billingType, cycle, value,
			nullIfEmpty(sub.NextDueDate), sub.Status).Scan(&subscriptionID); e != nil {
			return e
		}

		_, e = tx.Exec(ctx, `
			UPDATE tenants SET plan=$2, billing_status='active', past_due_since=NULL, updated_at=now()
			WHERE id=$1::uuid`, tenantID, in.PlanKey)
		return e
	})
	if err != nil {
		return nil, err
	}
	// Sincroniza as faturas geradas pela assinatura.
	_, _ = s.SyncTenant(ctx, tenantID)
	return s.GetTenantBilling(ctx, tenantID)
}

// CancelSubscription cancela a assinatura da igreja.
func (s *Service) CancelSubscription(ctx context.Context, tenantID string) error {
	gw, err := s.PlatformGateway()
	if err != nil {
		return ErrDisabled
	}
	return s.Store.WithSystem(ctx, func(tx pgx.Tx) error {
		sub, e := scanActiveSubscription(ctx, tx, tenantID)
		if store.IsNotFound(e) {
			return ErrNoSubscription
		}
		if e != nil {
			return e
		}
		if sub.ExternalID != "" {
			_ = gw.CancelSubscription(ctx, sub.ExternalID)
		}
		if _, e := tx.Exec(ctx, `UPDATE billing_subscriptions SET status='canceled', canceled_at=now(), updated_at=now() WHERE id=$1::uuid`, sub.ID); e != nil {
			return e
		}
		_, e = tx.Exec(ctx, `UPDATE tenants SET billing_status='canceled', updated_at=now() WHERE id=$1::uuid`, tenantID)
		return e
	})
}

// CreateCharge cria uma cobranca avulsa (Fluxo A).
func (s *Service) CreateCharge(ctx context.Context, tenantID string, in ChargeInputView) (*BillingInvoice, error) {
	gw, err := s.PlatformGateway()
	if err != nil || !s.Enabled() {
		return nil, ErrDisabled
	}
	provider := gw.Provider()
	due := in.DueDate
	if due == "" {
		due = time.Now().Format("2006-01-02")
	}
	billingType := in.BillingType
	if billingType == "" {
		billingType = BillingPix
	}
	var inv *BillingInvoice
	err = s.Store.WithSystem(ctx, func(tx pgx.Tx) error {
		var tenantName string
		if e := tx.QueryRow(ctx, `SELECT name FROM tenants WHERE id=$1::uuid`, tenantID).Scan(&tenantName); e != nil {
			return e
		}
		customerID, e := s.ensureCustomer(ctx, tx, tenantID, provider, gw, SubscribeInput{
			Email: in.Email, Phone: in.Phone, CPFCNPJ: in.CPFCNPJ,
		}, tenantName)
		if e != nil {
			return e
		}
		sub, _ := scanActiveSubscription(ctx, tx, tenantID)
		var subID any
		if sub != nil {
			subID = sub.ID
		}
		charge, e := gw.CreateCharge(ctx, ChargeInput{
			CustomerExternalID: customerID,
			ExternalReference:  "tenant:" + tenantID,
			Description:        in.Description,
			ValueCents:         in.ValueCents,
			DueDate:            due,
			BillingType:        billingType,
			Card:               in.Card,
		})
		if e != nil {
			return e
		}
		var qr *PixQRCode
		if billingType == BillingPix && charge.ExternalID != "" {
			if q, e := gw.PixQRCode(ctx, charge.ExternalID); e == nil {
				qr = q
			}
		}
		return upsertInvoice(ctx, tx, tenantID, subID, provider, charge, qr)
	})
	if err != nil {
		return nil, err
	}
	// Recarrega a fatura recem-criada.
	tb, err := s.GetTenantBilling(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	if len(tb.Invoices) > 0 {
		inv = &tb.Invoices[0]
	}
	return inv, nil
}

// RefundInvoice estorna uma fatura.
func (s *Service) RefundInvoice(ctx context.Context, tenantID, invoiceID string) error {
	gw, err := s.PlatformGateway()
	if err != nil {
		return ErrDisabled
	}
	return s.Store.WithSystem(ctx, func(tx pgx.Tx) error {
		var externalID string
		if e := tx.QueryRow(ctx,
			`SELECT COALESCE(external_id,'') FROM billing_invoices WHERE id=$1::uuid AND tenant_id=$2::uuid`,
			invoiceID, tenantID).Scan(&externalID); e != nil {
			return e
		}
		if externalID == "" {
			return errors.New("fatura sem id no gateway")
		}
		charge, e := gw.RefundCharge(ctx, externalID, nil)
		if e != nil {
			return e
		}
		_, e = tx.Exec(ctx,
			`UPDATE billing_invoices SET status=$2, raw=$3::jsonb, updated_at=now() WHERE id=$1::uuid`,
			invoiceID, charge.Status, string(rawJSON(charge)))
		return e
	})
}

// SyncTenant busca as cobrancas da assinatura no gateway e atualiza as faturas.
func (s *Service) SyncTenant(ctx context.Context, tenantID string) (*TenantBilling, error) {
	gw, err := s.PlatformGateway()
	if err != nil || !s.Enabled() {
		return nil, ErrDisabled
	}
	err = s.Store.WithSystem(ctx, func(tx pgx.Tx) error {
		sub, e := scanActiveSubscription(ctx, tx, tenantID)
		if store.IsNotFound(e) || (sub != nil && sub.ExternalID == "") {
			return nil
		}
		if e != nil {
			return e
		}
		if sub.ExternalID != "" {
			if remote, e := gw.GetSubscription(ctx, sub.ExternalID); e == nil {
				_, _ = tx.Exec(ctx, `
					UPDATE billing_subscriptions SET status=$2, next_due_date=NULLIF($3,'')::date,
						value_cents=$4, updated_at=now() WHERE id=$1::uuid`,
					sub.ID, remote.Status, remote.NextDueDate, remote.ValueCents)
			}
		}
		// Atualiza faturas conhecidas.
		rows, e := tx.Query(ctx,
			`SELECT id, external_id FROM billing_invoices WHERE tenant_id=$1::uuid AND external_id IS NOT NULL
			 AND status IN ('pending','overdue')`, tenantID)
		if e != nil {
			return e
		}
		type ref struct{ id, ext string }
		var refs []ref
		for rows.Next() {
			var r ref
			if e := rows.Scan(&r.id, &r.ext); e != nil {
				rows.Close()
				return e
			}
			refs = append(refs, r)
		}
		rows.Close()
		for _, r := range refs {
			charge, e := gw.GetCharge(ctx, r.ext)
			if e != nil {
				continue
			}
			if _, e := tx.Exec(ctx, `
				UPDATE billing_invoices SET status=$2, paid_at=NULLIF($3,'')::timestamptz,
					updated_at=now() WHERE id=$1::uuid`,
				r.id, charge.Status, charge.PaidAt); e != nil {
				return e
			}
		}
		// Atualiza o status de cobranca do tenant.
		return s.recomputeTenantStatus(ctx, tx, tenantID)
	})
	if err != nil {
		return nil, err
	}
	return s.GetTenantBilling(ctx, tenantID)
}

// recomputeTenantStatus deriva billing_status/past_due_since da ultima fatura.
func (s *Service) recomputeTenantStatus(ctx context.Context, tx pgx.Tx, tenantID string) error {
	var overdue int
	var dueSince *time.Time
	if err := tx.QueryRow(ctx, `
		SELECT COUNT(*), MIN(due_date)::timestamptz
		FROM billing_invoices
		WHERE tenant_id=$1::uuid AND status IN ('overdue')`, tenantID).Scan(&overdue, &dueSince); err != nil {
		return err
	}
	if overdue > 0 {
		_, err := tx.Exec(ctx, `
			UPDATE tenants SET billing_status='overdue',
				past_due_since=COALESCE(past_due_since,$2), updated_at=now()
			WHERE id=$1::uuid AND billing_exempt=false`, tenantID, dueSince)
		return err
	}
	_, err := tx.Exec(ctx, `
		UPDATE tenants SET billing_status='active', past_due_since=NULL,
			is_active = CASE WHEN suspended_at IS NOT NULL THEN true ELSE is_active END,
			suspended_at = CASE WHEN suspended_at IS NOT NULL THEN NULL ELSE suspended_at END,
			updated_at=now()
		WHERE id=$1::uuid AND billing_status IN ('overdue','suspended')`, tenantID)
	return err
}

// upsertInvoice grava/atualiza uma fatura a partir de uma cobranca do gateway.
func upsertInvoice(ctx context.Context, tx pgx.Tx, tenantID string, subID any, provider string, ch *Charge, qr *PixQRCode) error {
	payload, image := "", ""
	if qr != nil {
		payload, image = qr.Payload, qr.EncodedImage
	}
	var id string
	return tx.QueryRow(ctx, `
		INSERT INTO billing_invoices
			(tenant_id, subscription_id, provider, external_id, billing_type, status, value_cents,
			 net_value_cents, due_date, paid_at, invoice_url, bank_slip_url, pix_payload, pix_qr_image, raw)
		VALUES ($1::uuid,$2,$3,$4,$5,$6,$7,$8,NULLIF($9,'')::date,NULLIF($10,'')::timestamptz,
		        $11,$12,$13,$14,$15::jsonb)
		ON CONFLICT (provider, external_id) WHERE external_id IS NOT NULL
		DO UPDATE SET status=EXCLUDED.status, paid_at=EXCLUDED.paid_at,
			pix_payload=COALESCE(NULLIF(EXCLUDED.pix_payload,''), billing_invoices.pix_payload),
			pix_qr_image=COALESCE(NULLIF(EXCLUDED.pix_qr_image,''), billing_invoices.pix_qr_image),
			raw=EXCLUDED.raw, updated_at=now()
		RETURNING id`,
		tenantID, subID, provider, nullIfEmpty(ch.ExternalID), ch.BillingType, ch.Status,
		ch.ValueCents, ch.NetValueCents, ch.DueDate, ch.PaidAt, ch.InvoiceURL, ch.BankSlipURL,
		payload, image, string(rawJSON(ch))).Scan(&id)
}

func rawJSON(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		return []byte("{}")
	}
	return b
}

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}
