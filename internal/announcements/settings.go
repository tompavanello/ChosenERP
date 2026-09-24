package announcements

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
)

// Templates padrão das automações (podem ser editados pela igreja).
const (
	DefaultBirthdayTemplate = "Feliz aniversário, {primeiro_nome}! A {igreja} celebra a sua vida hoje. Que Deus o(a) abençoe ricamente!"
	DefaultRosterTemplate   = "Olá, {primeiro_nome}! Lembrete: você está escalado(a) em \"{titulo}\" no dia {data}. Conte com você!"
	DefaultVisitorTemplate  = "Olá, {primeiro_nome}! Foi uma alegria receber você na {igreja}. Esperamos ver você novamente em breve!"
)

// NotificationSettings é a configuração de automações de WhatsApp do tenant (#31).
type NotificationSettings struct {
	TenantID                 string    `json:"tenant_id"`
	BirthdaysEnabled         bool      `json:"birthdays_enabled"`
	BirthdayTemplate         string    `json:"birthday_template"`
	RosterRemindersEnabled   bool      `json:"roster_reminders_enabled"`
	RosterReminderHours      int       `json:"roster_reminder_hours"`
	RosterTemplate           string    `json:"roster_template"`
	VisitorWelcomeEnabled    bool      `json:"visitor_welcome_enabled"`
	VisitorWelcomeDelayHours int       `json:"visitor_welcome_delay_hours"`
	VisitorWelcomeTemplate   string    `json:"visitor_welcome_template"`
	UpdatedAt                time.Time `json:"updated_at"`
}

// TenantNotificationSettings acrescenta os dados da igreja usados no template.
type TenantNotificationSettings struct {
	NotificationSettings
	TenantName string
	Timezone   string
}

type NotificationSettingsInput struct {
	BirthdaysEnabled         *bool   `json:"birthdays_enabled"`
	BirthdayTemplate         *string `json:"birthday_template"`
	RosterRemindersEnabled   *bool   `json:"roster_reminders_enabled"`
	RosterReminderHours      *int    `json:"roster_reminder_hours"`
	RosterTemplate           *string `json:"roster_template"`
	VisitorWelcomeEnabled    *bool   `json:"visitor_welcome_enabled"`
	VisitorWelcomeDelayHours *int    `json:"visitor_welcome_delay_hours"`
	VisitorWelcomeTemplate   *string `json:"visitor_welcome_template"`
}

func defaultNotificationSettings(tenantID string) *NotificationSettings {
	return &NotificationSettings{
		TenantID:                 tenantID,
		BirthdaysEnabled:         false,
		BirthdayTemplate:         DefaultBirthdayTemplate,
		RosterRemindersEnabled:   false,
		RosterReminderHours:      24,
		RosterTemplate:           DefaultRosterTemplate,
		VisitorWelcomeEnabled:    false,
		VisitorWelcomeDelayHours: 24,
		VisitorWelcomeTemplate:   DefaultVisitorTemplate,
		UpdatedAt:                time.Time{},
	}
}

// GetNotificationSettings lê a configuração do tenant; sem linha, devolve os
// padrões (as automações nascem desligadas).
func (r *Repo) GetNotificationSettings(ctx context.Context, tx pgx.Tx, tenantID string) (*NotificationSettings, error) {
	s := defaultNotificationSettings(tenantID)
	err := tx.QueryRow(ctx, `
		SELECT tenant_id::text, birthdays_enabled, birthday_template,
		       roster_reminders_enabled, roster_reminder_hours, roster_template,
		       visitor_welcome_enabled, visitor_welcome_delay_hours, visitor_welcome_template,
		       updated_at
		FROM notification_settings`).
		Scan(&s.TenantID, &s.BirthdaysEnabled, &s.BirthdayTemplate,
			&s.RosterRemindersEnabled, &s.RosterReminderHours, &s.RosterTemplate,
			&s.VisitorWelcomeEnabled, &s.VisitorWelcomeDelayHours, &s.VisitorWelcomeTemplate,
			&s.UpdatedAt)
	if err == pgx.ErrNoRows {
		return s, nil
	}
	if err != nil {
		return nil, err
	}
	return s, nil
}

// UpsertNotificationSettings grava a configuração, aplicando só os campos
// enviados sobre o valor atual (PATCH parcial).
func (r *Repo) UpsertNotificationSettings(ctx context.Context, tx pgx.Tx, tenantID string, in NotificationSettingsInput) (*NotificationSettings, error) {
	cur, err := r.GetNotificationSettings(ctx, tx, tenantID)
	if err != nil {
		return nil, err
	}
	if in.BirthdaysEnabled != nil {
		cur.BirthdaysEnabled = *in.BirthdaysEnabled
	}
	if in.BirthdayTemplate != nil && *in.BirthdayTemplate != "" {
		cur.BirthdayTemplate = *in.BirthdayTemplate
	}
	if in.RosterRemindersEnabled != nil {
		cur.RosterRemindersEnabled = *in.RosterRemindersEnabled
	}
	if in.RosterReminderHours != nil && *in.RosterReminderHours > 0 {
		cur.RosterReminderHours = *in.RosterReminderHours
	}
	if in.RosterTemplate != nil && *in.RosterTemplate != "" {
		cur.RosterTemplate = *in.RosterTemplate
	}
	if in.VisitorWelcomeEnabled != nil {
		cur.VisitorWelcomeEnabled = *in.VisitorWelcomeEnabled
	}
	if in.VisitorWelcomeDelayHours != nil && *in.VisitorWelcomeDelayHours > 0 {
		cur.VisitorWelcomeDelayHours = *in.VisitorWelcomeDelayHours
	}
	if in.VisitorWelcomeTemplate != nil && *in.VisitorWelcomeTemplate != "" {
		cur.VisitorWelcomeTemplate = *in.VisitorWelcomeTemplate
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO notification_settings
			(tenant_id, birthdays_enabled, birthday_template, roster_reminders_enabled,
			 roster_reminder_hours, roster_template, visitor_welcome_enabled,
			 visitor_welcome_delay_hours, visitor_welcome_template, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, now())
		ON CONFLICT (tenant_id) DO UPDATE SET
			birthdays_enabled = EXCLUDED.birthdays_enabled,
			birthday_template = EXCLUDED.birthday_template,
			roster_reminders_enabled = EXCLUDED.roster_reminders_enabled,
			roster_reminder_hours = EXCLUDED.roster_reminder_hours,
			roster_template = EXCLUDED.roster_template,
			visitor_welcome_enabled = EXCLUDED.visitor_welcome_enabled,
			visitor_welcome_delay_hours = EXCLUDED.visitor_welcome_delay_hours,
			visitor_welcome_template = EXCLUDED.visitor_welcome_template,
			updated_at = now()`,
		tenantID, cur.BirthdaysEnabled, cur.BirthdayTemplate, cur.RosterRemindersEnabled,
		cur.RosterReminderHours, cur.RosterTemplate, cur.VisitorWelcomeEnabled,
		cur.VisitorWelcomeDelayHours, cur.VisitorWelcomeTemplate)
	if err != nil {
		return nil, err
	}
	return r.GetNotificationSettings(ctx, tx, tenantID)
}

// ListNotificationSettings devolve os tenants com ao menos uma automação ligada
// (usada pelo worker em escopo de sistema).
func (r *Repo) ListNotificationSettings(ctx context.Context, tx pgx.Tx) ([]TenantNotificationSettings, error) {
	rows, err := tx.Query(ctx, `
		SELECT s.tenant_id::text, s.birthdays_enabled, s.birthday_template,
		       s.roster_reminders_enabled, s.roster_reminder_hours, s.roster_template,
		       s.visitor_welcome_enabled, s.visitor_welcome_delay_hours, s.visitor_welcome_template,
		       s.updated_at, t.name, COALESCE(NULLIF(t.timezone,''), 'America/Sao_Paulo')
		FROM notification_settings s
		JOIN tenants t ON t.id = s.tenant_id
		WHERE s.birthdays_enabled OR s.roster_reminders_enabled OR s.visitor_welcome_enabled
		ORDER BY s.tenant_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []TenantNotificationSettings{}
	for rows.Next() {
		var s TenantNotificationSettings
		if err := rows.Scan(&s.TenantID, &s.BirthdaysEnabled, &s.BirthdayTemplate,
			&s.RosterRemindersEnabled, &s.RosterReminderHours, &s.RosterTemplate,
			&s.VisitorWelcomeEnabled, &s.VisitorWelcomeDelayHours, &s.VisitorWelcomeTemplate,
			&s.UpdatedAt, &s.TenantName, &s.Timezone); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// ---------------------------------------------------------------------------
// Resolução dos destinatários das automações
// ---------------------------------------------------------------------------

// BirthdayRecipient é um aniversariante do dia.
type BirthdayRecipient struct {
	MemberID  string
	BranchID  string
	Phone     string
	FirstName string
}

// BirthdayRecipients devolve os membros que fazem aniversário no mês/dia dados.
// Baixados, transferidos e falecidos não recebem. O `tenantID` é explícito
// porque o worker roda com role 'system' (a RLS libera todos os tenants).
func (r *Repo) BirthdayRecipients(ctx context.Context, tx pgx.Tx, tenantID string, month, day int) ([]BirthdayRecipient, error) {
	rows, err := tx.Query(ctx, `
		SELECT id::text, COALESCE(branch_id::text,''), whatsapp,
		       COALESCE(NULLIF(first_name,''), full_name)
		FROM members
		WHERE whatsapp IS NOT NULL AND whatsapp <> ''
		  AND tenant_id = $1
		  AND birth_date IS NOT NULL
		  AND EXTRACT(MONTH FROM birth_date) = $2
		  AND EXTRACT(DAY FROM birth_date) = $3
		  AND membership_status NOT IN ('dismissed','transferred','deceased')
		ORDER BY full_name`, tenantID, month, day)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []BirthdayRecipient{}
	for rows.Next() {
		var b BirthdayRecipient
		if err := rows.Scan(&b.MemberID, &b.BranchID, &b.Phone, &b.FirstName); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

// RosterReminderRecipient é um voluntário convidado cuja escala se aproxima.
type RosterReminderRecipient struct {
	AssignmentID string
	BranchID     string
	Phone        string
	FirstName    string
	RosterTitle  string
	StartsAt     time.Time
}

// RosterReminderRecipients devolve os escalados ainda "convidados" (não
// confirmaram) cuja escala começa dentro da janela [from, to].
func (r *Repo) RosterReminderRecipients(ctx context.Context, tx pgx.Tx, tenantID string, from, to time.Time) ([]RosterReminderRecipient, error) {
	rows, err := tx.Query(ctx, `
		SELECT ra.id::text, ra.branch_id::text, m.whatsapp,
		       COALESCE(NULLIF(m.first_name,''), m.full_name), r.title, r.starts_at
		FROM roster_assignments ra
		JOIN rosters r ON r.id = ra.roster_id
		JOIN members m ON m.id = ra.member_id
		WHERE ra.tenant_id = $1
		  AND ra.status = 'convidado'
		  AND r.status IN ('rascunho','publicada')
		  AND r.starts_at >= $2 AND r.starts_at <= $3
		  AND m.whatsapp IS NOT NULL AND m.whatsapp <> ''
		ORDER BY r.starts_at`, tenantID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []RosterReminderRecipient{}
	for rows.Next() {
		var rec RosterReminderRecipient
		if err := rows.Scan(&rec.AssignmentID, &rec.BranchID, &rec.Phone,
			&rec.FirstName, &rec.RosterTitle, &rec.StartsAt); err != nil {
			return nil, err
		}
		out = append(out, rec)
	}
	return out, rows.Err()
}

// VisitorWelcomeRecipient é um visitante a ser acolhido.
type VisitorWelcomeRecipient struct {
	VisitorID string
	BranchID  string
	Phone     string
	FirstName string
}

// VisitorWelcomeRecipients devolve visitantes criados entre `since` e `until`
// (normalmente [config.atualizacao, agora - atraso]) que ainda não são membros.
// Limitar pelo `since` evita disparar boas-vindas para o cadastro histórico no
// momento em que a automação é ligada.
func (r *Repo) VisitorWelcomeRecipients(ctx context.Context, tx pgx.Tx, tenantID string, since, until time.Time) ([]VisitorWelcomeRecipient, error) {
	rows, err := tx.Query(ctx, `
		SELECT id::text, COALESCE(branch_id::text,''), whatsapp,
		       COALESCE(NULLIF(first_name,''), 'visitante')
		FROM visitors
		WHERE whatsapp IS NOT NULL AND whatsapp <> ''
		  AND tenant_id = $1
		  AND converted_to_member_id IS NULL
		  AND created_at >= $2 AND created_at <= $3
		ORDER BY created_at`, tenantID, since, until)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []VisitorWelcomeRecipient{}
	for rows.Next() {
		var v VisitorWelcomeRecipient
		if err := rows.Scan(&v.VisitorID, &v.BranchID, &v.Phone, &v.FirstName); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
