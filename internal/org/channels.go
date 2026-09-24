package org

import (
	"context"

	"github.com/jackc/pgx/v5"
)

// BranchChannels reune a configuracao de comunicacao de uma filial: WhatsApp
// (instancia Evolution conectada por QR) e SMTP de e-mail proprio. A senha do
// SMTP nunca sai do banco - so o flag SMTPPasswordSet.
type BranchChannels struct {
	BranchID         string `json:"branch_id"`
	WhatsAppPhone    string `json:"whatsapp_phone"`
	WhatsAppInstance string `json:"whatsapp_instance"`
	WhatsAppStatus   string `json:"whatsapp_status"` // disconnected|connecting|connected
	SMTPHost         string `json:"smtp_host"`
	SMTPPort         int    `json:"smtp_port"`
	SMTPUser         string `json:"smtp_user"`
	SMTPPasswordSet  bool   `json:"smtp_password_set"`
	SMTPFrom         string `json:"smtp_from"`
	SMTPFromName     string `json:"smtp_from_name"`
	SMTPSecure       bool   `json:"smtp_secure"`
}

// BranchChannelsInput e o corpo de edicao (parcial). SMTPPassword nil mantem a
// senha atual; string vazia tambem mantem (para nao apagar por engano).
type BranchChannelsInput struct {
	WhatsAppPhone *string `json:"whatsapp_phone"`
	SMTPHost      *string `json:"smtp_host"`
	SMTPPort      *int    `json:"smtp_port"`
	SMTPUser      *string `json:"smtp_user"`
	SMTPPassword  *string `json:"smtp_password"`
	SMTPFrom      *string `json:"smtp_from"`
	SMTPFromName  *string `json:"smtp_from_name"`
	SMTPSecure    *bool   `json:"smtp_secure"`
}

const channelsCols = `b.id::text,
	COALESCE(b.whatsapp_phone, ''),
	COALESCE(b.whatsapp_instance, ''),
	b.whatsapp_status,
	COALESCE(b.smtp_host, ''),
	COALESCE(b.smtp_port, 0),
	COALESCE(b.smtp_user, ''),
	(b.smtp_password IS NOT NULL AND b.smtp_password <> ''),
	COALESCE(b.smtp_from, ''),
	COALESCE(b.smtp_from_name, ''),
	b.smtp_secure`

func scanChannels(row pgx.Row) (*BranchChannels, error) {
	var c BranchChannels
	err := row.Scan(&c.BranchID, &c.WhatsAppPhone, &c.WhatsAppInstance, &c.WhatsAppStatus,
		&c.SMTPHost, &c.SMTPPort, &c.SMTPUser, &c.SMTPPasswordSet,
		&c.SMTPFrom, &c.SMTPFromName, &c.SMTPSecure)
	return &c, err
}

// GetBranchChannels le a configuracao de canais de uma filial do tenant atual.
func (r *Repo) GetBranchChannels(ctx context.Context, tx pgx.Tx, id string) (*BranchChannels, error) {
	return scanChannels(tx.QueryRow(ctx,
		`SELECT `+channelsCols+` FROM branches b WHERE b.id = $1::uuid`, id))
}

// UpdateBranchChannels grava a configuracao (parcial). Campos nulos sao
// preservados; a senha so e trocada quando informada (nao vazia).
func (r *Repo) UpdateBranchChannels(ctx context.Context, tx pgx.Tx, id string, in BranchChannelsInput) (*BranchChannels, error) {
	var updatedID string
	err := tx.QueryRow(ctx, `
		UPDATE branches SET
			whatsapp_phone = COALESCE($2, whatsapp_phone),
			smtp_host      = COALESCE($3, smtp_host),
			smtp_port      = COALESCE($4, smtp_port),
			smtp_user      = COALESCE($5, smtp_user),
			smtp_password  = CASE WHEN $6::boolean THEN $7 ELSE smtp_password END,
			smtp_from      = COALESCE($8, smtp_from),
			smtp_from_name = COALESCE($9, smtp_from_name),
			smtp_secure    = COALESCE($10, smtp_secure),
			updated_at     = now()
		WHERE id = $1::uuid
		RETURNING id::text`,
		id, in.WhatsAppPhone, in.SMTPHost, in.SMTPPort, in.SMTPUser,
		in.SMTPPassword != nil && *in.SMTPPassword != "", strVal(in.SMTPPassword),
		in.SMTPFrom, in.SMTPFromName, in.SMTPSecure).
		Scan(&updatedID)
	if err != nil {
		return nil, err
	}
	return r.GetBranchChannels(ctx, tx, updatedID)
}

// SetBranchWhatsApp grava a instancia/estado da conexao WhatsApp da filial.
func (r *Repo) SetBranchWhatsApp(ctx context.Context, tx pgx.Tx, id, instance, status string) error {
	tag, err := tx.Exec(ctx, `
		UPDATE branches SET
			whatsapp_instance = $2,
			whatsapp_status = $3,
			updated_at = now()
		WHERE id = $1::uuid`, id, instance, status)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func strVal(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
