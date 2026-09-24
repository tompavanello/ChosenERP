package delivery

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"mime/quotedprintable"
	"net"
	"net/http"
	"net/smtp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"chosenerp/internal/store"
)

// Channels suportados pela outbox de documentos.
const (
	ChannelEmail    = "email"
	ChannelWhatsApp = "whatsapp"
)

// Message é o conteúdo a ser entregue por um canal específico.
// TenantID/BranchID identificam o remetente quando a filial tem canais
// próprios (WhatsApp/SMTP); sem eles, usa-se a configuração global.
type Message struct {
	Channel    string
	Recipient  string
	Subject    string
	HTML       string
	Text       string
	Link       string
	TenantName string
	TenantID   string
	BranchID   string
}

type Sender interface {
	Channel() string
	Provider() string
	Send(ctx context.Context, m Message) error
}

type Dispatcher struct {
	senders   map[string]Sender
	providers map[string]string
	// Store (opcional) permite resolver canais POR FILIAL em branco. Quando
	// ausente ou sem configuração, usa-se o sender global.
	Store *store.Store
	// Credenciais do servidor Evolution (instâncias são por filial).
	EvolutionBaseURL string
	EvolutionAPIKey  string
}

// Config reúne as credenciais dos provedores de envio.
type Config struct {
	SMTP             SMTPConfig
	WhatsApp         WhatsAppConfig
	Evolution        EvolutionConfig
	WhatsAppProvider string
	DisableRealSend  bool
}

type SMTPConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	From     string
	FromName string
}

type WhatsAppConfig struct {
	Token      string
	PhoneID    string
	APIVersion string
	HTTP       *http.Client
}

type EvolutionConfig struct {
	BaseURL  string
	APIKey   string
	Instance string
	HTTP     *http.Client
}

// NewDispatcher monta os senders reais quando configurados; caso contrário cai
// no modo simulado (mantém o comportamento de dev sem quebrar o fluxo).
func NewDispatcher(cfg Config) *Dispatcher {
	d := &Dispatcher{senders: map[string]Sender{}, providers: map[string]string{}}
	d.senders[ChannelEmail] = newEmailSender(cfg)
	d.senders[ChannelWhatsApp] = newWhatsAppSender(cfg)
	d.providers[ChannelEmail] = "smtp"
	d.providers[ChannelWhatsApp] = cfg.WhatsAppProvider
	d.EvolutionBaseURL = cfg.Evolution.BaseURL
	d.EvolutionAPIKey = cfg.Evolution.APIKey
	return d
}

func newEmailSender(cfg Config) Sender {
	if cfg.DisabledFor(ChannelEmail) {
		return &simulatedSender{channel: ChannelEmail}
	}
	return &EmailSender{
		Host: cfg.SMTP.Host, Port: cfg.SMTP.Port,
		User: cfg.SMTP.User, Password: cfg.SMTP.Password,
		From: cfg.SMTP.From, FromName: cfg.SMTP.FromName,
	}
}

func newWhatsAppSender(cfg Config) Sender {
	if cfg.DisabledFor(ChannelWhatsApp) {
		return &simulatedSender{channel: ChannelWhatsApp, provider: "none"}
	}
	if cfg.WhatsAppProvider == "evolution" {
		httpc := cfg.Evolution.HTTP
		if httpc == nil {
			httpc = &http.Client{Timeout: 20 * time.Second}
		}
		return &EvolutionSender{
			BaseURL: cfg.Evolution.BaseURL, APIKey: cfg.Evolution.APIKey,
			Instance: cfg.Evolution.Instance, HTTP: httpc,
		}
	}
	httpc := cfg.WhatsApp.HTTP
	if httpc == nil {
		httpc = &http.Client{Timeout: 20 * time.Second}
	}
	return &WhatsAppSender{
		Token: cfg.WhatsApp.Token, PhoneID: cfg.WhatsApp.PhoneID,
		APIVersion: cfg.WhatsApp.APIVersion, HTTP: httpc,
	}
}

// DisabledFor indica se o canal não está configurado (deve ser simulado).
func (c Config) DisabledFor(channel string) bool {
	if c.DisableRealSend {
		return true
	}
	if channel == ChannelEmail {
		return c.SMTP.Host == "" || c.SMTP.From == ""
	}
	if channel == ChannelWhatsApp {
		if c.WhatsAppProvider == "evolution" {
			return c.Evolution.BaseURL == "" || c.Evolution.APIKey == "" || c.Evolution.Instance == ""
		}
		return c.WhatsApp.Token == "" || c.WhatsApp.PhoneID == ""
	}
	return true
}

// Send roteia a mensagem para o sender do canal. Se a filial informada tiver
// canais próprios configurados (SMTP/Evolution), usa-os; senão, cai no global.
func (d *Dispatcher) Send(ctx context.Context, m Message) error {
	if d.Store != nil && m.TenantID != "" && m.BranchID != "" {
		if s := d.branchSender(ctx, m); s != nil {
			return s.Send(ctx, m)
		}
	}
	s, ok := d.senders[m.Channel]
	if !ok {
		return fmt.Errorf("canal de envio desconhecido: %q", m.Channel)
	}
	return s.Send(ctx, m)
}

// branchSender devolve um sender específico da filial, ou nil para usar o global.
func (d *Dispatcher) branchSender(ctx context.Context, m Message) Sender {
	var host, user, pass, from, fromName, instance *string
	var port *int
	var secure bool
	err := d.Store.WithSystem(ctx, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `
			SELECT smtp_host, smtp_port, smtp_user, smtp_password, smtp_from,
			       smtp_from_name, smtp_secure, whatsapp_instance
			FROM branches
			WHERE id = $1::uuid AND tenant_id = $2::uuid`,
			m.BranchID, m.TenantID).
			Scan(&host, &port, &user, &pass, &from, &fromName, &secure, &instance)
	})
	if err != nil {
		return nil
	}

	switch m.Channel {
	case ChannelEmail:
		if host == nil || *host == "" || from == nil || *from == "" {
			return nil
		}
		p := 587
		if port != nil && *port > 0 {
			p = *port
		}
		fn := "Chosen ERP"
		if fromName != nil && *fromName != "" {
			fn = *fromName
		}
		return &EmailSender{
			Host: *host, Port: p, User: deref(user), Password: deref(pass),
			From: *from, FromName: fn,
		}
	case ChannelWhatsApp:
		// Se a filial tem instância Evolution e o servidor está configurado,
		// ela vence o provedor global (o WhatsApp da filial é o remetente).
		if instance == nil || *instance == "" || d.EvolutionBaseURL == "" || d.EvolutionAPIKey == "" {
			return nil
		}
		return &EvolutionSender{
			BaseURL: d.EvolutionBaseURL, APIKey: d.EvolutionAPIKey,
			Instance: *instance, HTTP: &http.Client{Timeout: 20 * time.Second},
		}
	}
	return nil
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// ProviderFor retorna o nome do provedor configurado para o canal (ex: "evolution", "meta", "smtp").
func (d *Dispatcher) ProviderFor(channel string) string {
	if p, ok := d.providers[channel]; ok {
		return p
	}
	return "unknown"
}

// ---------------------------------------------------------------------------
// Simulated (dev)
// ---------------------------------------------------------------------------

type simulatedSender struct{ channel, provider string }

func (s *simulatedSender) Channel() string  { return s.channel }
func (s *simulatedSender) Provider() string { return s.provider }
func (s *simulatedSender) Send(_ context.Context, m Message) error {
	log.Printf("[envio:simulado] canal=%s provedor=%s destino=%q (provedor não configurado)", m.Channel, s.provider, m.Recipient)
	return nil
}

// ---------------------------------------------------------------------------
// SMTP (e-mail)
// ---------------------------------------------------------------------------

type EmailSender struct {
	Host     string
	Port     int
	User     string
	Password string
	From     string
	FromName string
}

func (s *EmailSender) Channel() string  { return ChannelEmail }
func (s *EmailSender) Provider() string { return "smtp" }

func (s *EmailSender) Send(ctx context.Context, m Message) error {
	if m.Recipient == "" {
		return errors.New("destinatário de e-mail vazio")
	}
	var auth smtp.Auth
	if s.User != "" {
		auth = smtp.PlainAuth("", s.User, s.Password, s.Host)
	}
	from := s.From
	if s.FromName != "" {
		from = fmt.Sprintf("%s <%s>", s.FromName, s.From)
	}
	body, err := buildEmail(from, m)
	if err != nil {
		return err
	}
	tlsCfg := &tls.Config{ServerName: s.Host}
	return sendSMTP(ctx, s.Host, s.Port, auth, s.From, []string{m.Recipient}, body, tlsCfg)
}

func buildEmail(from string, m Message) ([]byte, error) {
	var buf bytes.Buffer
	writeHeader := func(k, v string) { fmt.Fprintf(&buf, "%s: %s\r\n", k, v) }
	writeHeader("From", from)
	writeHeader("To", m.Recipient)
	writeHeader("Subject", subjectUTF8(m.Subject))
	writeHeader("MIME-Version", "1.0")
	writeHeader("Date", time.Now().UTC().Format(time.RFC1123Z))
	boundary := "boundary-" + time.Now().UTC().Format("20060102150405.000000000")
	writeHeader("Content-Type", fmt.Sprintf(`multipart/alternative; boundary="%s"`, boundary))
	writeHeader("", "")
	fmt.Fprintf(&buf, "Este é um e-mail multipart/mixed.\r\n")

	fmt.Fprintf(&buf, "--%s\r\n", boundary)
	writeHeader("Content-Type", "text/plain; charset=utf-8")
	writeHeader("Content-Transfer-Encoding", "quoted-printable")
	writeHeader("", "")
	qp := quotedprintable.NewWriter(&buf)
	if m.Text != "" {
		_, _ = qp.Write([]byte(m.Text))
	}
	if m.Link != "" {
		_, _ = qp.Write([]byte("\r\n\r\nAcesse: " + m.Link))
	}
	_ = qp.Close()

	if m.HTML != "" {
		fmt.Fprintf(&buf, "\r\n--%s\r\n", boundary)
		writeHeader("Content-Type", "text/html; charset=utf-8")
		writeHeader("Content-Transfer-Encoding", "quoted-printable")
		writeHeader("", "")
		qp2 := quotedprintable.NewWriter(&buf)
		_, _ = qp2.Write([]byte(m.HTML))
		_ = qp2.Close()
	}
	fmt.Fprintf(&buf, "\r\n--%s--\r\n", boundary)
	return buf.Bytes(), nil
}

func subjectUTF8(s string) string {
	if s == "" {
		return s
	}
	return "=?UTF-8?B?" + base64Encode([]byte(s)) + "?="
}

func base64Encode(b []byte) string {
	const table = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"
	var out strings.Builder
	for i := 0; i < len(b); i += 3 {
		var chunk = [3]byte{}
		n := copy(chunk[:], b[i:])
		out.WriteByte(table[chunk[0]>>2])
		out.WriteByte(table[((chunk[0]&0x03)<<4)|(chunk[1]>>4)])
		if n > 1 {
			out.WriteByte(table[((chunk[1]&0x0f)<<2)|(chunk[2]>>6)])
		} else {
			out.WriteByte('=')
		}
		if n > 2 {
			out.WriteByte(table[chunk[2]&0x3f])
		} else {
			out.WriteByte('=')
		}
	}
	return out.String()
}

// sendSMTP entrega um e-mail via SMTP com suporte a STARTTLS (587) e TLS
// implícito (465). O contexto permite cancelamento da escrita.
func sendSMTP(ctx context.Context, host string, port int, auth smtp.Auth, from string, to []string, body []byte, tlsCfg *tls.Config) error {
	addr := net.JoinHostPort(host, fmt.Sprintf("%d", port))
	var c *smtp.Client
	var err error
	if port == 465 {
		conn, derr := tls.Dial("tcp", addr, tlsCfg)
		if derr != nil {
			return fmt.Errorf("smtp tls dial: %w", derr)
		}
		c, err = smtp.NewClient(conn, host)
	} else {
		conn, derr := net.Dial("tcp", addr)
		if derr != nil {
			return fmt.Errorf("smtp dial: %w", derr)
		}
		c, err = smtp.NewClient(conn, host)
		if err == nil {
			if ok, _ := c.Extension("STARTTLS"); ok {
				if terr := c.StartTLS(tlsCfg); terr != nil {
					_ = c.Close()
					return fmt.Errorf("smtp starttls: %w", terr)
				}
			}
		}
	}
	if err != nil {
		return fmt.Errorf("smtp client: %w", err)
	}
	defer func() { _ = c.Close() }()

	if auth != nil {
		if err := c.Auth(auth); err != nil {
			return fmt.Errorf("smtp auth: %w", err)
		}
	}
	if err := c.Mail(from); err != nil {
		return fmt.Errorf("smtp mail: %w", err)
	}
	for _, rcpt := range to {
		if rcpt == "" {
			continue
		}
		if err := c.Rcpt(rcpt); err != nil {
			return fmt.Errorf("smtp rcpt %q: %w", rcpt, err)
		}
	}
	w, err := c.Data()
	if err != nil {
		return fmt.Errorf("smtp data: %w", err)
	}
	done := make(chan error, 1)
	go func() {
		_, werr := w.Write(body)
		cerr := w.Close()
		if werr != nil {
			done <- werr
		} else {
			done <- cerr
		}
	}()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-done:
		if err != nil {
			return fmt.Errorf("smtp write: %w", err)
		}
	}
	return c.Quit()
}

// ---------------------------------------------------------------------------
// WhatsApp Business Cloud API
// ---------------------------------------------------------------------------

type WhatsAppSender struct {
	Token      string
	PhoneID    string
	APIVersion string
	HTTP       *http.Client
}

func (s *WhatsAppSender) Channel() string  { return ChannelWhatsApp }
func (s *WhatsAppSender) Provider() string { return "meta" }

func (s *WhatsAppSender) Send(ctx context.Context, m Message) error {
	if m.Recipient == "" {
		return errors.New("destinatário de WhatsApp vazio")
	}
	body := m.Text
	if m.Link != "" {
		body += "\n\n" + m.Link
	}
	payload := map[string]any{
		"messaging_product": "whatsapp",
		"to":                m.Recipient,
		"type":              "text",
		"text":              map[string]string{"body": body},
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	url := "https://graph.facebook.com/" + s.APIVersion + "/" + s.PhoneID + "/messages"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+s.Token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var e struct {
			Error struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&e)
		return fmt.Errorf("whatsapp status=%d: %s", resp.StatusCode, e.Error.Message)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Evolution API (WhatsApp Web alternativo — API aberta self-hosted)
// ---------------------------------------------------------------------------

type EvolutionSender struct {
	BaseURL  string
	APIKey   string
	Instance string
	HTTP     *http.Client
}

func (s *EvolutionSender) Channel() string  { return ChannelWhatsApp }
func (s *EvolutionSender) Provider() string { return "evolution" }

func (s *EvolutionSender) Send(ctx context.Context, m Message) error {
	if m.Recipient == "" {
		return errors.New("destinatário de WhatsApp vazio")
	}
	body := m.Text
	if m.Link != "" {
		body += "\n\n" + m.Link
	}
	payload := map[string]any{
		"number":      m.Recipient,
		"text":        body,
		"delay":       1234,
		"linkPreview": true,
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	url := strings.TrimRight(s.BaseURL, "/") + "/message/sendText/" + s.Instance
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("apikey", s.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var e struct {
			Error  string `json:"error"`
			Detail string `json:"detail"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&e)
		return fmt.Errorf("evolution api status=%d: %s", resp.StatusCode, e.Error)
	}
	return nil
}
