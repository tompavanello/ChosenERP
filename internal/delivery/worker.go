package delivery

import (
	"context"
	"log"
	"time"

	"github.com/jackc/pgx/v5"

	"chosenerp/internal/documents"
	"chosenerp/internal/store"
)

// Worker consome a outbox (document_deliveries) e desempenha o envio real
// de documentos pendentes/falhos (retry), operando DENTRO do contexto RLS
// de cada filial/tenant para preservar o isolamento multi-tenant.
type Worker struct {
	Store       *store.Store
	Documents   *documents.Repo
	Dispatcher  *Dispatcher
	BaseURL     string
	Interval    time.Duration
	Batch       int
	MaxAttempts int
}

// Run bloqueia processando a fila até o contexto ser cancelado.
func (w *Worker) Run(ctx context.Context) {
	if w.Interval <= 0 {
		w.Interval = 30 * time.Second
	}
	if w.Batch <= 0 {
		w.Batch = 50
	}
	if w.MaxAttempts <= 0 {
		w.MaxAttempts = 5
	}
	ticker := time.NewTicker(w.Interval)
	defer ticker.Stop()
	log.Printf("[delivery-worker] iniciado (intervalo=%s, lote=%d)", w.Interval, w.Batch)
	for {
		select {
		case <-ctx.Done():
			log.Println("[delivery-worker] encerrado")
			return
		case <-ticker.C:
			if err := w.ProcessPending(ctx); err != nil {
				log.Printf("[delivery-worker] erro ao processar fila: %v", err)
			}
		}
	}
}

// ProcessPending varre a outbox (visão "Sede") e tenta entregar cada item
// dentro do contexto RLS da sua própria filial.
func (w *Worker) ProcessPending(ctx context.Context) error {
	var pending []documents.PendingDelivery
	err := w.Store.WithSystem(ctx, func(tx pgx.Tx) error {
		var err error
		pending, err = w.Documents.ListPending(ctx, tx, w.Batch, w.MaxAttempts)
		return err
	})
	if err != nil {
		return err
	}
	for _, p := range pending {
		if err := w.process(ctx, p); err != nil {
			log.Printf("[delivery-worker] delivery %s (%s): %v", p.ID, p.Channel, err)
		}
	}
	return nil
}

func (w *Worker) process(ctx context.Context, p documents.PendingDelivery) error {
	bounds := store.Bounds{TenantID: p.TenantID, BranchID: p.BranchID, Role: "system"}
	msg, err := w.buildMessage(ctx, bounds, p)
	if err != nil {
		return err
	}
	if err := w.Dispatcher.Send(ctx, msg); err != nil {
		return w.mark(ctx, bounds, p.ID, err)
	}
	return w.mark(ctx, bounds, p.ID, nil)
}

// buildMessage carrega o documento e renderiza o corpo dentro do escopo RLS.
func (w *Worker) buildMessage(ctx context.Context, bounds store.Bounds, p documents.PendingDelivery) (Message, error) {
	var msg Message
	err := w.Store.WithTenant(ctx, bounds, func(tx pgx.Tx) error {
		doc, err := w.Documents.GetByID(ctx, tx, p.DocumentID)
		if err != nil {
			return err
		}
		tenant, _ := w.Documents.TenantName(ctx, tx)
		html, err := documents.RenderReceiptHTML(doc, tenant)
		if err != nil {
			return err
		}
		link := PublicLink(w.BaseURL, doc.Kind, doc.QRToken)
		msg = Message{
			Channel: p.Channel, Recipient: p.Recipient,
			Subject: doc.Title, HTML: html,
			Text: "Código de validação: " + doc.QRToken,
			Link: link, TenantName: tenant,
			TenantID: p.TenantID, BranchID: p.BranchID,
		}
		return nil
	})
	return msg, err
}

func (w *Worker) mark(ctx context.Context, bounds store.Bounds, id string, sendErr error) error {
	return w.Store.WithTenant(ctx, bounds, func(tx pgx.Tx) error {
		if sendErr != nil {
			return w.Documents.MarkFailed(ctx, tx, id, sendErr.Error())
		}
		return w.Documents.MarkSent(ctx, tx, id)
	})
}
