package payments

import (
	"context"
	"log"
	"time"

	"github.com/jackc/pgx/v5"

	"chosenerp/internal/store"
)

// Worker reconcilia as cobrancas no gateway e aplica a suspensao automatica
// das igrejas inadimplentes. Roda periodicamente no main.
type Worker struct {
	Store            *store.Store
	Service          *Service
	Interval         time.Duration
	SuspendEnabled   bool
	SuspendAfterDays int
	// OnEvent e chamado quando uma igreja e suspensa/reativada (notificacao).
	OnEvent func(ctx context.Context, tenantID, tenantName, event string)
	Now     func() time.Time
}

func (w *Worker) now() time.Time {
	if w.Now != nil {
		return w.Now()
	}
	return time.Now()
}

// Run dispara o processamento em intervalos regulares.
func (w *Worker) Run(ctx context.Context) {
	if w.Interval <= 0 {
		w.Interval = 5 * time.Minute
	}
	ticker := time.NewTicker(w.Interval)
	defer ticker.Stop()
	// Uma passada inicial leve (sem bloquear o boot em caso de gateway fora).
	if err := w.ProcessOnce(ctx); err != nil {
		log.Printf("payments worker: %v", err)
	}
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := w.ProcessOnce(ctx); err != nil {
				log.Printf("payments worker: %v", err)
			}
		}
	}
}

// ProcessOnce sincroniza as assinaturas e aplica suspensao/reativacao.
func (w *Worker) ProcessOnce(ctx context.Context) error {
	if w.Service == nil || !w.Service.Enabled() {
		return nil
	}

	// 1) Sincroniza as igrejas com assinatura viva no gateway.
	var ids []string
	if err := w.Store.WithSystem(ctx, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `
			SELECT DISTINCT tenant_id::text FROM billing_subscriptions
			WHERE status='active' AND external_id IS NOT NULL`)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var id string
			if err := rows.Scan(&id); err != nil {
				return err
			}
			ids = append(ids, id)
		}
		return rows.Err()
	}); err != nil {
		return err
	}
	for _, id := range ids {
		if _, err := w.Service.SyncTenant(ctx, id); err != nil {
			log.Printf("payments sync tenant %s: %v", id, err)
		}
	}

	// 2) Suspende as igrejas com atraso acima da tolerancia.
	if w.SuspendEnabled {
		if err := w.suspendOverdue(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (w *Worker) suspendOverdue(ctx context.Context) error {
	deadline := w.now().AddDate(0, 0, -w.SuspendAfterDays)
	type suspended struct{ id, name string }
	var list []suspended
	err := w.Store.WithSystem(ctx, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `
			UPDATE tenants SET is_active=false, billing_status='suspended',
				suspended_at=now(), updated_at=now()
			WHERE billing_status='overdue' AND billing_exempt=false AND is_active=true
			  AND past_due_since IS NOT NULL AND past_due_since <= $1
			RETURNING id::text, name`, deadline)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var s suspended
			if err := rows.Scan(&s.id, &s.name); err != nil {
				return err
			}
			list = append(list, s)
		}
		return rows.Err()
	})
	if err != nil {
		return err
	}
	for _, s := range list {
		log.Printf("payments: igreja suspensa por inadimplencia: %s (%s)", s.name, s.id)
		if w.OnEvent != nil {
			w.OnEvent(ctx, s.id, s.name, "suspended")
		}
	}
	return nil
}
