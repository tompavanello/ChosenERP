package backup

import (
	"context"
	"log"
	"time"

	"github.com/jackc/pgx/v5"
)

// Worker executa a rotina agendada de backup: a cada ciclo le a configuracao
// global (backup_settings) e, quando habilitada e dentro do intervalo, gera um
// backup de cada igreja ativa e aplica a retencao.
type Worker struct {
	Manager  *Manager
	Interval time.Duration

	lastRun time.Time
}

// Run bloqueia ate o contexto ser cancelado.
func (w *Worker) Run(ctx context.Context) {
	if w.Interval <= 0 {
		w.Interval = time.Hour
	}
	ticker := time.NewTicker(w.Interval)
	defer ticker.Stop()
	log.Printf("[backup-worker] iniciado (intervalo=%s)", w.Interval)
	for {
		select {
		case <-ctx.Done():
			log.Println("[backup-worker] encerrado")
			return
		case <-ticker.C:
			w.tick(ctx)
		}
	}
}

func (w *Worker) tick(ctx context.Context) {
	s, err := w.Manager.GetSettings(ctx)
	if err != nil {
		log.Printf("[backup-worker] nao foi possivel ler a configuracao: %v", err)
		return
	}
	if !s.Enabled {
		return
	}
	interval := time.Duration(s.IntervalHours) * time.Hour
	if interval <= 0 {
		interval = 24 * time.Hour
	}
	if !w.lastRun.IsZero() && time.Since(w.lastRun) < interval {
		return
	}
	w.lastRun = time.Now()

	tenants, err := w.activeTenants(ctx)
	if err != nil {
		log.Printf("[backup-worker] nao foi possivel listar as igrejas: %v", err)
		return
	}
	for _, id := range tenants {
		if _, err := w.Manager.Create(ctx, id, "scheduled", ""); err != nil {
			log.Printf("[backup-worker] falha no backup da igreja %s: %v", id, err)
		}
	}
	if removed, err := w.Manager.CleanupExpired(ctx, s.RetentionDays); err != nil {
		log.Printf("[backup-worker] falha na retencao: %v", err)
	} else if removed > 0 {
		log.Printf("[backup-worker] %d backup(s) expirado(s) removido(s)", removed)
	}
}

func (w *Worker) activeTenants(ctx context.Context) ([]string, error) {
	var ids []string
	err := w.Manager.Store.WithSystem(ctx, func(tx pgx.Tx) error {
		rows, e := tx.Query(ctx, `SELECT id::text FROM tenants WHERE is_active = true ORDER BY created_at`)
		if e != nil {
			return e
		}
		defer rows.Close()
		for rows.Next() {
			var id string
			if e := rows.Scan(&id); e != nil {
				return e
			}
			ids = append(ids, id)
		}
		return rows.Err()
	})
	return ids, err
}
