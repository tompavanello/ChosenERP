package httpapi

import (
	"net/http"

	"chosenerp/internal/auth"
	"chosenerp/internal/benefactors"
	"chosenerp/internal/documents"
	"chosenerp/internal/families"
	"chosenerp/internal/finance"
	"chosenerp/internal/members"
	"chosenerp/internal/store"
	"chosenerp/internal/visitors"
)

// App agrega as dependências dos handlers HTTP.
type App struct {
	Config      *Config
	Store       *store.Store
	Auth        *auth.Service
	Members     *members.Repo
	Finance     *finance.Repo
	Documents   *documents.Repo
	Families    *families.Repo
	Visitors    *visitors.Repo
	Benefactors *benefactors.Repo
}

// NewRouter monta o gateway HTTP e suas rotas.
func NewRouter(cfg *Config, st *store.Store, authSvc *auth.Service, membersRepo *members.Repo, finRepo *finance.Repo, docRepo *documents.Repo, famRepo *families.Repo, visitRepo *visitors.Repo, benefRepo *benefactors.Repo) http.Handler {
	app := &App{
		Config:      cfg,
		Store:       st,
		Auth:        authSvc,
		Members:     membersRepo,
		Finance:     finRepo,
		Documents:   docRepo,
		Families:    famRepo,
		Visitors:    visitRepo,
		Benefactors: benefRepo,
	}
	mux := http.NewServeMux()

	// Público
	mux.HandleFunc("GET /healthz", app.handleHealth)
	mux.HandleFunc("GET /metrics", app.handleMetrics)
	mux.HandleFunc("POST /api/v1/auth/login", app.handleLogin)
	mux.HandleFunc("POST /api/v1/auth/refresh", app.handleRefresh)

	// Autenticado
	authed := app.Authenticator
	mux.Handle("GET /api/v1/me", authed(http.HandlerFunc(app.handleMe)))
	mux.Handle("GET /api/v1/members", authed(http.HandlerFunc(app.handleListMembers)))
	mux.Handle("POST /api/v1/members", authed(http.HandlerFunc(app.handleCreateMember)))
	mux.Handle("GET /api/v1/members/{id}", authed(http.HandlerFunc(app.handleGetMember)))
	mux.Handle("PATCH /api/v1/members/{id}", authed(http.HandlerFunc(app.handleUpdateMember)))
	mux.Handle("GET /api/v1/members/{id}/tree", authed(http.HandlerFunc(app.handleMemberTree)))
	mux.Handle("POST /api/v1/members/{id}/relationships", authed(http.HandlerFunc(app.handleAddRelationship)))
	mux.Handle("POST /api/v1/members/{id}/card", authed(http.HandlerFunc(app.handleIssueCard)))

	// Famílias
	mux.Handle("GET /api/v1/families", authed(http.HandlerFunc(app.handleListFamilies)))
	mux.Handle("POST /api/v1/families", authed(http.HandlerFunc(app.handleCreateFamily)))
	mux.Handle("GET /api/v1/families/{id}/members", authed(http.HandlerFunc(app.handleFamilyMembers)))
	mux.Handle("POST /api/v1/families/{id}/members", authed(http.HandlerFunc(app.handleAddFamilyMember)))

	// Visitantes
	mux.Handle("GET /api/v1/visitors", authed(http.HandlerFunc(app.handleListVisitors)))
	mux.Handle("POST /api/v1/visitors", authed(http.HandlerFunc(app.handleCreateVisitor)))
	mux.Handle("PATCH /api/v1/visitors/{id}/stage", authed(http.HandlerFunc(app.handleUpdateVisitorStage)))

	// Benfeitores
	mux.Handle("GET /api/v1/benefactors", authed(http.HandlerFunc(app.handleListBenefactors)))
	mux.Handle("POST /api/v1/benefactors", authed(http.HandlerFunc(app.handleCreateBenefactor)))

	// Financeiro
	mux.Handle("GET /api/v1/finance/categories", authed(http.HandlerFunc(app.handleListCategories)))
	mux.Handle("POST /api/v1/finance/categories", authed(http.HandlerFunc(app.handleCreateCategory)))
	mux.Handle("GET /api/v1/finance/transactions", authed(http.HandlerFunc(app.handleListTxn)))
	mux.Handle("POST /api/v1/finance/transactions", authed(http.HandlerFunc(app.handleCreateTxn)))
	mux.Handle("GET /api/v1/finance/balance", authed(http.HandlerFunc(app.handleBalance)))

	// Documentos digitais (leitura por token do QR)
	mux.Handle("GET /api/v1/documents/by-token/{token}", authed(http.HandlerFunc(app.handleGetDocumentByToken)))

	// Recibos: renderização e envio (prefixo próprio evita ambiguidade de rotas)
	mux.Handle("GET /api/v1/receipts/{id}", authed(http.HandlerFunc(app.handleRenderReceipt)))
	mux.Handle("POST /api/v1/receipts/{id}/send", authed(http.HandlerFunc(app.handleSendDocument)))
	mux.Handle("GET /api/v1/receipts/{id}/deliveries", authed(http.HandlerFunc(app.handleListDeliveries)))

	// Relatórios
	mux.Handle("GET /api/v1/reports/balance", authed(http.HandlerFunc(app.handleMonthlyBalance)))
	mux.Handle("GET /api/v1/reports/dre", authed(http.HandlerFunc(app.handleDRE)))

	return withCORS(mux)
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
