package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"chosenerp/internal/auth"
	"chosenerp/internal/benefactors"
	"chosenerp/internal/config"
	"chosenerp/internal/documents"
	"chosenerp/internal/families"
	"chosenerp/internal/finance"
	"chosenerp/internal/httpapi"
	"chosenerp/internal/members"
	"chosenerp/internal/store"
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
	router := httpapi.NewRouter(&httpapi.Config{JWTSecret: cfg.JWTSecret}, st, authSvc,
		&members.Repo{}, &finance.Repo{}, &documents.Repo{},
		&families.Repo{}, &visitors.Repo{}, &benefactors.Repo{})

	srv := &http.Server{
		Addr:         ":" + cfg.HTTPPort,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
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
