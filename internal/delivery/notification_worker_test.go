package delivery

import (
	"testing"
	"time"
)

func TestRenderTemplate(t *testing.T) {
	got := renderTemplate(
		"Olá, {primeiro_nome}! Feliz aniversário na {igreja}!",
		map[string]string{"primeiro_nome": "Ana", "igreja": "Igreja Central"},
	)
	want := "Olá, Ana! Feliz aniversário na Igreja Central!"
	if got != want {
		t.Errorf("renderTemplate = %q, quer %q", got, want)
	}
	// Chave desconhecida permanece intacta.
	if got := renderTemplate("{nao_existe}", map[string]string{"x": "y"}); got != "{nao_existe}" {
		t.Errorf("chave desconhecida deveria permanecer, veio %q", got)
	}
}

func TestLoadLocation(t *testing.T) {
	if loc := loadLocation("America/Sao_Paulo"); loc == nil || loc.String() != "America/Sao_Paulo" {
		t.Errorf("loadLocation válido falhou: %v", loc)
	}
	if loc := loadLocation("TZ/Inexistente"); loc != time.UTC {
		t.Errorf("timezone inválido deve cair em UTC, veio %v", loc)
	}
	if loc := loadLocation(""); loc != time.UTC {
		t.Errorf("timezone vazio deve cair em UTC, veio %v", loc)
	}
}

func TestNotificationWorkerNowDefault(t *testing.T) {
	w := &NotificationWorker{}
	if w.now().IsZero() {
		t.Error("now() sem override não deveria ser zero")
	}
	fixed := time.Date(2026, 9, 23, 10, 0, 0, 0, time.UTC)
	w.Now = func() time.Time { return fixed }
	if !w.now().Equal(fixed) {
		t.Error("now() deveria respeitar o override")
	}
}
