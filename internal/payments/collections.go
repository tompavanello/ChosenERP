package payments

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"chosenerp/internal/store"
)

// ErrProductInvalido sinaliza produto sem nome/categoria.
var ErrProductInvalido = errors.New("nome e categoria do produto sao obrigatorios")

// ErrProductAccount exige que o produto tenha uma conta bancaria ativa.
var ErrProductAccount = errors.New("selecione uma conta bancaria ativa para o produto")

func validateProductAccount(ctx context.Context, tx pgx.Tx, accountID string) error {
	if strings.TrimSpace(accountID) == "" {
		return ErrProductAccount
	}
	var active bool
	if err := tx.QueryRow(ctx, `SELECT is_active FROM financial_accounts WHERE id = $1::uuid`, accountID).Scan(&active); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return errors.New("conta bancaria do produto nao encontrada no escopo")
		}
		return err
	}
	if !active {
		return ErrProductAccount
	}
	return nil
}

// ---- Configuracao de gateway por igreja (Fluxo B) ----

// GatewayConfig e a visao segura de uma config (sem credenciais).
type GatewayConfig struct {
	ID             string `json:"id"`
	Provider       string `json:"provider"`
	Environment    string `json:"environment"`
	IsActive       bool   `json:"is_active"`
	IsDefault      bool   `json:"is_default"`
	HasCredentials bool   `json:"has_credentials"`
}

// GatewayConfigInput cria/atualiza uma config. APIKey/WebhookToken vazios
// preservam os valores ja gravados.
type GatewayConfigInput struct {
	Provider     string `json:"provider"`
	Environment  string `json:"environment"`
	APIKey       string `json:"api_key"`
	BaseURL      string `json:"base_url"`
	WebhookToken string `json:"webhook_token"`
	IsActive     *bool  `json:"is_active"`
	IsDefault    *bool  `json:"is_default"`
}

// ListGatewayConfigs lista as configs da igreja (sem credenciais).
func (s *Service) ListGatewayConfigs(ctx context.Context, b store.Bounds) ([]GatewayConfig, error) {
	out := []GatewayConfig{}
	err := s.Store.WithTenant(ctx, b, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `
			SELECT id, provider, environment, is_active, is_default,
			       (credentials ? 'enc' OR length(credentials::text) > 2)
			FROM payment_gateway_configs
			WHERE tenant_id = $1::uuid
			ORDER BY is_default DESC, provider`, b.TenantID)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var c GatewayConfig
			if err := rows.Scan(&c.ID, &c.Provider, &c.Environment, &c.IsActive, &c.IsDefault, &c.HasCredentials); err != nil {
				return err
			}
			out = append(out, c)
		}
		return rows.Err()
	})
	return out, err
}

// SaveGatewayConfig grava as credenciais cifradas de um provider da igreja.
func (s *Service) SaveGatewayConfig(ctx context.Context, b store.Bounds, in GatewayConfigInput) (*GatewayConfig, error) {
	if in.Provider == "" {
		return nil, errors.New("provider obrigatorio")
	}
	env := in.Environment
	if env == "" {
		env = "sandbox"
	}
	var out GatewayConfig
	err := s.Store.WithTenant(ctx, b, func(tx pgx.Tx) error {
		// Credenciais existentes (para preservar segredos nao reenviados).
		existing := Credentials{}
		var existingRaw []byte
		err := tx.QueryRow(ctx, `
			SELECT credentials FROM payment_gateway_configs
			WHERE tenant_id=$1::uuid AND provider=$2`, b.TenantID, in.Provider).Scan(&existingRaw)
		if err == nil {
			if raw, e := DecryptCredentials(s.credKey(), existingRaw); e == nil && len(raw) > 0 {
				_ = json.Unmarshal(raw, &existing)
			}
		} else if !store.IsNotFound(err) {
			return err
		}
		creds := Credentials{
			APIKey:       firstNonEmpty(in.APIKey, existing.APIKey),
			BaseURL:      firstNonEmpty(in.BaseURL, existing.BaseURL),
			WebhookToken: firstNonEmpty(in.WebhookToken, existing.WebhookToken),
		}
		raw, _ := json.Marshal(creds)
		enc, e := EncryptCredentials(s.credKey(), raw)
		if e != nil {
			return e
		}
		isDefault := in.IsDefault != nil && *in.IsDefault
		if isDefault {
			if _, e := tx.Exec(ctx,
				`UPDATE payment_gateway_configs SET is_default=false, updated_at=now() WHERE tenant_id=$1::uuid`,
				b.TenantID); e != nil {
				return e
			}
		}
		return tx.QueryRow(ctx, `
			INSERT INTO payment_gateway_configs (tenant_id, provider, environment, credentials, is_active, is_default)
			VALUES ($1::uuid,$2,$3,$4::jsonb,COALESCE($5,true),COALESCE($6,false))
			ON CONFLICT (tenant_id, provider) DO UPDATE SET
				environment=EXCLUDED.environment, credentials=EXCLUDED.credentials,
				is_active=COALESCE($5, payment_gateway_configs.is_active),
				is_default=COALESCE($6, payment_gateway_configs.is_default),
				updated_at=now()
			RETURNING id, provider, environment, is_active, is_default, true`,
			b.TenantID, in.Provider, env, string(enc), in.IsActive, in.IsDefault).
			Scan(&out.ID, &out.Provider, &out.Environment, &out.IsActive, &out.IsDefault, &out.HasCredentials)
	})
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// TenantGateway resolve o gateway ativo da igreja (provider vazio = default).
func (s *Service) TenantGateway(ctx context.Context, b store.Bounds, provider string) (Gateway, string, error) {
	if s.Factory == nil {
		return nil, "", ErrDisabled
	}
	var creds Credentials
	var chosen string
	err := s.Store.WithTenant(ctx, b, func(tx pgx.Tx) error {
		var raw []byte
		err := tx.QueryRow(ctx, `
			SELECT provider, credentials FROM payment_gateway_configs
			WHERE tenant_id=$1::uuid AND is_active=true
			  AND ($2='' OR provider=$2)
			ORDER BY is_default DESC, created_at ASC LIMIT 1`, b.TenantID, provider).
			Scan(&chosen, &raw)
		if err != nil {
			return err
		}
		dec, e := DecryptCredentials(s.credKey(), raw)
		if e != nil {
			return e
		}
		return json.Unmarshal(dec, &creds)
	})
	if err != nil {
		if store.IsNotFound(err) {
			return nil, "", errors.New("nenhum gateway configurado para a igreja")
		}
		return nil, "", err
	}
	gw, err := s.Factory(chosen, creds)
	if err != nil {
		return nil, "", err
	}
	return gw, chosen, nil
}

// ---- Produtos (Fluxo B) ----

// Product e um produto oferecido pela igreja.
type Product struct {
	ID                  string   `json:"id"`
	BranchID            string   `json:"branch_id"`
	Name                string   `json:"name"`
	Description         string   `json:"description"`
	AmountCents         *int64   `json:"amount_cents"`
	AllowedBillingTypes []string `json:"allowed_billing_types"`
	CategoryID          string   `json:"category_id"`
	CategoryName        string   `json:"category_name"`
	AccountID           *string  `json:"account_id"`
	IsRecurring         bool     `json:"is_recurring"`
	IsActive            bool     `json:"is_active"`
	SortOrder           int      `json:"sort_order"`
}

// ProductInput cria/edita um produto.
type ProductInput struct {
	Name                string   `json:"name"`
	Description         string   `json:"description"`
	AmountCents         *int64   `json:"amount_cents"`
	AllowedBillingTypes []string `json:"allowed_billing_types"`
	CategoryID          string   `json:"category_id"`
	AccountID           string   `json:"account_id"`
	IsRecurring         *bool    `json:"is_recurring"`
	IsActive            *bool    `json:"is_active"`
	SortOrder           *int     `json:"sort_order"`
}

const productCols = `p.id, COALESCE(p.branch_id::text,''), p.name, COALESCE(p.description,''),
	p.amount_cents, COALESCE(p.allowed_billing_types, ARRAY[]::text[]), p.category_id,
	COALESCE(c.name,''), p.account_id, p.is_recurring, p.is_active, p.sort_order`

func scanProduct(row pgx.Row) (*Product, error) {
	var p Product
	err := row.Scan(&p.ID, &p.BranchID, &p.Name, &p.Description, &p.AmountCents,
		&p.AllowedBillingTypes, &p.CategoryID, &p.CategoryName, &p.AccountID,
		&p.IsRecurring, &p.IsActive, &p.SortOrder)
	return &p, err
}

// ListProducts lista o catalogo de produtos da igreja.
func (s *Service) ListProducts(ctx context.Context, b store.Bounds) ([]Product, error) {
	out := []Product{}
	err := s.Store.WithTenant(ctx, b, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `
			SELECT `+productCols+`
			FROM payment_products p
			LEFT JOIN financial_categories c ON c.id = p.category_id
			WHERE p.tenant_id = $1::uuid
			ORDER BY p.sort_order, p.name`, b.TenantID)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			p, err := scanProduct(rows)
			if err != nil {
				return err
			}
			out = append(out, *p)
		}
		return rows.Err()
	})
	return out, err
}

// CreateProduct insere um produto.
func (s *Service) CreateProduct(ctx context.Context, b store.Bounds, in ProductInput) (*Product, error) {
	if in.Name == "" || in.CategoryID == "" {
		return nil, ErrProductInvalido
	}
	var id string
	err := s.Store.WithTenant(ctx, b, func(tx pgx.Tx) error {
		if err := validateProductAccount(ctx, tx, in.AccountID); err != nil {
			return err
		}
		return tx.QueryRow(ctx, `
			INSERT INTO payment_products
				(tenant_id, branch_id, name, description, amount_cents, allowed_billing_types,
				 category_id, account_id, is_recurring, is_active, sort_order)
			VALUES ($1::uuid, NULLIF($2,'')::uuid, $3, $4, $5, COALESCE($6, ARRAY['PIX','BOLETO','CREDIT_CARD']::text[]),
			        $7::uuid, NULLIF($8,'')::uuid, COALESCE($9,false), COALESCE($10,true), COALESCE($11,0))
			RETURNING id`,
			b.TenantID, b.BranchID, in.Name, in.Description, in.AmountCents,
			nullIfEmptyStrings(in.AllowedBillingTypes), in.CategoryID, in.AccountID,
			in.IsRecurring, in.IsActive, in.SortOrder).Scan(&id)
	})
	if err != nil {
		return nil, err
	}
	return s.getProduct(ctx, b, id)
}

// UpdateProduct edita um produto.
func (s *Service) UpdateProduct(ctx context.Context, b store.Bounds, id string, in ProductInput) (*Product, error) {
	err := s.Store.WithTenant(ctx, b, func(tx pgx.Tx) error {
		if err := validateProductAccount(ctx, tx, in.AccountID); err != nil {
			return err
		}
		tag, e := tx.Exec(ctx, `
			UPDATE payment_products SET
				name=COALESCE(NULLIF($3,''), name),
				description=$4,
				amount_cents=$5,
				allowed_billing_types=COALESCE($6, allowed_billing_types),
				category_id=COALESCE(NULLIF($7,'')::uuid, category_id),
				account_id=NULLIF($8,'')::uuid,
				is_recurring=COALESCE($9, is_recurring),
				is_active=COALESCE($10, is_active),
				sort_order=COALESCE($11, sort_order),
				updated_at=now()
			WHERE id=$1::uuid AND tenant_id=$2::uuid`,
			id, b.TenantID, in.Name, in.Description, in.AmountCents,
			nullIfEmptyStrings(in.AllowedBillingTypes), in.CategoryID, in.AccountID,
			in.IsRecurring, in.IsActive, in.SortOrder)
		if e != nil {
			return e
		}
		if tag.RowsAffected() == 0 {
			return pgx.ErrNoRows
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return s.getProduct(ctx, b, id)
}

// DeleteProduct remove um produto (cobrancas antigas perdem o vinculo).
func (s *Service) DeleteProduct(ctx context.Context, b store.Bounds, id string) error {
	return s.Store.WithTenant(ctx, b, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `DELETE FROM payment_products WHERE id=$1::uuid AND tenant_id=$2::uuid`, id, b.TenantID)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return pgx.ErrNoRows
		}
		return nil
	})
}

func (s *Service) getProduct(ctx context.Context, b store.Bounds, id string) (*Product, error) {
	var out *Product
	err := s.Store.WithTenant(ctx, b, func(tx pgx.Tx) error {
		var e error
		out, e = scanProduct(tx.QueryRow(ctx, `
			SELECT `+productCols+`
			FROM payment_products p
			LEFT JOIN financial_categories c ON c.id = p.category_id
			WHERE p.id=$1::uuid AND p.tenant_id=$2::uuid`, id, b.TenantID))
		return e
	})
	return out, err
}

// ---- Cobrancas (Fluxo B) ----

// ChargeB e uma cobranca de arrecadacao.
type ChargeB struct {
	ID                     string     `json:"id"`
	ProductID              string     `json:"product_id"`
	MemberID               string     `json:"member_id"`
	Provider               string     `json:"provider"`
	ExternalID             string     `json:"external_id"`
	BillingType            string     `json:"billing_type"`
	Status                 string     `json:"status"`
	AmountCents            int64      `json:"amount_cents"`
	DueDate                *string    `json:"due_date"`
	PaidAt                 *time.Time `json:"paid_at"`
	PayerName              string     `json:"payer_name"`
	PayerCPFCNPJ           string     `json:"payer_cpf_cnpj"`
	PayerEmail             string     `json:"payer_email"`
	Description            string     `json:"description"`
	InvoiceURL             string     `json:"invoice_url"`
	BankSlipURL            string     `json:"bank_slip_url"`
	PixPayload             string     `json:"pix_payload"`
	PixQRImage             string     `json:"pix_qr_image"`
	FinancialTransactionID *string    `json:"financial_transaction_id"`
	CreatedAt              time.Time  `json:"created_at"`
}

// ChargeBInput cria uma cobranca de arrecadacao.
type ChargeBInput struct {
	ProductID    string `json:"product_id"`
	MemberID     string `json:"member_id"`
	BillingType  string `json:"billing_type"`
	AmountCents  *int64 `json:"amount_cents"`
	DueDate      string `json:"due_date"`
	PayerName    string `json:"payer_name"`
	PayerCPFCNPJ string `json:"payer_cpf_cnpj"`
	PayerEmail   string `json:"payer_email"`
	PayerPhone   string `json:"payer_phone"`
	Description  string `json:"description"`
	Card         *Card  `json:"card"`
}

const chargeBCols = `id, COALESCE(product_id::text,''), COALESCE(member_id::text,''), provider,
	COALESCE(external_id,''), billing_type, status, amount_cents, to_char(due_date,'YYYY-MM-DD'),
	paid_at, COALESCE(payer_name,''), COALESCE(payer_cpf_cnpj,''), COALESCE(payer_email,''),
	COALESCE(description,''), COALESCE(invoice_url,''), COALESCE(bank_slip_url,''),
	COALESCE(pix_payload,''), COALESCE(pix_qr_image,''), financial_transaction_id::text, created_at`

func scanChargeB(row pgx.Row) (*ChargeB, error) {
	var c ChargeB
	err := row.Scan(&c.ID, &c.ProductID, &c.MemberID, &c.Provider, &c.ExternalID,
		&c.BillingType, &c.Status, &c.AmountCents, &c.DueDate, &c.PaidAt,
		&c.PayerName, &c.PayerCPFCNPJ, &c.PayerEmail, &c.Description,
		&c.InvoiceURL, &c.BankSlipURL, &c.PixPayload, &c.PixQRImage,
		&c.FinancialTransactionID, &c.CreatedAt)
	return &c, err
}

// ListCharges lista as cobrancas da igreja.
func (s *Service) ListCharges(ctx context.Context, b store.Bounds, status string) ([]ChargeB, error) {
	out := []ChargeB{}
	err := s.Store.WithTenant(ctx, b, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `
			SELECT `+chargeBCols+`
			FROM payment_charges
			WHERE tenant_id=$1::uuid AND ($2='' OR status=$2)
			ORDER BY created_at DESC LIMIT 200`, b.TenantID, status)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			c, err := scanChargeB(rows)
			if err != nil {
				return err
			}
			out = append(out, *c)
		}
		return rows.Err()
	})
	return out, err
}

// CreateChargeB cria uma cobranca (PIX/boleto/cartao) na conta da igreja.
func (s *Service) CreateChargeB(ctx context.Context, b store.Bounds, in ChargeBInput) (*ChargeB, error) {
	if in.ProductID == "" {
		return nil, errors.New("produto obrigatorio")
	}
	gw, provider, err := s.TenantGateway(ctx, b, "")
	if err != nil {
		return nil, err
	}
	due := in.DueDate
	if due == "" {
		due = time.Now().Format("2006-01-02")
	}
	billingType := in.BillingType
	if billingType == "" {
		billingType = BillingPix
	}

	var chargeID string
	err = s.Store.WithTenant(ctx, b, func(tx pgx.Tx) error {
		// Produto define valor/categoria.
		var productAmount *int64
		var productName string
		if err := tx.QueryRow(ctx, `
			SELECT amount_cents, name FROM payment_products
			WHERE id=$1::uuid AND tenant_id=$2::uuid`, in.ProductID, b.TenantID).
			Scan(&productAmount, &productName); err != nil {
			return err
		}
		amount := in.AmountCents
		if amount == nil {
			amount = productAmount
		}
		if amount == nil || *amount <= 0 {
			return errors.New("valor da cobranca obrigatorio")
		}

		// O membro doador, quando informado, precisa pertencer a esta igreja.
		if in.MemberID != "" {
			var exists bool
			if err := tx.QueryRow(ctx,
				`SELECT true FROM members WHERE id=$1::uuid AND tenant_id=$2::uuid`,
				in.MemberID, b.TenantID).Scan(&exists); err != nil {
				if store.IsNotFound(err) {
					return errors.New("membro nao encontrado")
				}
				return err
			}
		}

		customerID, e := s.ensurePayerCustomer(ctx, tx, b.TenantID, provider, gw, in)
		if e != nil {
			return e
		}
		desc := in.Description
		if desc == "" {
			desc = productName
		}
		ch, e := gw.CreateCharge(ctx, ChargeInput{
			CustomerExternalID: customerID,
			ExternalReference:  "tenant:" + b.TenantID,
			Description:        desc,
			ValueCents:         *amount,
			DueDate:            due,
			BillingType:        billingType,
			Card:               in.Card,
		})
		if e != nil {
			return e
		}
		var qr *PixQRCode
		if billingType == BillingPix && ch.ExternalID != "" {
			if q, e := gw.PixQRCode(ctx, ch.ExternalID); e == nil {
				qr = q
			}
		}
		var pixPayload, pixImage string
		if qr != nil {
			pixPayload, pixImage = qr.Payload, qr.EncodedImage
		}
		return tx.QueryRow(ctx, `
			INSERT INTO payment_charges
				(tenant_id, branch_id, product_id, member_id, provider, external_id, billing_type,
				 amount_cents, status, due_date, paid_at, payer_name, payer_cpf_cnpj, payer_email,
				 description, invoice_url, bank_slip_url, pix_payload, pix_qr_image, raw)
			VALUES ($1::uuid, NULLIF($2,'')::uuid, NULLIF($3,'')::uuid, NULLIF($4,'')::uuid, $5, $6, $7,
			        $8, $9, NULLIF($10,'')::date, NULLIF($11,'')::timestamptz,
			        $12, $13, $14, $15, $16, $17, $18, $19, $20::jsonb)
			RETURNING id`,
			b.TenantID, b.BranchID, in.ProductID, in.MemberID, provider, nullIfEmpty(ch.ExternalID),
			billingType, *amount, ch.Status, ch.DueDate, ch.PaidAt, in.PayerName, in.PayerCPFCNPJ,
			in.PayerEmail, desc, ch.InvoiceURL, ch.BankSlipURL, pixPayload, pixImage, string(rawJSON(ch))).
			Scan(&chargeID)
	})
	if err != nil {
		return nil, err
	}
	return s.getChargeB(ctx, b, chargeID)
}

// RefundChargeB estorna uma cobranca de arrecadacao.
func (s *Service) RefundChargeB(ctx context.Context, b store.Bounds, id string) error {
	gw, provider, err := s.TenantGateway(ctx, b, "")
	if err != nil {
		return err
	}
	return s.Store.WithTenant(ctx, b, func(tx pgx.Tx) error {
		var externalID string
		if e := tx.QueryRow(ctx,
			`SELECT COALESCE(external_id,'') FROM payment_charges WHERE id=$1::uuid AND tenant_id=$2::uuid`,
			id, b.TenantID).Scan(&externalID); e != nil {
			return e
		}
		if externalID == "" {
			return errors.New("cobranca sem id no gateway")
		}
		ch, e := gw.RefundCharge(ctx, externalID, nil)
		if e != nil {
			return e
		}
		_, e = tx.Exec(ctx,
			`UPDATE payment_charges SET status=$3, raw=$4::jsonb, updated_at=now() WHERE id=$1::uuid AND provider=$2`,
			id, provider, ch.Status, string(rawJSON(ch)))
		return e
	})
}

func (s *Service) getChargeB(ctx context.Context, b store.Bounds, id string) (*ChargeB, error) {
	var out *ChargeB
	err := s.Store.WithTenant(ctx, b, func(tx pgx.Tx) error {
		var e error
		out, e = scanChargeB(tx.QueryRow(ctx, `
			SELECT `+chargeBCols+` FROM payment_charges WHERE id=$1::uuid AND tenant_id=$2::uuid`,
			id, b.TenantID))
		return e
	})
	return out, err
}

// ensurePayerCustomer reaproveita/cria o cliente do pagador no gateway.
func (s *Service) ensurePayerCustomer(ctx context.Context, tx pgx.Tx, tenantID, provider string, gw Gateway, in ChargeBInput) (string, error) {
	cpf := onlyDigits(in.PayerCPFCNPJ)
	if cpf != "" {
		var externalID string
		err := tx.QueryRow(ctx, `
			SELECT external_id FROM payment_customers
			WHERE tenant_id=$1::uuid AND provider=$2 AND cpf_cnpj=$3`, tenantID, provider, cpf).Scan(&externalID)
		if err == nil {
			return externalID, nil
		}
		if !store.IsNotFound(err) {
			return "", err
		}
	}
	cust, err := gw.CreateCustomer(ctx, CustomerInput{
		ExternalReference: "tenant:" + tenantID + ":payer",
		Name:              firstNonEmpty(in.PayerName, "Contribuinte"),
		Email:             in.PayerEmail,
		Phone:             in.PayerPhone,
		CPFCNPJ:           in.PayerCPFCNPJ,
	})
	if err != nil {
		return "", err
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO payment_customers (tenant_id, provider, external_id, cpf_cnpj, name, email, phone)
		VALUES ($1::uuid,$2,$3,NULLIF($4,''),$5,$6,$7)
		ON CONFLICT (tenant_id, provider, cpf_cnpj) DO UPDATE SET external_id=EXCLUDED.external_id,
			name=EXCLUDED.name, email=EXCLUDED.email, phone=EXCLUDED.phone, updated_at=now()`,
		tenantID, provider, cust.ExternalID, cpf, cust.Name, cust.Email, cust.Phone)
	return cust.ExternalID, err
}

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

func nullIfEmptyStrings(v []string) any {
	if len(v) == 0 {
		return nil
	}
	return v
}

func onlyDigits(s string) string {
	var b []byte
	for i := 0; i < len(s); i++ {
		if s[i] >= '0' && s[i] <= '9' {
			b = append(b, s[i])
		}
	}
	return string(b)
}
