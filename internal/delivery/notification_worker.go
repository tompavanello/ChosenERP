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

// NotificationWorker dispara as automacoes de WhatsApp (#31): aniversario do
// dia, lembrete de escala e boas-vindas a visitante. Ele NAO envia: apenas
// enfileira mensagens na outbox (announcement_deliveries); o AnnouncementWorker
// faz o envio real, reaproveitando o mesmo Dispatcher e o retry.
type NotificationWorker struct {
	Store         *store.Store
	Announcements *announcements.Repo
	Dispatcher    *Dispatcher
	Interval      time.Duration

	// Now permite fixar o relogio nos testes; nil = time.Now.
	Now func() time.Time
}

func (w *NotificationWorker) Run(ctx context.Context) {
	if w.Interval <= 0 {
		w.Interval = time.Hour
	}
	ticker := time.NewTicker(w.Interval)
	defer ticker.Stop()
	log.Printf("[notification-worker] iniciado (intervalo=%s)", w.Interval)

	if n, err := w.ProcessOnce(ctx); err != nil {
		log.Printf("[notification-worker] erro na varredura inicial: %v", err)
	} else if n > 0 {
		log.Printf("[notification-worker] %d mensagem(ns) de automacao enfileirada(s)", n)
	}

	for {
		select {
		case <-ctx.Done():
			log.Println("[notification-worker] encerrado")
			return
		case <-ticker.C:
			if n, err := w.ProcessOnce(ctx); err != nil {
				log.Printf("[notification-worker] erro ao processar automacoes: %v", err)
			} else if n > 0 {
				log.Printf("[notification-worker] %d mensagem(ns) de automacao enfileirada(s)", n)
			}
		}
	}
}

func (w *NotificationWorker) now() time.Time {
	if w.Now != nil {
		return w.Now()
	}
	return time.Now()
}

// ProcessOnce executa uma varredura e devolve quantas mensagens foram
// enfileiradas (sem contar as deduplicadas).
func (w *NotificationWorker) ProcessOnce(ctx context.Context) (int, error) {
	var settings []announcements.TenantNotificationSettings
	err := w.Store.WithSystem(ctx, func(tx pgx.Tx) error {
		var e error
		settings, e = w.Announcements.ListNotificationSettings(ctx, tx)
		return e
	})
	if err != nil {
		return 0, err
	}

	total := 0
	for _, s := range settings {
		n, err := w.processTenant(ctx, s)
		if err != nil {
			log.Printf("[notification-worker] tenant %s: %v", s.TenantID, err)
			continue
		}
		total += n
	}
	return total, nil
}

func (w *NotificationWorker) processTenant(ctx context.Context, s announcements.TenantNotificationSettings) (int, error) {
	provider := w.Dispatcher.ProviderFor(ChannelWhatsApp)
	now := w.now()
	loc := loadLocation(s.Timezone)
	local := now.In(loc)
	count := 0

	err := w.Store.WithSystem(ctx, func(tx pgx.Tx) error {
		if s.BirthdaysEnabled {
			recs, err := w.Announcements.BirthdayRecipients(ctx, tx, s.TenantID, int(local.Month()), local.Day())
			if err != nil {
				return err
			}
			for _, r := range recs {
				body := renderTemplate(s.BirthdayTemplate, map[string]string{
					"primeiro_nome": r.FirstName,
					"nome":          r.FirstName,
					"igreja":        s.TenantName,
				})
				ok, err := w.Announcements.CreateAutomatedDelivery(ctx, tx, announcements.AutomatedDelivery{
					TenantID: s.TenantID, BranchID: r.BranchID,
					Channel: ChannelWhatsApp, Provider: provider,
					Recipient: r.Phone, RecipientName: r.FirstName,
					Title: "Feliz aniversario", Body: body,
					Source:    announcements.SourceBirthday,
					DedupeKey: "birthday:" + r.MemberID + ":" + local.Format("2006-01-02"),
				})
				if err != nil {
					return err
				}
				if ok {
					count++
				}
			}
		}

		if s.RosterRemindersEnabled {
			to := now.Add(time.Duration(s.RosterReminderHours) * time.Hour)
			recs, err := w.Announcements.RosterReminderRecipients(ctx, tx, s.TenantID, now, to)
			if err != nil {
				return err
			}
			for _, r := range recs {
				body := renderTemplate(s.RosterTemplate, map[string]string{
					"primeiro_nome": r.FirstName,
					"nome":          r.FirstName,
					"titulo":        r.RosterTitle,
					"data":          r.StartsAt.In(loc).Format("02/01/2006 15:04"),
					"igreja":        s.TenantName,
				})
				ok, err := w.Announcements.CreateAutomatedDelivery(ctx, tx, announcements.AutomatedDelivery{
					TenantID: s.TenantID, BranchID: r.BranchID,
					Channel: ChannelWhatsApp, Provider: provider,
					Recipient: r.Phone, RecipientName: r.FirstName,
					Title: "Lembrete de escala", Body: body,
					Source:    announcements.SourceRoster,
					DedupeKey: "roster:" + r.AssignmentID,
				})
				if err != nil {
					return err
				}
				if ok {
					count++
				}
			}
		}

		if s.VisitorWelcomeEnabled {
			until := now.Add(-time.Duration(s.VisitorWelcomeDelayHours) * time.Hour)
			since := s.UpdatedAt
			if since.IsZero() {
				since = until.Add(-30 * 24 * time.Hour)
			}
			recs, err := w.Announcements.VisitorWelcomeRecipients(ctx, tx, s.TenantID, since, until)
			if err != nil {
				return err
			}
			for _, r := range recs {
				body := renderTemplate(s.VisitorWelcomeTemplate, map[string]string{
					"primeiro_nome": r.FirstName,
					"nome":          r.FirstName,
					"igreja":        s.TenantName,
				})
				ok, err := w.Announcements.CreateAutomatedDelivery(ctx, tx, announcements.AutomatedDelivery{
					TenantID: s.TenantID, BranchID: r.BranchID,
					Channel: ChannelWhatsApp, Provider: provider,
					Recipient: r.Phone, RecipientName: r.FirstName,
					Title: "Boas-vindas", Body: body,
					Source:    announcements.SourceVisitorWelcome,
					DedupeKey: "visitor:" + r.VisitorID,
				})
				if err != nil {
					return err
				}
				if ok {
					count++
				}
			}
		}
		return nil
	})
	return count, err
}

// renderTemplate troca {chave} pelo valor; chaves desconhecidas ficam como estao.
func renderTemplate(tpl string, vars map[string]string) string {
	out := tpl
	for k, v := range vars {
		out = strings.ReplaceAll(out, "{"+k+"}", v)
	}
	return out
}

func loadLocation(tz string) *time.Location {
	if tz != "" {
		if l, err := time.LoadLocation(tz); err == nil {
			return l
		}
	}
	return time.UTC
}
