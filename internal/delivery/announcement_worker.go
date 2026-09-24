package delivery

import (
	"context"
	"log"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"chosenerp/internal/announcements"
	"chosenerp/internal/store"
)

type AnnouncementWorker struct {
	Store         *store.Store
	Announcements *announcements.Repo
	Dispatcher    *Dispatcher
	Interval      time.Duration
	Batch         int
	MaxAttempts   int
}

func (w *AnnouncementWorker) Run(ctx context.Context) {
	if w.Interval <= 0 {
		w.Interval = 30 * time.Second
	}
	if w.Batch <= 0 {
		w.Batch = 100
	}
	if w.MaxAttempts <= 0 {
		w.MaxAttempts = 5
	}
	ticker := time.NewTicker(w.Interval)
	defer ticker.Stop()
	log.Printf("[announcement-worker] iniciado (intervalo=%s, lote=%d)", w.Interval, w.Batch)
	for {
		select {
		case <-ctx.Done():
			log.Println("[announcement-worker] encerrado")
			return
		case <-ticker.C:
			if err := w.ProcessPending(ctx); err != nil {
				log.Printf("[announcement-worker] erro ao processar fila: %v", err)
			}
		}
	}
}

func (w *AnnouncementWorker) ProcessPending(ctx context.Context) error {
	var pending []announcements.PendingAnnouncementDelivery
	err := w.Store.WithSystem(ctx, func(tx pgx.Tx) error {
		var err error
		pending, err = w.Announcements.PendingDeliveries(ctx, tx, w.Batch, w.MaxAttempts)
		return err
	})
	if err != nil {
		return err
	}
	for _, p := range pending {
		if err := w.process(ctx, p); err != nil {
			log.Printf("[announcement-worker] delivery %s (anúncio %s → %s): %v",
				p.ID, p.AnnouncementID, p.Recipient, err)
		}
	}
	return nil
}

func (w *AnnouncementWorker) process(ctx context.Context, p announcements.PendingAnnouncementDelivery) error {
	bounds := store.Bounds{TenantID: p.TenantID, BranchID: p.BranchID, Role: "system"}
	msg := Message{
		Channel:    p.Channel,
		Recipient:  p.Recipient,
		Subject:    p.Title,
		Text:       p.Body,
		Link:       "",
		TenantName: "",
		TenantID:   p.TenantID,
		BranchID:   p.BranchID,
	}
	if p.Channel == ChannelWhatsApp && msg.Text == "" {
		msg.Text = p.Title
	}

	if err := w.Dispatcher.Send(ctx, msg); err != nil {
		return w.mark(ctx, bounds, p.ID, err.Error(), "")
	}
	return w.mark(ctx, bounds, p.ID, "", "")
}

func (w *AnnouncementWorker) mark(ctx context.Context, bounds store.Bounds, id, errMsg, providerMsgID string) error {
	return w.Store.WithTenant(ctx, bounds, func(tx pgx.Tx) error {
		if errMsg != "" {
			return w.Announcements.MarkFailed(ctx, tx, id, truncateErr(errMsg))
		}
		return w.Announcements.MarkSent(ctx, tx, id, providerMsgID)
	})
}

func truncateErr(s string) string {
	if len(s) > 500 {
		return s[:500]
	}
	return strings.TrimSpace(s)
}
