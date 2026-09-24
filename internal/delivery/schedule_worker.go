package delivery

import (
	"context"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"chosenerp/internal/announcements"
	"chosenerp/internal/store"
)

// ScheduleWorker dispara os comunicados agendados (#41): uma vez, diario ou
// relativo a um evento (N minutos antes/depois). So enfileira na outbox; o
// AnnouncementWorker envia de fato.
type ScheduleWorker struct {
	Store         *store.Store
	Announcements *announcements.Repo
	Dispatcher    *Dispatcher
	Interval      time.Duration

	// Now permite fixar o relogio nos testes; nil = time.Now.
	Now func() time.Time
}

// eventGraceWindow e por quanto tempo depois do horario marcado um agendamento
// por evento ainda dispara (tolera worker parado por algumas horas).
const eventGraceWindow = 6 * time.Hour

func (w *ScheduleWorker) Run(ctx context.Context) {
	if w.Interval <= 0 {
		w.Interval = time.Minute
	}
	ticker := time.NewTicker(w.Interval)
	defer ticker.Stop()
	log.Printf("[schedule-worker] iniciado (intervalo=%s)", w.Interval)

	w.tick(ctx)
	for {
		select {
		case <-ctx.Done():
			log.Println("[schedule-worker] encerrado")
			return
		case <-ticker.C:
			w.tick(ctx)
		}
	}
}

func (w *ScheduleWorker) tick(ctx context.Context) {
	if n, err := w.ProcessOnce(ctx); err != nil {
		log.Printf("[schedule-worker] erro ao processar agendamentos: %v", err)
	} else if n > 0 {
		log.Printf("[schedule-worker] %d comunicado(s) agendado(s) enfileirado(s)", n)
	}
}

func (w *ScheduleWorker) now() time.Time {
	if w.Now != nil {
		return w.Now()
	}
	return time.Now()
}

// ProcessOnce varre os agendamentos e enfileira os que venceram.
func (w *ScheduleWorker) ProcessOnce(ctx context.Context) (int, error) {
	var scheduled []announcements.ScheduledAnnouncement
	err := w.Store.WithSystem(ctx, func(tx pgx.Tx) error {
		var e error
		scheduled, e = w.Announcements.ListScheduled(ctx, tx)
		return e
	})
	if err != nil {
		return 0, err
	}

	now := w.now()
	total := 0
	for i := range scheduled {
		due, period := scheduleDue(scheduled[i], now)
		if !due {
			continue
		}
		n, err := w.enqueue(ctx, &scheduled[i], period)
		if err != nil {
			log.Printf("[schedule-worker] comunicado %s: %v", scheduled[i].ID, err)
			continue
		}
		total += n
	}
	return total, nil
}

// enqueue resolve os destinatarios e cria as entregas do disparo. Sempre marca
// o comunicado como executado para nao repetir a cada tick.
func (w *ScheduleWorker) enqueue(ctx context.Context, s *announcements.ScheduledAnnouncement, period string) (int, error) {
	channel := s.Channel
	if channel != ChannelEmail && channel != ChannelWhatsApp {
		channel = ChannelWhatsApp
	}
	provider := w.Dispatcher.ProviderFor(channel)
	count := 0

	err := w.Store.WithSystem(ctx, func(tx pgx.Tx) error {
		in := announcements.SendInput{Audience: s.Audience, AudienceFilter: s.AudienceFilter}
		recipients, err := w.Announcements.ResolveRecipients(ctx, tx, s.TenantID, in)
		if err != nil {
			return err
		}
		ann := s.Announcement
		for _, r := range recipients {
			key := "schedule:" + s.ID + ":" + period + ":" + r.Phone
			ok, err := w.Announcements.CreateScheduledDelivery(ctx, tx, &ann, provider, r.Phone, r.FullName, key)
			if err != nil {
				return err
			}
			if ok {
				count++
			}
		}
		if err := w.Announcements.CreateRun(ctx, tx, &ann, period, count); err != nil {
			return err
		}
		return w.Announcements.MarkRun(ctx, tx, s.ID)
	})
	return count, err
}

// scheduleDue diz se o agendamento venceu e qual a chave de periodo (dia/evento)
// usada na deduplicacao. Vencido = chegou a hora, nao rodou ainda e (no caso de
// evento) ainda esta dentro da janela de tolerancia.
func scheduleDue(s announcements.ScheduledAnnouncement, now time.Time) (bool, string) {
	loc := loadLocation(s.Timezone)
	switch s.ScheduleType {
	case "once":
		if s.ScheduleAt == nil || now.Before(*s.ScheduleAt) {
			return false, ""
		}
		if s.LastRunAt != nil && !s.LastRunAt.Before(*s.ScheduleAt) {
			return false, ""
		}
		return true, "once"

	case "daily":
		if s.ScheduleTime == nil {
			return false, ""
		}
		hh, mm, err := parseHHMM(*s.ScheduleTime)
		if err != nil {
			return false, ""
		}
		local := now.In(loc)
		fire := time.Date(local.Year(), local.Month(), local.Day(), hh, mm, 0, 0, loc)
		if local.Before(fire) {
			return false, ""
		}
		if s.LastRunAt != nil && !s.LastRunAt.Before(fire) {
			return false, ""
		}
		return true, fire.Format("2006-01-02")

	case "event":
		if s.EventStartsAt == nil || s.ScheduleEventID == nil {
			return false, ""
		}
		fire := s.EventStartsAt.Add(time.Duration(s.ScheduleOffsetMinutes) * time.Minute)
		if now.Before(fire) || now.After(fire.Add(eventGraceWindow)) {
			return false, ""
		}
		if s.LastRunAt != nil && !s.LastRunAt.Before(fire) {
			return false, ""
		}
		return true, "evt:" + *s.ScheduleEventID
	}
	return false, ""
}

func parseHHMM(v string) (int, int, error) {
	parts := strings.SplitN(v, ":", 2)
	if len(parts) != 2 {
		return 0, 0, strconv.ErrSyntax
	}
	hh, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, err
	}
	mm, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, err
	}
	return hh, mm, nil
}
