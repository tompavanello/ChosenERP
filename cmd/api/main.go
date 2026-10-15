package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"chosenerp/internal/announcements"
	"chosenerp/internal/audit"
	"chosenerp/internal/auth"
	"chosenerp/internal/backup"
	"chosenerp/internal/benefactors"
	"chosenerp/internal/cargos"
	"chosenerp/internal/config"
	"chosenerp/internal/delivery"
	"chosenerp/internal/documents"
	"chosenerp/internal/events"
	"chosenerp/internal/families"
	"chosenerp/internal/finance"
	"chosenerp/internal/governance"
	"chosenerp/internal/groups"
	"chosenerp/internal/httpapi"
	"chosenerp/internal/kids"
	"chosenerp/internal/lgpd"
	"chosenerp/internal/materials"
	"chosenerp/internal/memberevents"
	"chosenerp/internal/members"
	"chosenerp/internal/ministries"
	"chosenerp/internal/org"
	"chosenerp/internal/payments"
	"chosenerp/internal/payments/asaas"
	"chosenerp/internal/prayer"
	"chosenerp/internal/programacao"
	"chosenerp/internal/push"
	"chosenerp/internal/requests"
	"chosenerp/internal/rosters"
	"chosenerp/internal/store"
	"chosenerp/internal/suppliers"
	"chosenerp/internal/users"
	"chosenerp/internal/visitors"
)

func main() {
	ctx := context.Background()
	cfg := config.Load()
	if err := cfg.Validate(); err != nil {
		log.Fatalf("config: %v", err)
	}
	if cfg.JWTSecret == config.DefaultJWTSecret {
		log.Println("AVISO: JWT_SECRET esta no valor default de desenvolvimento. Defina um segredo forte em producao.")
	}

	st, err := store.New(ctx, cfg.DatabaseURL, cfg.MigrateURL)
	if err != nil {
		log.Fatalf("store: %v", err)
	}
	defer st.Close()

	if err := st.Migrate(ctx); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	authSvc := auth.NewService(st, cfg.JWTSecret, cfg.JWTAccessExpiry, cfg.JWTRefreshExpiry)

	dispatcher := delivery.NewDispatcher(delivery.Config{
		SMTP: delivery.SMTPConfig{
			Host: cfg.SmtpHost, Port: cfg.SmtpPort, User: cfg.SmtpUser,
			Password: cfg.SmtpPassword, From: cfg.SmtpFrom, FromName: cfg.SmtpFromName,
		},
		WhatsApp: delivery.WhatsAppConfig{
			Token: cfg.WhatsAppToken, PhoneID: cfg.WhatsAppPhoneID, APIVersion: cfg.WhatsAppAPIVersion,
		},
		Evolution: delivery.EvolutionConfig{
			BaseURL: cfg.EvolutionAPIURL, APIKey: cfg.EvolutionAPIKey, Instance: cfg.EvolutionInstance,
		},
		WhatsAppProvider: cfg.WhatsAppProvider,
	})
	// Permite resolver canais (SMTP/WhatsApp) por filial no momento do envio.
	dispatcher.Store = st

	// Web Push (VAPID): desabilitado quando as chaves nao estao configuradas.
	pushRepo := &push.Repo{
		Config: push.Config{
			PublicKey:  cfg.VAPIDPublicKey,
			PrivateKey: cfg.VAPIDPrivateKey,
			Subject:    cfg.VAPIDSubject,
		},
		Client: &http.Client{Timeout: 15 * time.Second},
	}

	docRepo := &documents.Repo{}
	finRepo := &finance.Repo{}

	// Pagamentos / gateways (Asaas). O Factory evita o ciclo de import entre
	// payments e payments/asaas.
	paymentsSvc := payments.NewService(st, payments.Config{
		DefaultProvider:   cfg.PaymentsDefaultProvider,
		AsaasBaseURL:      cfg.AsaasBaseURL,
		AsaasAPIKey:       cfg.AsaasAPIKey,
		AsaasWebhookToken: cfg.AsaasWebhookToken,
		CredentialsKey:    cfg.PaymentsCredentialsKey,
		JWTSecret:         cfg.JWTSecret,
		SuspendEnabled:    cfg.PaymentsSuspendEnabled,
		SuspendAfterDays:  cfg.PaymentsSuspendAfterDays,
	})
	paymentsSvc.Factory = func(provider string, c payments.Credentials) (payments.Gateway, error) {
		switch provider {
		case "asaas", "":
			return asaas.New(c.BaseURL, c.APIKey), nil
		default:
			return nil, fmt.Errorf("provedor de pagamento nao suportado: %s", provider)
		}
	}

	router := httpapi.NewRouter(&httpapi.Config{
		JWTSecret:           cfg.JWTSecret,
		AppBaseURL:          cfg.AppBaseURL,
		UploadDir:           cfg.UploadDir,
		MemberPhotoMaxBytes: cfg.MemberPhotoMaxBytes,
		EvolutionAPIURL:     cfg.EvolutionAPIURL,
		EvolutionAPIKey:     cfg.EvolutionAPIKey,
		PublicBaseDomain:    cfg.PublicBaseDomain,
		AIBaseURL:           cfg.AIBaseURL,
		AIAPIKey:            cfg.AIAPIKey,
		AIModel:             cfg.AIModel,
		APIBibleKey:         cfg.APIBibleKey,
		APIBibleBaseURL:     cfg.APIBibleBaseURL,
		BackupDir:           cfg.BackupDir,
	}, st, authSvc,
		&members.Repo{}, finRepo, &audit.Repo{}, docRepo,
		&families.Repo{}, &visitors.Repo{}, &benefactors.Repo{}, &suppliers.Repo{},
		&ministries.Repo{}, &groups.Repo{}, &announcements.Repo{},
		&cargos.Repo{}, &programacao.Repo{}, &users.Repo{}, &events.Repo{}, &lgpd.Repo{}, &governance.Repo{}, &org.Repo{}, &rosters.Repo{}, &kids.Repo{}, &memberevents.Repo{}, &prayer.Repo{}, &materials.Repo{}, pushRepo, &requests.Repo{}, dispatcher, paymentsSvc)

	// Worker da outbox de envio (retry de pendentes/falhos)
	worker := &delivery.Worker{
		Store:      st,
		Documents:  docRepo,
		Dispatcher: dispatcher,
		BaseURL:    cfg.AppBaseURL,
		Interval:   cfg.DeliveryInterval,
	}
	go worker.Run(ctx)

	// Worker de comunicados (disparo assincrono de avisos por WhatsApp/e-mail)
	annWorker := &delivery.AnnouncementWorker{
		Store:         st,
		Announcements: &announcements.Repo{},
		Dispatcher:    dispatcher,
		Interval:      cfg.DeliveryInterval,
	}
	go annWorker.Run(ctx)

	// Worker das automacoes de WhatsApp (#31): enfileira aniversarios, lembretes
	// de escala e boas-vindas a visitantes; o worker de comunicados faz o envio.
	notifWorker := &delivery.NotificationWorker{
		Store:         st,
		Announcements: &announcements.Repo{},
		Dispatcher:    dispatcher,
		Push:          pushRepo,
		Interval:      cfg.NotificationInterval,
	}
	go notifWorker.Run(ctx)

	// Worker dos comunicados agendados (uma vez, diario ou relativo a evento).
	scheduleWorker := &delivery.ScheduleWorker{
		Store:         st,
		Announcements: &announcements.Repo{},
		Dispatcher:    dispatcher,
		Push:          pushRepo,
		Interval:      cfg.ScheduleInterval,
	}
	go scheduleWorker.Run(ctx)

	// Worker de doacoes recorrentes (gera lancamentos + recibos automaticos)
	recWorker := &finance.RecurringWorker{
		Store:     st,
		Finance:   finRepo,
		Documents: docRepo,
		Interval:  cfg.RecurringInterval,
	}
	go recWorker.Run(ctx)

	// Worker de cobranca: reconcilia assinaturas no gateway e suspende as
	// igrejas inadimplentes apos a tolerancia configurada.
	payWorker := &payments.Worker{
		Store:            st,
		Service:          paymentsSvc,
		Interval:         cfg.PaymentsPollInterval,
		SuspendEnabled:   cfg.PaymentsSuspendEnabled,
		SuspendAfterDays: cfg.PaymentsSuspendAfterDays,
		OnEvent: func(_ context.Context, tenantID, tenantName, event string) {
			log.Printf("payments: igreja %q (%s) -> %s", tenantName, tenantID, event)
		},
	}
	go payWorker.Run(ctx)

	// Worker de backup agendado por igreja (le backup_settings e aplica retencao).
	backupMgr := &backup.Manager{Store: st, Dir: cfg.BackupDir, UploadDir: cfg.UploadDir}
	go (&backup.Worker{Manager: backupMgr, Interval: cfg.BackupPollInterval}).Run(ctx)

	srv := &http.Server{
		Addr:    ":" + cfg.HTTPPort,
		Handler: router,
		// ReadTimeout cobre o corpo INTEIRO da requisicao. Com 15s, um upload de
		// foto em rede movel lenta era cortado no meio; 60s acomoda o limite de
		// MEMBER_PHOTO_MAX_KB sem afetar as rotas JSON.
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	go func() {
		log.Printf("Chosen ERP API ouvindo em :%s", cfg.HTTPPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("http: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	shutdownCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
	log.Println("servidor encerrado")
}
