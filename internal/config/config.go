package config

import (
	"errors"
	"os"
	"strconv"
	"strings"
	"time"
)

// DefaultJWTSecret e o segredo de desenvolvimento. O boot falha se JWT_SECRET
// estiver vazio e avisa quando permanecer neste valor.
const DefaultJWTSecret = "dev-insecure-secret-change-me"

// Validate confere configuracoes obrigatorias.
func (c Config) Validate() error {
	if strings.TrimSpace(c.JWTSecret) == "" {
		return errors.New("JWT_SECRET obrigatorio")
	}
	return nil
}

// Config agrega as variaveis de ambiente da API.
type Config struct {
	HTTPPort         string
	DatabaseURL      string
	MigrateURL       string
	JWTSecret        string
	JWTAccessExpiry  time.Duration
	JWTRefreshExpiry time.Duration

	// Notificacao de documentos (recibos/carteirinhas) - envio real
	AppBaseURL string
	UploadDir  string
	// MemberPhotoMaxBytes limita o upload de foto do membro. E separado do limite
	// de anexos do financeiro (50 MB) porque a foto e uma imagem pequena e vai
	// trafegar em rede movel.
	MemberPhotoMaxBytes int64
	SmtpHost            string
	SmtpPort            int
	SmtpUser            string
	SmtpPassword        string
	SmtpFrom            string
	SmtpFromName        string
	WhatsAppToken       string
	WhatsAppPhoneID     string
	WhatsAppAPIVersion  string
	WhatsAppProvider    string
	EvolutionAPIURL     string
	EvolutionAPIKey     string
	EvolutionInstance   string
	// PublicBaseDomain: dominio base do white-label (subdominio por igreja).
	PublicBaseDomain string
	DeliveryInterval    time.Duration
	RecurringInterval   time.Duration
	// NotificationInterval e o intervalo da varredura das automacoes de
	// WhatsApp (aniversarios, escalas, visitantes). O padrao e horario.
	NotificationInterval time.Duration
	// ScheduleInterval e o intervalo da varredura dos comunicados agendados.
	ScheduleInterval time.Duration
	// Web Push (VAPID): chaves do par gerado por `go run ./cmd/vapid-gen` e o
	// contato (sub do JWT). Vazio desabilita o envio de push.
	VAPIDPublicKey  string
	VAPIDPrivateKey string
	VAPIDSubject    string

	// ---- Pagamentos / gateways (Asaas e afins) ----
	// PaymentsDefaultProvider e o provider usado pela plataforma (Fluxo A: a
	// Chosen ERP cobrando as igrejas).
	PaymentsDefaultProvider string
	// AsaasAPIKey/AsaasBaseURL configuram a conta da plataforma no Asaas.
	// Base URL de sandbox: https://sandbox.asaas.com/api/v3
	// Base URL de producao: https://api.asaas.com/v3
	AsaasAPIKey       string
	AsaasBaseURL      string
	AsaasWebhookToken string
	// PaymentsCredentialsKey cifra as credenciais de gateway por igreja no banco
	// (AES-GCM). Vazio => deriva do JWT_SECRET (recomendado definir um proprio).
	PaymentsCredentialsKey string
	// PaymentsSuspendEnabled liga a suspensao automatica da igreja inadimplente
	// (tenants.is_active = false) apos PaymentsSuspendAfterDays de atraso.
	PaymentsSuspendEnabled   bool
	PaymentsSuspendAfterDays int
	// PaymentsPollInterval e o intervalo do worker de cobranca (reconciliacao).
	PaymentsPollInterval time.Duration

	// ---- Conteudo do membro (IA + Biblia) ----
	// IA OpenAI-compativel (devocional). Sem AI_API_KEY o recurso fica off.
	AIBaseURL string
	AIAPIKey  string
	AIModel   string
	// api.bible (Biblia licenciada). Sem API_BIBLE_KEY o recurso fica off.
	APIBibleKey     string
	APIBibleBaseURL string

	// ---- Backup por igreja ----
	// BackupDir e o diretorio onde os pacotes .zip sao persistidos.
	BackupDir string
	// BackupPollInterval e a cadencia com que o worker verifica se e hora de
	// rodar o backup agendado (a periodicidade real vem de backup_settings).
	BackupPollInterval time.Duration
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getenvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

func getenvBool(key string, fallback bool) bool {
	if v := os.Getenv(key); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return fallback
}

func Load() Config {
	return Config{
		HTTPPort:         getenv("API_HTTP_PORT", "38080"),
		DatabaseURL:      getenv("DATABASE_URL", "postgres://chosenerp_app:chosenapp@127.0.0.1:35432/chosenerp?sslmode=disable"),
		MigrateURL:       getenv("MIGRATE_DATABASE_URL", "postgres://postgres:sinc@127.0.0.1:35432/chosenerp?sslmode=disable"),
		JWTSecret:        getenv("JWT_SECRET", "dev-insecure-secret-change-me"),
		JWTAccessExpiry:  time.Duration(getenvInt("JWT_EXPIRATION_MINUTES", 15)) * time.Minute,
		JWTRefreshExpiry: time.Duration(getenvInt("REFRESH_EXPIRATION_HOURS", 168)) * time.Hour,

		UploadDir:            getenv("UPLOAD_DIR", "/tmp/chosenerp/uploads"),
		AppBaseURL:           getenv("APP_BASE_URL", "http://localhost:38080"),
		MemberPhotoMaxBytes:  int64(getenvInt("MEMBER_PHOTO_MAX_KB", 5<<10)) << 10,
		SmtpHost:             getenv("SMTP_HOST", ""),
		SmtpPort:             getenvInt("SMTP_PORT", 587),
		SmtpUser:             getenv("SMTP_USER", ""),
		SmtpPassword:         getenv("SMTP_PASSWORD", ""),
		SmtpFrom:             getenv("SMTP_FROM", ""),
		SmtpFromName:         getenv("SMTP_FROM_NAME", "Chosen ERP"),
		WhatsAppToken:        getenv("WHATSAPP_TOKEN", ""),
		WhatsAppPhoneID:      getenv("WHATSAPP_PHONE_ID", ""),
		WhatsAppAPIVersion:   getenv("WHATSAPP_API_VERSION", "v19.0"),
		WhatsAppProvider:     getenv("WHATSAPP_PROVIDER", "evolution"),
		EvolutionAPIURL:      getenv("EVOLUTION_API_URL", ""),
		EvolutionAPIKey:      getenv("EVOLUTION_API_KEY", ""),
		EvolutionInstance:    getenv("EVOLUTION_INSTANCE", ""),
		PublicBaseDomain:     getenv("PUBLIC_BASE_DOMAIN", "erpchosen.com.br"),
		DeliveryInterval:     time.Duration(getenvInt("DELIVERY_POLL_SECONDS", 30)) * time.Second,
		RecurringInterval:    time.Duration(getenvInt("RECURRING_POLL_SECONDS", 3600)) * time.Second,
		NotificationInterval: time.Duration(getenvInt("NOTIFICATION_POLL_SECONDS", 3600)) * time.Second,
		ScheduleInterval:     time.Duration(getenvInt("SCHEDULE_POLL_SECONDS", 60)) * time.Second,
		VAPIDPublicKey:       getenv("VAPID_PUBLIC_KEY", ""),
		VAPIDPrivateKey:      getenv("VAPID_PRIVATE_KEY", ""),
		VAPIDSubject:         getenv("VAPID_SUBJECT", "mailto:contato@erpchosen.com.br"),

		PaymentsDefaultProvider:  getenv("PAYMENTS_DEFAULT_PROVIDER", "asaas"),
		AsaasAPIKey:              getenv("ASAAS_API_KEY", ""),
		AsaasBaseURL:             getenv("ASAAS_BASE_URL", "https://sandbox.asaas.com/api/v3"),
		AsaasWebhookToken:        getenv("ASAAS_WEBHOOK_TOKEN", ""),
		PaymentsCredentialsKey:   getenv("PAYMENTS_CREDENTIALS_KEY", ""),
		PaymentsSuspendEnabled:   getenvBool("PAYMENTS_SUSPEND_ENABLED", true),
		PaymentsSuspendAfterDays: getenvInt("PAYMENTS_SUSPEND_AFTER_DAYS", 10),
		PaymentsPollInterval:     time.Duration(getenvInt("PAYMENTS_POLL_SECONDS", 300)) * time.Second,

		AIBaseURL:       getenv("AI_BASE_URL", ""),
		AIAPIKey:        getenv("AI_API_KEY", ""),
		AIModel:         getenv("AI_MODEL", "gpt-4o-mini"),
		APIBibleKey:     getenv("API_BIBLE_KEY", ""),
		APIBibleBaseURL: getenv("API_BIBLE_BASE_URL", ""),

		BackupDir:          getenv("BACKUP_DIR", "/tmp/chosenerp/backups"),
		BackupPollInterval: time.Duration(getenvInt("BACKUP_POLL_SECONDS", 3600)) * time.Second,
	}
}
