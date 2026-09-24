package config

import (
	"os"
	"strconv"
	"time"
)

// Config agrega as variáveis de ambiente da API.
type Config struct {
	HTTPPort         string
	DatabaseURL      string
	MigrateURL       string
	JWTSecret        string
	JWTAccessExpiry  time.Duration
	JWTRefreshExpiry time.Duration

	// Notificação de documentos (recibos/carteirinhas) — envio real
	AppBaseURL string
	UploadDir  string
	// MemberPhotoMaxBytes limita o upload de foto do membro. É separado do limite
	// de anexos do financeiro (50 MB) porque a foto é uma imagem pequena e vai
	// trafegar em rede móvel.
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
	DeliveryInterval    time.Duration
	RecurringInterval   time.Duration
	// NotificationInterval é o intervalo da varredura das automações de
	// WhatsApp (aniversários, escalas, visitantes). O padrão é horário.
	NotificationInterval time.Duration
	// ScheduleInterval é o intervalo da varredura dos comunicados agendados.
	ScheduleInterval time.Duration
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
		WhatsAppProvider:     getenv("WHATSAPP_PROVIDER", "meta"),
		EvolutionAPIURL:      getenv("EVOLUTION_API_URL", ""),
		EvolutionAPIKey:      getenv("EVOLUTION_API_KEY", ""),
		EvolutionInstance:    getenv("EVOLUTION_INSTANCE", ""),
		DeliveryInterval:     time.Duration(getenvInt("DELIVERY_POLL_SECONDS", 30)) * time.Second,
		RecurringInterval:    time.Duration(getenvInt("RECURRING_POLL_SECONDS", 3600)) * time.Second,
		NotificationInterval: time.Duration(getenvInt("NOTIFICATION_POLL_SECONDS", 3600)) * time.Second,
		ScheduleInterval:     time.Duration(getenvInt("SCHEDULE_POLL_SECONDS", 60)) * time.Second,
	}
}
