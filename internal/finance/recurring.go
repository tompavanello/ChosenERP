package finance

import (
	"context"
	"log"
	"time"

	"github.com/jackc/pgx/v5"

	"chosenerp/internal/documents"
	"chosenerp/internal/store"
)

// Recurring e um agendamento de doacao recorrente (dizimo/oferta).
type Recurring struct {
	ID            string     `json:"id"`
	BranchID      string     `json:"branch_id"`
	MemberID      *string    `json:"member_id,omitempty"`
	MemberName    *string    `json:"member_name,omitempty"`
	BenefactorID  *string    `json:"benefactor_id,omitempty"`
	CategoryID    *string    `json:"category_id,omitempty"`
	CategoryName  *string    `json:"category_name,omitempty"`
	AccountID     *string    `json:"account_id,omitempty"`
	AccountName   *string    `json:"account_name,omitempty"`
	Subtype       string     `json:"subtype"`
	Amount        float64    `json:"amount"`
	Frequency     string     `json:"frequency"`
	NextRunAt     time.Time  `json:"next_run_at"`
	LastRunAt     *time.Time `json:"last_run_at,omitempty"`
	TimesRun      int        `json:"times_run"`
	IsActive      bool       `json:"is_active"`
	PaymentMethod *string    `json:"payment_method,omitempty"`
	Description   *string    `json:"description,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
}

type CreateRecurringInput struct {
	Subtype       string  `json:"subtype"`
	Amount        float64 `json:"amount"`
	Frequency     string  `json:"frequency"`
	CategoryID    *string `json:"category_id"`
	AccountID     *string `json:"account_id"`
	MemberID      *string `json:"member_id"`
	BenefactorID  *string `json:"benefactor_id"`
	PaymentMethod *string `json:"payment_method"`
	Description   *string `json:"description"`
	PeriodStart   *string `json:"period_start"`
}

type UpdateRecurringInput struct {
	Amount    *float64 `json:"amount"`
	IsActive  *bool    `json:"is_active"`
	Subtype   *string  `json:"subtype"`
	Frequency *string  `json:"frequency"`
}

const (
	subtypeTithes       = "dizimo"
	subtypeOffering     = "oferta"
	subtypeDonation     = "doacao"
	frequencyWeekly     = "weekly"
	frequencyMonthly    = "monthly"
	frequencyYearly     = "yearly"
	recurringTypeIncome = "income"
)

// nextRunFor avanca a data de execucao ate o proximo momento >= now.
func nextRunFor(periodStart time.Time, frequency string, now time.Time) time.Time {
	if periodStart.IsZero() {
		periodStart = now
	}
	next := periodStart.AddDate(0, 0, 0)
	for !next.After(now) {
		switch frequency {
		case frequencyWeekly:
			next = next.AddDate(0, 0, 7)
		case frequencyYearly:
			next = next.AddDate(1, 0, 0)
		default: // monthly
			next = next.AddDate(0, 1, 0)
		}
	}
	return next
}

// normalizeRecurring valida e devolve o subtype normalizado (ou "" se invalido).
func normalizeRecurring(in CreateRecurringInput) (string, bool) {
	if in.Amount <= 0 {
		return "", false
	}
	sub := in.Subtype
	if sub == "" {
		sub = subtypeDonation
	}
	switch sub {
	case subtypeTithes, subtypeOffering, subtypeDonation:
	default:
		return "", false
	}
	switch in.Frequency {
	case frequencyWeekly, frequencyMonthly, frequencyYearly:
	default:
		return "", false
	}
	return sub, true
}

// CreateRecurring agenda uma doacao recorrente no escopo da filial.
func (r *Repo) CreateRecurring(ctx context.Context, tx pgx.Tx, tenantID, branchID string, in CreateRecurringInput) (*Recurring, error) {
	sub, ok := normalizeRecurring(in)
	if !ok {
		return nil, pgx.ErrNoRows
	}
	periodStart, err := parseOccurredAt(in.PeriodStart)
	if err != nil {
		return nil, err
	}
	next := nextRunFor(periodStart, in.Frequency, time.Now())

	var rec Recurring
	err = tx.QueryRow(ctx, `
		INSERT INTO recurring_donations
			(tenant_id, branch_id, member_id, benefactor_id, category_id, subtype,
			 amount, frequency, period_start, next_run_at, payment_method, description, account_id)
		VALUES ($1, NULLIF($2,'')::uuid, NULLIF($3,'')::uuid, NULLIF($4,'')::uuid,
				NULLIF($5,'')::uuid, $6, $7, $8, $9, $10, $11, NULLIF($12,''), NULLIF($13,'')::uuid)
		RETURNING id::text, branch_id::text, subtype, amount::float8, frequency,
		          next_run_at, times_run, is_active, payment_method, description, created_at`,
		tenantID, branchID, in.MemberID, in.BenefactorID, in.CategoryID, sub,
		in.Amount, in.Frequency, periodStart, next, in.PaymentMethod, in.Description, in.AccountID).
		Scan(&rec.ID, &rec.BranchID, &rec.Subtype, &rec.Amount, &rec.Frequency,
			&rec.NextRunAt, &rec.TimesRun, &rec.IsActive, &rec.PaymentMethod, &rec.Description, &rec.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &rec, nil
}

// ListRecurring retorna os agendamentos visiveis no escopo.
func (r *Repo) ListRecurring(ctx context.Context, tx pgx.Tx) ([]Recurring, error) {
	rows, err := tx.Query(ctx, `
		SELECT rd.id::text, rd.branch_id::text, rd.member_id::text, m.full_name,
		       rd.benefactor_id::text, rd.category_id::text, c.name,
		       rd.account_id::text, a.name,
		       rd.subtype, rd.amount::float8, rd.frequency, rd.next_run_at,
		       rd.last_run_at, rd.times_run, rd.is_active, rd.payment_method,
		       rd.description, rd.created_at
		FROM recurring_donations rd
		LEFT JOIN members m ON m.id = rd.member_id
		LEFT JOIN financial_categories c ON c.id = rd.category_id
		LEFT JOIN financial_accounts a ON a.id = rd.account_id
		ORDER BY rd.is_active DESC, rd.next_run_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Recurring{}
	for rows.Next() {
		var rec Recurring
		var mid, bid, cid, aid, aname *string
		if err := rows.Scan(&rec.ID, &rec.BranchID, &mid, &rec.MemberName,
			&bid, &cid, &rec.CategoryName,
			&aid, &aname,
			&rec.Subtype, &rec.Amount, &rec.Frequency,
			&rec.NextRunAt, &rec.LastRunAt, &rec.TimesRun, &rec.IsActive,
			&rec.PaymentMethod, &rec.Description, &rec.CreatedAt); err != nil {
			return nil, err
		}
		if mid != nil && *mid != "" {
			rec.MemberID = mid
		}
		if bid != nil && *bid != "" {
			rec.BenefactorID = bid
		}
		if cid != nil && *cid != "" {
			rec.CategoryID = cid
		}
		if aid != nil && *aid != "" {
			rec.AccountID = aid
		}
		if aname != nil && *aname != "" {
			rec.AccountName = aname
		}
		out = append(out, rec)
	}
	return out, rows.Err()
}

// UpdateRecurring altera valor/status do agendamento (pausar/retomar/ajustar).
func (r *Repo) UpdateRecurring(ctx context.Context, tx pgx.Tx, id string, in UpdateRecurringInput) (*Recurring, error) {
	var rec Recurring
	err := tx.QueryRow(ctx, `
		UPDATE recurring_donations
		SET amount = COALESCE($2, amount),
		    subtype = COALESCE($3, subtype),
		    frequency = COALESCE($4, frequency),
		    is_active = COALESCE($5, is_active)
		WHERE id = $1::uuid AND (COALESCE($2, amount) > 0)
		RETURNING id::text, branch_id::text, subtype, amount::float8, frequency,
		          next_run_at, last_run_at, times_run, is_active, payment_method, description, created_at`,
		id, in.Amount, in.Subtype, in.Frequency, in.IsActive).
		Scan(&rec.ID, &rec.BranchID, &rec.Subtype, &rec.Amount, &rec.Frequency,
			&rec.NextRunAt, &rec.LastRunAt, &rec.TimesRun, &rec.IsActive,
			&rec.PaymentMethod, &rec.Description, &rec.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &rec, nil
}

const recurringSelect = `
	SELECT rd.id::text, rd.tenant_id::text, rd.branch_id::text,
	       rd.member_id::text, rd.benefactor_id::text, rd.category_id::text,
	       rd.account_id::text, rd.subtype, rd.amount::float8, rd.frequency, rd.next_run_at,
	       rd.payment_method, rd.description
	FROM recurring_donations rd
	WHERE rd.is_active AND rd.next_run_at <= now()`

type dueRecurring struct {
	ID        string
	TenantID  string
	BranchID  string
	MemberID  *string
	BenID     *string
	CatID     *string
	AcctID    *string
	Subtype   string
	Amount    float64
	Frequency string
	NextRun   time.Time
	Payment   string
	Desc      string
}

// RecurringWorker gera lancamentos automaticos para agendamentos vencidos,
// operando dentro do contexto RLS de cada filial (receita + recibo automatico).
// Quando o doador tem contato, o recibo tambem e enfileirado para envio
// (a entrega real e feita pelo worker de outbox).
type RecurringWorker struct {
	Store     *store.Store
	Finance   *Repo
	Documents *documents.Repo
	Interval  time.Duration
	Batch     int
}

func (w *RecurringWorker) Run(ctx context.Context) {
	if w.Interval <= 0 {
		w.Interval = time.Hour
	}
	if w.Batch <= 0 {
		w.Batch = 100
	}
	ticker := time.NewTicker(w.Interval)
	defer ticker.Stop()
	log.Printf("[recurring-worker] iniciado (intervalo=%s, lote=%d)", w.Interval, w.Batch)
	for {
		select {
		case <-ctx.Done():
			log.Println("[recurring-worker] encerrado")
			return
		case <-ticker.C:
			if err := w.ProcessDue(ctx); err != nil {
				log.Printf("[recurring-worker] erro: %v", err)
			}
		}
	}
}

func (w *RecurringWorker) ProcessDue(ctx context.Context) error {
	var due []dueRecurring
	err := w.Store.WithSystem(ctx, func(tx pgx.Tx) error {
		var err error
		due, err = w.listDue(ctx, tx, w.Batch)
		return err
	})
	if err != nil {
		return err
	}
	for _, d := range due {
		if err := w.process(ctx, d); err != nil {
			log.Printf("[recurring-worker] recurring %s: %v", d.ID, err)
		}
	}
	return nil
}

func (w *RecurringWorker) listDue(ctx context.Context, tx pgx.Tx, limit int) ([]dueRecurring, error) {
	rows, err := tx.Query(ctx, recurringSelect+" ORDER BY rd.next_run_at LIMIT $1", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []dueRecurring{}
	for rows.Next() {
		var d dueRecurring
		if err := rows.Scan(&d.ID, &d.TenantID, &d.BranchID, &d.MemberID, &d.BenID, &d.CatID,
			&d.AcctID, &d.Subtype, &d.Amount, &d.Frequency, &d.NextRun, &d.Payment, &d.Desc); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

func (w *RecurringWorker) process(ctx context.Context, d dueRecurring) error {
	bounds := store.Bounds{TenantID: d.TenantID, BranchID: d.BranchID, Role: "system"}
	return w.Store.WithTenant(ctx, bounds, func(tx pgx.Tx) error {
		desc := d.Desc
		if desc == "" {
			desc = "Doacao recorrente (" + d.Subtype + ")"
		}
		// A conta contabil e obrigatoria no lancamento. Recorrencias antigas
		// podem nao ter uma vinculada; resolve pelo subtipo (dizimo/oferta/doacao).
		catID := d.CatID
		if catID == nil || *catID == "" {
			cid, err := defaultIncomeCategoryID(ctx, tx, d.Subtype)
			if err != nil {
				return err
			}
			catID = &cid
		}
		in := CreateTxnInput{
			Type:          recurringTypeIncome,
			Amount:        d.Amount,
			CategoryID:    catID,
			AccountID:     d.AcctID,
			PaymentMethod: nullIfEmpty(&d.Payment),
			Description:   &desc,
			DonorMemberID: d.MemberID,
			BenefactorID:  d.BenID,
		}
		t, docID, _, _, err := w.Finance.Create(ctx, tx, d.TenantID, d.BranchID, in)
		if err != nil {
			return err
		}

		// Enfileira o envio do recibo ao doador, se houver contato cadastrado.
		if w.Documents != nil && docID != "" {
			if channel, to := donorContact(ctx, tx, d.MemberID, d.BenID); to != "" {
				if _, err := w.Documents.QueueDelivery(ctx, tx, d.TenantID, d.BranchID, docID, channel, to); err != nil {
					// Falha no enfileiramento nao deve desfazer o lancamento.
					log.Printf("[recurring-worker] recibo %s nao enfileirado: %v", t.ID, err)
				}
			}
		}

		_, err = tx.Exec(ctx, `
			UPDATE recurring_donations
			SET last_run_at = now(), times_run = times_run + 1,
			    next_run_at = $2
			WHERE id = $1::uuid`,
			d.ID, nextRunFor(d.NextRun, d.Frequency, time.Now()))
		return err
	})
}

// donorContact resolve o melhor canal de contato do doador (e-mail preferencial,
// senao WhatsApp). Retorna ("", "") quando nao ha contato cadastrado.
func donorContact(ctx context.Context, tx pgx.Tx, memberID, benefactorID *string) (string, string) {
	if memberID != nil && *memberID != "" {
		var email, whatsapp, phone *string
		err := tx.QueryRow(ctx, `
			SELECT email, whatsapp, phone FROM members WHERE id = $1::uuid`, *memberID).
			Scan(&email, &whatsapp, &phone)
		if err == nil {
			if email != nil && *email != "" {
				return "email", *email
			}
			if whatsapp != nil && *whatsapp != "" {
				return "whatsapp", *whatsapp
			}
			if phone != nil && *phone != "" {
				return "whatsapp", *phone
			}
			return "", ""
		}
	}
	if benefactorID != nil && *benefactorID != "" {
		var email, phone *string
		err := tx.QueryRow(ctx, `
			SELECT email, phone FROM benefactors WHERE id = $1::uuid`, *benefactorID).
			Scan(&email, &phone)
		if err == nil {
			if email != nil && *email != "" {
				return "email", *email
			}
			if phone != nil && *phone != "" {
				return "whatsapp", *phone
			}
		}
	}
	return "", ""
}

// defaultIncomeCategoryID resolve a conta de entrada para uma recorrencia sem
// categoria vinculada, pelo subtipo (dizimo=101, oferta=102, doacao=103); sem
// correspondencia, usa a primeira conta de entrada ativa.
func defaultIncomeCategoryID(ctx context.Context, tx pgx.Tx, subtype string) (string, error) {
	code := ""
	switch subtype {
	case "dizimo":
		code = "101"
	case "oferta":
		code = "102"
	case "doacao":
		code = "103"
	}
	var id string
	err := tx.QueryRow(ctx, `
		SELECT id::text FROM financial_categories
		WHERE type = 'income' AND is_active AND ($1 = '' OR code = $1)
		ORDER BY (code = $1) DESC, code
		LIMIT 1`, code).Scan(&id)
	return id, err
}
