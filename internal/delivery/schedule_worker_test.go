package delivery

import (
	"testing"
	"time"

	"chosenerp/internal/announcements"
)

func ptrStr(s string) *string { return &s }

func TestScheduleDue_Once(t *testing.T) {
	at := time.Date(2026, 9, 23, 10, 0, 0, 0, time.UTC)
	s := announcements.ScheduledAnnouncement{
		Announcement: announcements.Announcement{ScheduleType: "once", ScheduleAt: &at},
	}
	// antes da hora
	if due, _ := scheduleDue(s, at.Add(-time.Minute)); due {
		t.Error("não deveria vencer antes do horário")
	}
	// na hora
	if due, period := scheduleDue(s, at); !due || period != "once" {
		t.Errorf("deveria vencer, due=%v period=%q", due, period)
	}
	// já rodou
	ran := at.Add(time.Second)
	s.LastRunAt = &ran
	if due, _ := scheduleDue(s, at.Add(time.Hour)); due {
		t.Error("não deveria repetir agendamento único já executado")
	}
}

func TestScheduleDue_Daily(t *testing.T) {
	loc, _ := time.LoadLocation("America/Sao_Paulo")
	s := announcements.ScheduledAnnouncement{
		Announcement: announcements.Announcement{ScheduleType: "daily", ScheduleTime: ptrStr("08:00")},
		Timezone:     "America/Sao_Paulo",
	}

	before := time.Date(2026, 9, 23, 7, 0, 0, 0, loc)
	if due, _ := scheduleDue(s, before); due {
		t.Error("não deveria vencer antes do horário diário")
	}

	after := time.Date(2026, 9, 23, 12, 0, 0, 0, loc)
	due, period := scheduleDue(s, after)
	if !due || period != "2026-09-23" {
		t.Errorf("deveria vencer com período do dia, due=%v period=%q", due, period)
	}

	// Já rodou hoje → não repete.
	s.LastRunAt = &after
	if due, _ := scheduleDue(s, after.Add(time.Hour)); due {
		t.Error("não deveria repetir no mesmo dia")
	}
}

func TestScheduleDue_Event(t *testing.T) {
	starts := time.Date(2026, 9, 23, 19, 0, 0, 0, time.UTC)
	s := announcements.ScheduledAnnouncement{
		Announcement: announcements.Announcement{
			ScheduleType:          "event",
			ScheduleEventID:       ptrStr("evt-1"),
			ScheduleOffsetMinutes: -30,
		},
		EventStartsAt: &starts,
	}

	if due, _ := scheduleDue(s, starts.Add(-time.Hour)); due {
		t.Error("não deveria vencer antes de 30min do evento")
	}
	// 18:35 = 5 min antes do disparo (18:30)
	if due, period := scheduleDue(s, starts.Add(-25*time.Minute)); !due || period != "evt:evt-1" {
		t.Errorf("deveria vencer 30min antes, due=%v period=%q", due, period)
	}
	// Depois da janela de tolerância.
	if due, _ := scheduleDue(s, starts.Add(eventGraceWindow+time.Hour)); due {
		t.Error("não deveria vencer fora da janela de tolerância")
	}
}

func TestParseHHMM(t *testing.T) {
	if hh, mm, err := parseHHMM("08:05"); err != nil || hh != 8 || mm != 5 {
		t.Errorf("parseHHMM(08:05) = %d:%d %v", hh, mm, err)
	}
	if _, _, err := parseHHMM("8h"); err == nil {
		t.Error("formato inválido deveria falhar")
	}
}
