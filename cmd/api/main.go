package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"chosenerp/internal/announcements"
	"chosenerp/internal/audit"
	"chosenerp/internal/auth"
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
	"chosenerp/internal/members"
	"chosenerp/internal/ministries"
	"chosenerp/internal/org"
	"chosenerp/internal/rosters"
	"chosenerp/internal/store"
	"chosenerp/internal/suppliers"
	"chosenerp/internal/users"
	"chosenerp/internal/visitors"
)

func main() {
	ctx := context.Background()
	cfg := config.Load()

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

	docRepo := &documents.Repo{}
	finRepo := &finance.Repo{}
	router := httpapi.NewRouter(&httpapi.Config{
		JWTSecret:           cfg.JWTSecret,
		AppBaseURL:          cfg.AppBaseURL,
		UploadDir:           cfg.UploadDir,
		MemberPhotoMaxBytes: cfg.MemberPhotoMaxBytes,
		EvolutionAPIURL:     cfg.EvolutionAPIURL,
		EvolutionAPIKey:     cfg.EvolutionAPIKey,
	}, st, authSvc,
		&members.Repo{}, finRepo, &audit.Repo{}, docRepo,
		&families.Repo{}, &visitors.Repo{}, &benefactors.Repo{}, &suppliers.Repo{},
		&ministries.Repo{}, &groups.Repo{}, &announcements.Repo{},
		&cargos.Repo{}, &users.Repo{}, &events.Repo{}, &lgpd.Repo{}, &governance.Repo{}, &org.Repo{}, &rosters.Repo{}, &kids.Repo{}, dispatcher)

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
		Interval:      cfg.NotificationInterval,
	}
	go notifWorker.Run(ctx)

	// Worker dos comunicados agendados (uma vez, diario ou relativo a evento).
	scheduleWorker := &delivery.ScheduleWorker{
		Store:         st,
		Announcements: &announcements.Repo{},
		Dispatcher:    dispatcher,
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
