package httpapi

import (
	"net/http"

	"chosenerp/internal/announcements"
	"chosenerp/internal/audit"
	"chosenerp/internal/auth"
	"chosenerp/internal/benefactors"
	"chosenerp/internal/cargos"
	"chosenerp/internal/delivery"
	"chosenerp/internal/documents"
	"chosenerp/internal/events"
	"chosenerp/internal/families"
	"chosenerp/internal/finance"
	"chosenerp/internal/governance"
	"chosenerp/internal/groups"
	"chosenerp/internal/kids"
	"chosenerp/internal/lgpd"
	"chosenerp/internal/materials"
	"chosenerp/internal/memberevents"
	"chosenerp/internal/members"
	"chosenerp/internal/ministries"
	"chosenerp/internal/org"
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

// App agrega as dependencias dos handlers HTTP.
type App struct {
	Config        *Config
	Store         *store.Store
	Auth          *auth.Service
	Members       *members.Repo
	Finance       *finance.Repo
	Audits        *audit.Repo
	Documents     *documents.Repo
	Families      *families.Repo
	Visitors      *visitors.Repo
	Benefactors   *benefactors.Repo
	Suppliers     *suppliers.Repo
	Ministries    *ministries.Repo
	Groups        *groups.Repo
	Announcements *announcements.Repo
	Cargos        *cargos.Repo
	Programacao   *programacao.Repo
	Users         *users.Repo
	Events        *events.Repo
	LGPD          *lgpd.Repo
	Governance    *governance.Repo
	Org           *org.Repo
	Rosters       *rosters.Repo
	Kids          *kids.Repo
	MemberEvents  *memberevents.Repo
	Prayers       *prayer.Repo
	Materials     *materials.Repo
	Push          *push.Repo
	Requests      *requests.Repo
	Dispatch      *delivery.Dispatcher
	// Limitador protege as rotas /api/v1/public/* (sem auth).
	Limitador *limitadorPublico
}

// NewRouter monta o gateway HTTP e suas rotas.
func NewRouter(cfg *Config, st *store.Store, authSvc *auth.Service, membersRepo *members.Repo, finRepo *finance.Repo, auditRepo *audit.Repo, docRepo *documents.Repo, famRepo *families.Repo, visitRepo *visitors.Repo, benefRepo *benefactors.Repo, suppliersRepo *suppliers.Repo, ministRepo *ministries.Repo, groupsRepo *groups.Repo, annRepo *announcements.Repo, cargosRepo *cargos.Repo, programacaoRepo *programacao.Repo, usersRepo *users.Repo, eventsRepo *events.Repo, lgpdRepo *lgpd.Repo, govRepo *governance.Repo, orgRepo *org.Repo, rostersRepo *rosters.Repo, kidsRepo *kids.Repo, memberEventsRepo *memberevents.Repo, prayerRepo *prayer.Repo, materialsRepo *materials.Repo, pushRepo *push.Repo, requestsRepo *requests.Repo, dispatcher *delivery.Dispatcher) http.Handler {
	app := &App{
		Config:        cfg,
		Store:         st,
		Auth:          authSvc,
		Members:       membersRepo,
		Finance:       finRepo,
		Audits:        auditRepo,
		Documents:     docRepo,
		Families:      famRepo,
		Visitors:      visitRepo,
		Benefactors:   benefRepo,
		Suppliers:     suppliersRepo,
		Ministries:    ministRepo,
		Groups:        groupsRepo,
		Announcements: annRepo,
		Cargos:        cargosRepo,
		Programacao:   programacaoRepo,
		Users:         usersRepo,
		Events:        eventsRepo,
		LGPD:          lgpdRepo,
		Governance:    govRepo,
		Org:           orgRepo,
		Rosters:       rostersRepo,
		Kids:          kidsRepo,
		MemberEvents:  memberEventsRepo,
		Prayers:       prayerRepo,
		Materials:     materialsRepo,
		Push:          pushRepo,
		Requests:      requestsRepo,
		Dispatch:      dispatcher,
		Limitador:     newLimitadorPublico(),
	}
	mux := http.NewServeMux()

	// Publico
	mux.HandleFunc("GET /healthz", app.handleHealth)
	mux.HandleFunc("GET /metrics", app.handleMetrics)
	mux.HandleFunc("POST /api/v1/auth/login", app.handleLogin)
	mux.HandleFunc("POST /api/v1/auth/refresh", app.handleRefresh)
	mux.HandleFunc("POST /api/v1/auth/select-tenant", app.handleSelectTenant)
	// Branding da igreja por slug (tela de login do subdominio).
	mux.Handle("GET /api/v1/public/tenant/{slug}", app.publico(app.handlePublicTenant))
	// Formulario de contato do site institucional (grava marketing_leads).
	mux.Handle("POST /api/v1/public/leads", app.publico(app.handleCreateLead))

	// Autenticado + entitlement do plano (modulos fora do plano => 403).
	authed := func(h http.Handler) http.Handler {
		return app.Authenticator(app.featureHandler(h))
	}
	mux.Handle("GET /api/v1/me", authed(http.HandlerFunc(app.handleMe)))
	mux.Handle("GET /api/v1/me/tenants", authed(http.HandlerFunc(app.handleMeTenants)))
	mux.Handle("PATCH /api/v1/me", authed(http.HandlerFunc(app.handleUpdateProfile)))
	mux.Handle("POST /api/v1/me/password", authed(http.HandlerFunc(app.handleChangePassword)))

	// App do membro (escopo self): o membro e resolvido pelo vinculo da
	// identidade (memberships.member_id), nunca por id do cliente.
	mux.Handle("GET /api/v1/me/member", authed(http.HandlerFunc(app.handleMeMember)))
	mux.Handle("PATCH /api/v1/me/member", authed(http.HandlerFunc(app.handleUpdateMeMember)))
	mux.Handle("GET /api/v1/me/family", authed(http.HandlerFunc(app.handleMeFamily)))
	mux.Handle("GET /api/v1/me/events", authed(http.HandlerFunc(app.handleMeEvents)))
	mux.Handle("GET /api/v1/me/announcements", authed(http.HandlerFunc(app.handleMeAnnouncements)))
	mux.Handle("GET /api/v1/me/birthdays", authed(http.HandlerFunc(app.handleMeBirthdays)))
	mux.Handle("GET /api/v1/me/ministries", authed(http.HandlerFunc(app.handleMeMinistries)))
	mux.Handle("GET /api/v1/me/contributions", authed(http.HandlerFunc(app.handleMeContributions)))
	mux.Handle("GET /api/v1/me/groups", authed(http.HandlerFunc(app.handleMeGroups)))
	mux.Handle("GET /api/v1/me/led-groups", authed(http.HandlerFunc(app.handleMeLedGroups)))
	mux.Handle("GET /api/v1/me/materials", authed(http.HandlerFunc(app.handleListMyMaterials)))
	mux.Handle("GET /api/v1/me/materials/{id}/file", authed(http.HandlerFunc(app.handleDownloadMaterial)))
	mux.Handle("GET /api/v1/me/rosters", authed(http.HandlerFunc(app.handleMeRosters)))
	mux.Handle("PATCH /api/v1/me/rosters/assignments/{assignmentId}", authed(http.HandlerFunc(app.handleMeRespondRoster)))
	mux.Handle("GET /api/v1/me/frequency", authed(http.HandlerFunc(app.handleMeFrequency)))
	mux.Handle("GET /api/v1/me/push/public-key", authed(http.HandlerFunc(app.handlePushPublicKey)))
	mux.Handle("POST /api/v1/me/push/subscribe", authed(http.HandlerFunc(app.handleMePushSubscribe)))
	mux.Handle("DELETE /api/v1/me/push/subscribe", authed(http.HandlerFunc(app.handleMePushUnsubscribe)))
	mux.Handle("GET /api/v1/me/requests", authed(http.HandlerFunc(app.handleListMyRequests)))
	mux.Handle("POST /api/v1/me/requests", authed(http.HandlerFunc(app.handleCreateMyRequest)))
	mux.Handle("GET /api/v1/requests", authed(app.perm("members.read", app.handleListRequests)))
	mux.Handle("PATCH /api/v1/requests/{id}", authed(app.perm("members.write", app.handleUpdateRequest)))
	mux.Handle("GET /api/v1/me/prayer-requests", authed(http.HandlerFunc(app.handleListMyPrayers)))
	mux.Handle("POST /api/v1/me/prayer-requests", authed(http.HandlerFunc(app.handleCreatePrayer)))
	mux.Handle("GET /api/v1/me/prayer-wall", authed(http.HandlerFunc(app.handlePrayerWall)))
	mux.Handle("POST /api/v1/me/prayer-requests/{id}/react", authed(http.HandlerFunc(app.handleReactPrayer)))

	// Moderacao pastoral dos pedidos de oracao.
	mux.Handle("GET /api/v1/prayer-requests", authed(app.perm("prayer.read", app.handleListPrayersModeration)))
	mux.Handle("PATCH /api/v1/prayer-requests/{id}", authed(app.perm("prayer.moderate", app.handleUpdatePrayer)))
	mux.Handle("POST /api/v1/auth/switch-tenant", authed(http.HandlerFunc(app.handleSwitchTenant)))
	mux.Handle("GET /api/v1/members", authed(http.HandlerFunc(app.handleListMembers)))
	mux.Handle("POST /api/v1/members", authed(http.HandlerFunc(app.handleCreateMember)))
	mux.Handle("GET /api/v1/members/{id}", authed(http.HandlerFunc(app.handleGetMember)))
	mux.Handle("PATCH /api/v1/members/{id}", authed(http.HandlerFunc(app.handleUpdateMember)))
	mux.Handle("DELETE /api/v1/members/{id}", authed(http.HandlerFunc(app.handleDeleteMember)))
	mux.Handle("POST /api/v1/members/{id}/transfer", authed(app.perm("members.write", app.handleTransferMember)))
	mux.Handle("GET /api/v1/members/{id}/documents", authed(http.HandlerFunc(app.handleListMemberDocuments)))
	mux.Handle("POST /api/v1/members/{id}/documents", authed(app.perm("members.write", app.handleIssueMemberDocument)))
	mux.Handle("GET /api/v1/members/{id}/tree", authed(http.HandlerFunc(app.handleMemberTree)))
	mux.Handle("POST /api/v1/members/{id}/relationships", authed(http.HandlerFunc(app.handleAddRelationship)))
	mux.Handle("POST /api/v1/members/{id}/card", authed(http.HandlerFunc(app.handleIssueCard)))
	mux.Handle("GET /api/v1/members/{id}/card", authed(http.HandlerFunc(app.handleGetCard)))
	mux.Handle("POST /api/v1/members/{id}/photo", authed(http.HandlerFunc(app.handleUploadMemberPhoto)))
	mux.Handle("DELETE /api/v1/members/{id}/photo", authed(http.HandlerFunc(app.handleDeleteMemberPhoto)))
	mux.Handle("GET /api/v1/members/{id}/families", authed(http.HandlerFunc(app.handleMemberFamilies)))

	// Acesso do membro ao app (Sede): criar/redefinir senha provisoria, editar
	// identificador e ativar/desativar.
	mux.Handle("GET /api/v1/members/{id}/access", authed(http.HandlerFunc(app.handleGetMemberAccess)))
	mux.Handle("POST /api/v1/members/{id}/access", authed(http.HandlerFunc(app.handleCreateMemberAccess)))
	mux.Handle("PATCH /api/v1/members/{id}/access", authed(http.HandlerFunc(app.handleUpdateMemberAccess)))
	mux.Handle("POST /api/v1/members/{id}/access/password", authed(http.HandlerFunc(app.handleResetMemberAccessPassword)))

	// Historico eclesiastico do membro (requisito 1.8)
	mux.Handle("GET /api/v1/members/{id}/history", authed(http.HandlerFunc(app.handleListMemberHistory)))
	mux.Handle("POST /api/v1/members/{id}/history", authed(http.HandlerFunc(app.handleAddMemberHistory)))

	// Catalogo configuravel de eventos da vida eclesiastica (000065)
	mux.Handle("GET /api/v1/member-event-kinds", authed(http.HandlerFunc(app.handleListMemberEventKinds)))
	mux.Handle("POST /api/v1/member-event-kinds", authed(http.HandlerFunc(app.handleCreateMemberEventKind)))
	mux.Handle("PATCH /api/v1/member-event-kinds/{id}", authed(http.HandlerFunc(app.handleUpdateMemberEventKind)))
	mux.Handle("DELETE /api/v1/member-event-kinds/{id}", authed(http.HandlerFunc(app.handleDeleteMemberEventKind)))

	// Cargos (funcoes/ministerios) e mandatos do membro
	mux.Handle("GET /api/v1/cargos", authed(http.HandlerFunc(app.handleListCargos)))
	mux.Handle("POST /api/v1/cargos", authed(http.HandlerFunc(app.handleCreateCargo)))
	mux.Handle("PATCH /api/v1/cargos/{id}", authed(http.HandlerFunc(app.handleUpdateCargo)))
	mux.Handle("DELETE /api/v1/cargos/{id}", authed(http.HandlerFunc(app.handleDeleteCargo)))
	mux.Handle("GET /api/v1/members/{id}/cargos", authed(http.HandlerFunc(app.handleListMemberCargos)))
	mux.Handle("POST /api/v1/members/{id}/cargos", authed(http.HandlerFunc(app.handleAssignCargo)))
	mux.Handle("PATCH /api/v1/members/{id}/cargos/{linkId}", authed(http.HandlerFunc(app.handleUpdateMemberCargo)))
	mux.Handle("DELETE /api/v1/members/{id}/cargos/{linkId}", authed(http.HandlerFunc(app.handleUnassignCargo)))

	// Familias
	mux.Handle("GET /api/v1/families", authed(http.HandlerFunc(app.handleListFamilies)))
	mux.Handle("POST /api/v1/families", authed(http.HandlerFunc(app.handleCreateFamily)))
	mux.Handle("GET /api/v1/families/{id}", authed(http.HandlerFunc(app.handleGetFamily)))
	mux.Handle("PATCH /api/v1/families/{id}", authed(http.HandlerFunc(app.handleUpdateFamily)))
	mux.Handle("DELETE /api/v1/families/{id}", authed(http.HandlerFunc(app.handleDeleteFamily)))
	mux.Handle("GET /api/v1/families/{id}/members", authed(http.HandlerFunc(app.handleFamilyMembers)))
	mux.Handle("POST /api/v1/families/{id}/members", authed(http.HandlerFunc(app.handleAddFamilyMember)))
	mux.Handle("DELETE /api/v1/families/{id}/members/{memberId}", authed(http.HandlerFunc(app.handleUnlinkFamilyMember)))

	// Visitantes
	mux.Handle("GET /api/v1/visitors", authed(http.HandlerFunc(app.handleListVisitors)))
	mux.Handle("POST /api/v1/visitors", authed(http.HandlerFunc(app.handleCreateVisitor)))
	mux.Handle("PATCH /api/v1/visitors/{id}/stage", authed(http.HandlerFunc(app.handleUpdateVisitorStage)))
	mux.Handle("POST /api/v1/visitors/{id}/convert", authed(http.HandlerFunc(app.handleConvertVisitor)))
	mux.Handle("POST /api/v1/visitors/{id}/revert", authed(http.HandlerFunc(app.handleRevertVisitorConversion)))

	// Benfeitores
	mux.Handle("GET /api/v1/benefactors", authed(http.HandlerFunc(app.handleListBenefactors)))
	mux.Handle("POST /api/v1/benefactors", authed(http.HandlerFunc(app.handleCreateBenefactor)))

	// Fornecedores
	mux.Handle("GET /api/v1/suppliers", authed(http.HandlerFunc(app.handleListSuppliers)))
	mux.Handle("POST /api/v1/suppliers", authed(http.HandlerFunc(app.handleCreateSupplier)))
	mux.Handle("PATCH /api/v1/suppliers/{id}", authed(http.HandlerFunc(app.handleUpdateSupplier)))
	mux.Handle("DELETE /api/v1/suppliers/{id}", authed(http.HandlerFunc(app.handleDeleteSupplier)))

	// Financeiro
	mux.Handle("GET /api/v1/finance/categories", authed(http.HandlerFunc(app.handleListCategories)))
	mux.Handle("POST /api/v1/finance/categories", authed(http.HandlerFunc(app.handleCreateCategory)))
	mux.Handle("PATCH /api/v1/finance/categories/{id}", authed(http.HandlerFunc(app.handleUpdateCategory)))
	mux.Handle("DELETE /api/v1/finance/categories/{id}", authed(http.HandlerFunc(app.handleDeleteCategory)))
	mux.Handle("GET /api/v1/finance/category-groups", authed(http.HandlerFunc(app.handleListCategoryGroups)))
	mux.Handle("POST /api/v1/finance/category-groups", authed(http.HandlerFunc(app.handleCreateCategoryGroup)))
	mux.Handle("PATCH /api/v1/finance/category-groups/{id}", authed(http.HandlerFunc(app.handleUpdateCategoryGroup)))
	mux.Handle("DELETE /api/v1/finance/category-groups/{id}", authed(http.HandlerFunc(app.handleDeleteCategoryGroup)))
	mux.Handle("GET /api/v1/finance/accounts", authed(http.HandlerFunc(app.handleListAccounts)))
	mux.Handle("POST /api/v1/finance/accounts", authed(http.HandlerFunc(app.handleCreateAccount)))
	mux.Handle("PATCH /api/v1/finance/accounts/{id}", authed(http.HandlerFunc(app.handleUpdateAccount)))
	mux.Handle("DELETE /api/v1/finance/accounts/{id}", authed(http.HandlerFunc(app.handleDeleteAccount)))
	mux.Handle("GET /api/v1/finance/transactions", authed(http.HandlerFunc(app.handleListTxn)))
	mux.Handle("POST /api/v1/finance/transactions", authed(http.HandlerFunc(app.handleCreateTxn)))
	mux.Handle("POST /api/v1/finance/transactions/batch", authed(http.HandlerFunc(app.handleCreateTxnBatch)))
	mux.Handle("POST /api/v1/finance/transactions/import", authed(http.HandlerFunc(app.handleImportTransactions)))
	mux.Handle("POST /api/v1/finance/transactions/import/preview", authed(http.HandlerFunc(app.handlePreviewTransactions)))
	mux.Handle("POST /api/v1/finance/transactions/reorder", authed(http.HandlerFunc(app.handleReorderTxns)))
	mux.Handle("POST /api/v1/finance/transactions/{id}/void", authed(http.HandlerFunc(app.handleVoidTxn)))
	mux.Handle("POST /api/v1/finance/transactions/{id}/settle", authed(http.HandlerFunc(app.handleSettleTxn)))
	mux.Handle("GET /api/v1/finance/split", authed(http.HandlerFunc(app.handleGetSplit)))
	mux.Handle("PUT /api/v1/finance/split", authed(app.perm("finance.write", app.handlePutSplit)))
	mux.Handle("DELETE /api/v1/finance/transactions/{id}", authed(http.HandlerFunc(app.handleDeleteTxn)))
	mux.Handle("GET /api/v1/finance/transactions/{id}/events", authed(http.HandlerFunc(app.handleListTxnEvents)))
	mux.Handle("GET /api/v1/finance/transactions/{id}/attachments", authed(http.HandlerFunc(app.handleListAttachments)))
	mux.Handle("POST /api/v1/finance/transactions/{id}/attachments", authed(http.HandlerFunc(app.handleUploadAttachment)))
	mux.Handle("GET /api/v1/finance/balance", authed(http.HandlerFunc(app.handleBalance)))

	// Auditoria analitica do financeiro (documento imutavel apos fechamento)
	mux.Handle("GET /api/v1/finance/audits", authed(http.HandlerFunc(app.handleListAudits)))
	mux.Handle("POST /api/v1/finance/audits", authed(http.HandlerFunc(app.handleCreateAudit)))
	mux.Handle("GET /api/v1/finance/audits/{id}", authed(http.HandlerFunc(app.handleGetAudit)))
	mux.Handle("POST /api/v1/finance/audits/{id}/mark", authed(http.HandlerFunc(app.handleMarkAudit)))
	mux.Handle("POST /api/v1/finance/audits/{id}/close", authed(http.HandlerFunc(app.handleCloseAudit)))
	mux.Handle("GET /api/v1/finance/audits/{id}/export", authed(http.HandlerFunc(app.handleExportAudit)))
	mux.Handle("DELETE /api/v1/finance/audits/{id}", authed(http.HandlerFunc(app.handleDeleteAudit)))

	// Conciliacao financeira (trava o periodo ao conciliar)
	mux.Handle("GET /api/v1/finance/reconciliations", authed(http.HandlerFunc(app.handleListReconciliations)))
	mux.Handle("POST /api/v1/finance/reconciliations", authed(http.HandlerFunc(app.handleCreateReconciliation)))
	mux.Handle("GET /api/v1/finance/reconciliations/{id}", authed(http.HandlerFunc(app.handleGetReconciliation)))
	mux.Handle("POST /api/v1/finance/reconciliations/{id}/conciliate", authed(http.HandlerFunc(app.handleConciliateReconciliation)))
	mux.Handle("DELETE /api/v1/finance/reconciliations/{id}", authed(http.HandlerFunc(app.handleDeleteReconciliation)))

	// Conciliacao BANCARIA (importa extrato OFX/CSV e casa com os lancamentos)
	mux.Handle("GET /api/v1/finance/bank-imports", authed(http.HandlerFunc(app.handleListBankImports)))
	mux.Handle("POST /api/v1/finance/bank-imports", authed(http.HandlerFunc(app.handleImportBankStatement)))
	mux.Handle("GET /api/v1/finance/bank-imports/{id}", authed(http.HandlerFunc(app.handleGetBankImport)))
	mux.Handle("POST /api/v1/finance/bank-imports/{id}/entries/{entryId}/ignore", authed(http.HandlerFunc(app.handleIgnoreBankEntry)))
	mux.Handle("POST /api/v1/finance/bank-imports/{id}/entries/{entryId}/generate", authed(http.HandlerFunc(app.handleGenerateBankEntry)))
	mux.Handle("DELETE /api/v1/finance/bank-imports/{id}", authed(http.HandlerFunc(app.handleDeleteBankImport)))

	// Download de anexos (arquivo por nome - opaco, nao requer auth)
	mux.Handle("GET /api/v1/attachments/{filename}", http.HandlerFunc(app.handleDownloadAttachment))

	// Documentos digitais (leitura por token do QR)
	mux.Handle("GET /api/v1/documents/by-token/{token}", authed(http.HandlerFunc(app.handleGetDocumentByToken)))
	mux.Handle("GET /api/v1/member-documents/{id}/html", authed(http.HandlerFunc(app.handleRenderDocument)))

	// Recibos: renderizacao e envio (prefixo proprio evita ambiguidade de rotas)
	mux.Handle("GET /api/v1/receipts/{id}", authed(http.HandlerFunc(app.handleRenderReceipt)))
	mux.Handle("POST /api/v1/receipts/{id}/send", authed(http.HandlerFunc(app.handleSendDocument)))
	mux.Handle("GET /api/v1/receipts/{id}/deliveries", authed(http.HandlerFunc(app.handleListDeliveries)))

	// Relatorios
	mux.Handle("GET /api/v1/reports/balance", authed(http.HandlerFunc(app.handleMonthlyBalance)))
	mux.Handle("GET /api/v1/reports/dre", authed(http.HandlerFunc(app.handleDRE)))
	mux.Handle("GET /api/v1/reports/balance/export", authed(http.HandlerFunc(app.handleExportBalance)))
	mux.Handle("GET /api/v1/reports/dre/export", authed(http.HandlerFunc(app.handleExportDRE)))
	mux.Handle("GET /api/v1/reports/birthdays", authed(http.HandlerFunc(app.handleBirthdays)))
	mux.Handle("GET /api/v1/reports/contribution-drop", authed(http.HandlerFunc(app.handleContributionDrops)))
	mux.Handle("GET /api/v1/reports/birthdays/export", authed(http.HandlerFunc(app.handleExportBirthdays)))
	mux.Handle("GET /api/v1/reports/demographics", authed(http.HandlerFunc(app.handleDemographics)))
	mux.Handle("GET /api/v1/reports/demographics/export", authed(http.HandlerFunc(app.handleExportDemographics)))
	mux.Handle("GET /api/v1/reports/monthly-statement", authed(http.HandlerFunc(app.handleMonthlyStatement)))
	mux.Handle("GET /api/v1/reports/monthly-statement/export", authed(http.HandlerFunc(app.handleExportMonthlyStatement)))
	mux.Handle("GET /api/v1/reports/consolidated", authed(http.HandlerFunc(app.handleConsolidatedReport)))
	mux.Handle("GET /api/v1/reports/assembly", authed(http.HandlerFunc(app.handleAssemblyReport)))
	mux.Handle("GET /api/v1/reports/assembly/export", authed(http.HandlerFunc(app.handleExportAssembly)))
	mux.Handle("GET /api/v1/reports/attendance", authed(http.HandlerFunc(app.handleAttendanceReport)))
	mux.Handle("GET /api/v1/reports/attendance/export", authed(http.HandlerFunc(app.handleExportAttendance)))

	// Branches (selecao de filial em repasses) + configuracao do tenant
	mux.Handle("GET /api/v1/branches", authed(http.HandlerFunc(app.handleListBranches)))
	mux.Handle("POST /api/v1/branches", authed(http.HandlerFunc(app.handleCreateBranch)))
	mux.Handle("PATCH /api/v1/branches/{id}", authed(http.HandlerFunc(app.handleUpdateBranch)))
	mux.Handle("DELETE /api/v1/branches/{id}", authed(http.HandlerFunc(app.handleDeleteBranch)))

	// Canais de envio por filial (WhatsApp via Evolution + SMTP).
	mux.Handle("GET /api/v1/branches/{id}/channels", authed(http.HandlerFunc(app.handleGetBranchChannels)))
	mux.Handle("PATCH /api/v1/branches/{id}/channels", authed(http.HandlerFunc(app.handleUpdateBranchChannels)))
	mux.Handle("POST /api/v1/branches/{id}/whatsapp/connect", authed(http.HandlerFunc(app.handleConnectBranchWhatsApp)))
	mux.Handle("GET /api/v1/branches/{id}/whatsapp/state", authed(http.HandlerFunc(app.handleBranchWhatsAppState)))
	mux.Handle("POST /api/v1/branches/{id}/whatsapp/logout", authed(http.HandlerFunc(app.handleDisconnectBranchWhatsApp)))
	mux.Handle("GET /api/v1/tenant", authed(http.HandlerFunc(app.handleGetTenant)))
	mux.Handle("PATCH /api/v1/tenant", authed(http.HandlerFunc(app.handleUpdateTenant)))

	// Repasses entre filiais
	mux.Handle("GET /api/v1/finance/transfers", authed(http.HandlerFunc(app.handleListTransfers)))
	mux.Handle("POST /api/v1/finance/transfers", authed(http.HandlerFunc(app.handleCreateTransfer)))

	// Doacoes recorrentes
	mux.Handle("GET /api/v1/finance/recurring", authed(http.HandlerFunc(app.handleListRecurring)))
	mux.Handle("POST /api/v1/finance/recurring", authed(http.HandlerFunc(app.handleCreateRecurring)))
	mux.Handle("PATCH /api/v1/finance/recurring/{id}", authed(http.HandlerFunc(app.handleUpdateRecurring)))

	// Ministerios / escalas
	mux.Handle("GET /api/v1/ministries", authed(http.HandlerFunc(app.handleListMinistries)))
	mux.Handle("POST /api/v1/ministries", authed(http.HandlerFunc(app.handleCreateMinistry)))
	mux.Handle("PATCH /api/v1/ministries/{id}", authed(http.HandlerFunc(app.handleUpdateMinistry)))
	mux.Handle("DELETE /api/v1/ministries/{id}", authed(http.HandlerFunc(app.handleDeleteMinistry)))
	mux.Handle("GET /api/v1/ministries/{id}/members", authed(http.HandlerFunc(app.handleListMinistryMembers)))
	mux.Handle("POST /api/v1/ministries/{id}/members", authed(http.HandlerFunc(app.handleAddMinistryMember)))
	mux.Handle("POST /api/v1/ministries/{id}/members/batch", authed(http.HandlerFunc(app.handleAddMinistryMembersBatch)))
	mux.Handle("DELETE /api/v1/ministries/{id}/members/{memberId}", authed(http.HandlerFunc(app.handleRemoveMinistryMember)))

	// Escalas de voluntarios
	mux.Handle("GET /api/v1/rosters", authed(http.HandlerFunc(app.handleListRosters)))
	mux.Handle("POST /api/v1/rosters", authed(http.HandlerFunc(app.handleCreateRoster)))
	mux.Handle("GET /api/v1/rosters/suggestions", authed(http.HandlerFunc(app.handleRosterSuggestions)))
	mux.Handle("GET /api/v1/rosters/{id}", authed(http.HandlerFunc(app.handleGetRoster)))
	mux.Handle("PATCH /api/v1/rosters/{id}", authed(http.HandlerFunc(app.handleUpdateRoster)))
	mux.Handle("DELETE /api/v1/rosters/{id}", authed(http.HandlerFunc(app.handleDeleteRoster)))
	mux.Handle("POST /api/v1/rosters/{id}/assignments", authed(http.HandlerFunc(app.handleSetRosterAssignments)))
	mux.Handle("PATCH /api/v1/rosters/{id}/assignments/{assignmentId}", authed(http.HandlerFunc(app.handleRespondRosterAssignment)))
	mux.Handle("GET /api/v1/rosters/{id}/conflicts", authed(http.HandlerFunc(app.handleRosterConflicts)))

	// Pequenos grupos / celulas + check-in
	mux.Handle("GET /api/v1/groups", authed(http.HandlerFunc(app.handleListGroups)))
	mux.Handle("POST /api/v1/groups", authed(http.HandlerFunc(app.handleCreateGroup)))
	mux.Handle("PATCH /api/v1/groups/{id}", authed(http.HandlerFunc(app.handleUpdateGroup)))
	mux.Handle("DELETE /api/v1/groups/{id}", authed(http.HandlerFunc(app.handleDeleteGroup)))
	mux.Handle("POST /api/v1/groups/{id}/attendance", authed(http.HandlerFunc(app.handleCheckIn)))
	mux.Handle("GET /api/v1/groups/{id}/attendance", authed(http.HandlerFunc(app.handleListEventAttendance)))
	mux.Handle("GET /api/v1/groups/{id}/members", authed(http.HandlerFunc(app.handleListGroupMembers)))
	mux.Handle("POST /api/v1/groups/{id}/members", authed(http.HandlerFunc(app.handleAddGroupMember)))
	mux.Handle("DELETE /api/v1/groups/{id}/members/{memberId}", authed(http.HandlerFunc(app.handleRemoveGroupMember)))

	// ---- Materiais de estudo (arquivo/link) ----
	mux.Handle("GET /api/v1/materials", authed(app.perm("ministries.read", app.handleListMaterials)))
	mux.Handle("POST /api/v1/materials", authed(app.perm("ministries.write", app.handleCreateMaterial)))
	mux.Handle("DELETE /api/v1/materials/{id}", authed(app.perm("ministries.write", app.handleDeleteMaterial)))
	mux.Handle("GET /api/v1/materials/{id}/file", authed(app.perm("ministries.read", app.handleDownloadMaterial)))

	// Avisos (app do membro) - gestao + disparo
	mux.Handle("GET /api/v1/announcements", authed(http.HandlerFunc(app.handleListAnnouncements)))
	mux.Handle("POST /api/v1/announcements", authed(http.HandlerFunc(app.handleCreateAnnouncement)))
	mux.Handle("PATCH /api/v1/announcements/{id}", authed(http.HandlerFunc(app.handleUpdateAnnouncement)))
	mux.Handle("DELETE /api/v1/announcements/{id}", authed(http.HandlerFunc(app.handleDeleteAnnouncement)))
	mux.Handle("POST /api/v1/announcements/audience/preview", authed(http.HandlerFunc(app.handlePreviewAudience)))
	mux.Handle("POST /api/v1/announcements/{id}/send", authed(http.HandlerFunc(app.handleSendAnnouncement)))
	mux.Handle("GET /api/v1/announcements/{id}/deliveries", authed(http.HandlerFunc(app.handleListAnnouncementDeliveries)))
	mux.Handle("POST /api/v1/announcements/send-test", authed(http.HandlerFunc(app.handleSendTestMessage)))

	// Historico de disparos agendados (tela Execucoes)
	mux.Handle("GET /api/v1/notification-runs", authed(http.HandlerFunc(app.handleListNotificationRuns)))

	// Automacoes de WhatsApp (#31): aniversario, lembrete de escala e boas-vindas
	mux.Handle("GET /api/v1/notifications/settings", authed(http.HandlerFunc(app.handleGetNotificationSettings)))
	mux.Handle("PATCH /api/v1/notifications/settings", authed(http.HandlerFunc(app.handleUpdateNotificationSettings)))
	mux.Handle("POST /api/v1/notifications/run", authed(http.HandlerFunc(app.handleRunNotifications)))

	// Usuarios e acessos (autorizacao por papel no handler; RLS isola o tenant)
	mux.Handle("GET /api/v1/users", authed(http.HandlerFunc(app.handleListUsers)))
	mux.Handle("POST /api/v1/users", authed(http.HandlerFunc(app.handleCreateUser)))
	mux.Handle("PATCH /api/v1/users/{id}", authed(http.HandlerFunc(app.handleUpdateUser)))
	mux.Handle("POST /api/v1/users/{id}/password", authed(http.HandlerFunc(app.handleResetUserPassword)))
	mux.Handle("GET /api/v1/roles", authed(http.HandlerFunc(app.handleListRoles)))
	mux.Handle("GET /api/v1/permissions", authed(http.HandlerFunc(app.handleListPermissions)))

	// Reset operacional (platform admin): limpa os dados de teste mantendo a base.
	mux.Handle("POST /api/v1/admin/reset-data", authed(http.HandlerFunc(app.handleResetData)))
	// Igrejas (console da plataforma): listar e criar igreja + admin.
	mux.Handle("GET /api/v1/admin/tenants", authed(http.HandlerFunc(app.handleListTenants)))
	mux.Handle("POST /api/v1/admin/tenants", authed(http.HandlerFunc(app.handleCreateTenant)))
	mux.Handle("GET /api/v1/admin/tenants/{id}", authed(http.HandlerFunc(app.handleAdminGetTenant)))
	mux.Handle("PATCH /api/v1/admin/tenants/{id}", authed(http.HandlerFunc(app.handleAdminUpdateTenant)))
	mux.Handle("GET /api/v1/admin/tenants/{id}/usage", authed(http.HandlerFunc(app.handleAdminTenantUsage)))
	// Acessos de uma igreja (support da plataforma): usuarios, papeis e filiais.
	mux.Handle("GET /api/v1/admin/tenants/{id}/users", authed(http.HandlerFunc(app.handleAdminListTenantUsers)))
	mux.Handle("POST /api/v1/admin/tenants/{id}/users", authed(http.HandlerFunc(app.handleAdminCreateTenantUser)))
	mux.Handle("PATCH /api/v1/admin/tenants/{id}/users/{userId}", authed(http.HandlerFunc(app.handleAdminUpdateTenantUser)))
	mux.Handle("POST /api/v1/admin/tenants/{id}/users/{userId}/password", authed(http.HandlerFunc(app.handleAdminResetTenantUserPassword)))
	mux.Handle("GET /api/v1/admin/tenants/{id}/roles", authed(http.HandlerFunc(app.handleAdminListTenantRoles)))
	mux.Handle("GET /api/v1/admin/tenants/{id}/branches", authed(http.HandlerFunc(app.handleAdminListTenantBranches)))
	// Estatisticas gerais da plataforma
	mux.Handle("GET /api/v1/admin/stats", authed(http.HandlerFunc(app.handleAdminStats)))
	// Catalogo de planos (platform admin)
	mux.Handle("GET /api/v1/admin/plans", authed(http.HandlerFunc(app.handleListPlans)))
	mux.Handle("POST /api/v1/admin/plans", authed(http.HandlerFunc(app.handleCreatePlan)))
	mux.Handle("PATCH /api/v1/admin/plans/{key}", authed(http.HandlerFunc(app.handleUpdatePlan)))
	// Catalogo de modulos gateaveis (features)
	mux.Handle("GET /api/v1/admin/features", authed(http.HandlerFunc(app.handleListFeatures)))

	// MFA (TOTP) do proprio usuario
	mux.Handle("GET /api/v1/auth/mfa", authed(http.HandlerFunc(app.handleMFAStatus)))
	mux.Handle("POST /api/v1/auth/mfa/setup", authed(http.HandlerFunc(app.handleMFASetup)))
	mux.Handle("POST /api/v1/auth/mfa/enable", authed(http.HandlerFunc(app.handleMFAEnable)))
	mux.Handle("POST /api/v1/auth/mfa/disable", authed(http.HandlerFunc(app.handleMFADisable)))

	// Eventos, tipos de evento (catalogo selado, somente leitura), chamada nominal e frequencia do membro
	mux.Handle("GET /api/v1/event-kinds", authed(http.HandlerFunc(app.handleListEventKinds)))
	mux.Handle("GET /api/v1/events", authed(http.HandlerFunc(app.handleListEvents)))
	mux.Handle("POST /api/v1/events", authed(http.HandlerFunc(app.handleCreateEvent)))
	mux.Handle("GET /api/v1/events/{id}", authed(http.HandlerFunc(app.handleGetEvent)))
	mux.Handle("PATCH /api/v1/events/{id}", authed(http.HandlerFunc(app.handleUpdateEvent)))
	mux.Handle("DELETE /api/v1/events/{id}", authed(http.HandlerFunc(app.handleDeleteEvent)))
	mux.Handle("GET /api/v1/events/{id}/attendance", authed(http.HandlerFunc(app.handleListEventAttendance)))
	mux.Handle("POST /api/v1/events/{id}/attendance", authed(http.HandlerFunc(app.handleSaveEventAttendance)))
	mux.Handle("GET /api/v1/events/{id}/invitees", authed(http.HandlerFunc(app.handleListInvitees)))
	mux.Handle("POST /api/v1/events/{id}/invitees", authed(http.HandlerFunc(app.handleSetInvitees)))
	mux.Handle("GET /api/v1/members/{id}/frequency", authed(http.HandlerFunc(app.handleListFrequency)))
	mux.Handle("POST /api/v1/members/{id}/frequency", authed(http.HandlerFunc(app.handleSetFrequency)))

	// Programacao: grade de horarios recorrentes e publicacao na agenda
	mux.Handle("GET /api/v1/programacoes", authed(http.HandlerFunc(app.handleListProgramacao)))
	mux.Handle("POST /api/v1/programacoes", authed(http.HandlerFunc(app.handleCreateProgramacao)))
	mux.Handle("POST /api/v1/programacoes/generate", authed(http.HandlerFunc(app.handleGenerateProgramacao)))
	mux.Handle("PATCH /api/v1/programacoes/{id}", authed(http.HandlerFunc(app.handleUpdateProgramacao)))
	mux.Handle("DELETE /api/v1/programacoes/{id}", authed(http.HandlerFunc(app.handleDeleteProgramacao)))

	// LGPD: consentimento, portabilidade e anonimizacao
	mux.Handle("GET /api/v1/consent-terms", authed(http.HandlerFunc(app.handleListConsentTerms)))
	mux.Handle("POST /api/v1/consent-terms", authed(http.HandlerFunc(app.handleCreateConsentTerm)))
	mux.Handle("GET /api/v1/members/{id}/consents", authed(http.HandlerFunc(app.handleListMemberConsents)))
	mux.Handle("POST /api/v1/members/{id}/consents", authed(http.HandlerFunc(app.handleRecordConsent)))
	mux.Handle("GET /api/v1/members/{id}/export", authed(http.HandlerFunc(app.handleExportMemberData)))
	mux.Handle("POST /api/v1/members/{id}/anonymize", authed(http.HandlerFunc(app.handleAnonymizeMember)))

	// Governanca (Modulo 6): atas, assinatura interna, votacao e convenios
	mux.Handle("GET /api/v1/minutes", authed(http.HandlerFunc(app.handleListMinutes)))
	mux.Handle("POST /api/v1/minutes", authed(http.HandlerFunc(app.handleCreateMinute)))
	mux.Handle("GET /api/v1/minutes/{id}", authed(http.HandlerFunc(app.handleGetMinute)))
	mux.Handle("PATCH /api/v1/minutes/{id}", authed(http.HandlerFunc(app.handleUpdateMinute)))
	mux.Handle("DELETE /api/v1/minutes/{id}", authed(http.HandlerFunc(app.handleDeleteMinute)))
	mux.Handle("POST /api/v1/minutes/{id}/sign", authed(http.HandlerFunc(app.handleSignMinute)))
	mux.Handle("GET /api/v1/minutes/{id}/signatures", authed(http.HandlerFunc(app.handleListSignatures)))
	mux.Handle("GET /api/v1/votes", authed(http.HandlerFunc(app.handleListVotes)))
	mux.Handle("POST /api/v1/votes", authed(http.HandlerFunc(app.handleCreateVote)))
	mux.Handle("GET /api/v1/votes/{id}", authed(http.HandlerFunc(app.handleGetVote)))
	mux.Handle("PATCH /api/v1/votes/{id}", authed(http.HandlerFunc(app.handleUpdateVote)))
	mux.Handle("DELETE /api/v1/votes/{id}", authed(http.HandlerFunc(app.handleDeleteVote)))
	mux.Handle("POST /api/v1/votes/{id}/open", authed(http.HandlerFunc(app.handleOpenVote)))
	mux.Handle("POST /api/v1/votes/{id}/close", authed(http.HandlerFunc(app.handleCloseVote)))
	mux.Handle("POST /api/v1/votes/{id}/ballot", authed(http.HandlerFunc(app.handleCastBallot)))
	mux.Handle("GET /api/v1/votes/{id}/result", authed(http.HandlerFunc(app.handleVoteResult)))
	mux.Handle("GET /api/v1/legal-documents", authed(http.HandlerFunc(app.handleListLegalDocuments)))
	mux.Handle("POST /api/v1/legal-documents", authed(http.HandlerFunc(app.handleCreateLegalDocument)))
	mux.Handle("PATCH /api/v1/legal-documents/{id}", authed(http.HandlerFunc(app.handleUpdateLegalDocument)))
	mux.Handle("DELETE /api/v1/legal-documents/{id}", authed(http.HandlerFunc(app.handleDeleteLegalDocument)))
	mux.Handle("GET /api/v1/governance/mandates", authed(http.HandlerFunc(app.handleListMandates)))

	// Ministerio Infantil (Kids): trilha/conteudo, turmas, participantes,
	// encontros, check-in e evolucao.
	mux.Handle("GET /api/v1/kids/tracks", authed(http.HandlerFunc(app.handleListKidsTracks)))
	mux.Handle("POST /api/v1/kids/tracks", authed(http.HandlerFunc(app.handleCreateKidsTrack)))
	mux.Handle("PATCH /api/v1/kids/tracks/{id}", authed(http.HandlerFunc(app.handleUpdateKidsTrack)))
	mux.Handle("DELETE /api/v1/kids/tracks/{id}", authed(http.HandlerFunc(app.handleDeleteKidsTrack)))
	mux.Handle("GET /api/v1/kids/tracks/{id}/lessons", authed(http.HandlerFunc(app.handleListKidsLessons)))
	mux.Handle("POST /api/v1/kids/tracks/{id}/lessons", authed(http.HandlerFunc(app.handleCreateKidsLesson)))
	mux.Handle("PATCH /api/v1/kids/lessons/{id}", authed(http.HandlerFunc(app.handleUpdateKidsLesson)))
	mux.Handle("DELETE /api/v1/kids/lessons/{id}", authed(http.HandlerFunc(app.handleDeleteKidsLesson)))
	mux.Handle("GET /api/v1/kids/classes", authed(http.HandlerFunc(app.handleListKidsClasses)))
	mux.Handle("POST /api/v1/kids/classes", authed(http.HandlerFunc(app.handleCreateKidsClass)))
	mux.Handle("PATCH /api/v1/kids/classes/{id}", authed(http.HandlerFunc(app.handleUpdateKidsClass)))
	mux.Handle("DELETE /api/v1/kids/classes/{id}", authed(http.HandlerFunc(app.handleDeleteKidsClass)))
	mux.Handle("GET /api/v1/kids/classes/{id}/enrollments", authed(http.HandlerFunc(app.handleListKidsEnrollments)))
	mux.Handle("POST /api/v1/kids/classes/{id}/enrollments", authed(http.HandlerFunc(app.handleCreateKidsEnrollment)))
	mux.Handle("PATCH /api/v1/kids/enrollments/{id}", authed(http.HandlerFunc(app.handleUpdateKidsEnrollment)))
	mux.Handle("DELETE /api/v1/kids/enrollments/{id}", authed(http.HandlerFunc(app.handleDeleteKidsEnrollment)))
	mux.Handle("GET /api/v1/kids/enrollments/{id}/guardians", authed(http.HandlerFunc(app.handleListKidsGuardians)))
	mux.Handle("POST /api/v1/kids/enrollments/{id}/guardians", authed(http.HandlerFunc(app.handleAddKidsGuardian)))
	mux.Handle("DELETE /api/v1/kids/guardians/{id}", authed(http.HandlerFunc(app.handleDeleteKidsGuardian)))
	mux.Handle("GET /api/v1/kids/sessions", authed(http.HandlerFunc(app.handleListKidsSessions)))
	mux.Handle("POST /api/v1/kids/sessions", authed(http.HandlerFunc(app.handleCreateKidsSession)))
	mux.Handle("PATCH /api/v1/kids/sessions/{id}", authed(http.HandlerFunc(app.handleUpdateKidsSession)))
	mux.Handle("DELETE /api/v1/kids/sessions/{id}", authed(http.HandlerFunc(app.handleDeleteKidsSession)))
	mux.Handle("GET /api/v1/kids/sessions/{id}/roster", authed(http.HandlerFunc(app.handleKidsRoster)))
	mux.Handle("POST /api/v1/kids/sessions/{id}/checkin", authed(http.HandlerFunc(app.handleKidsCheckin)))
	mux.Handle("POST /api/v1/kids/sessions/{id}/checkout", authed(http.HandlerFunc(app.handleKidsCheckout)))
	mux.Handle("POST /api/v1/kids/sessions/{id}/absence", authed(http.HandlerFunc(app.handleKidsAbsence)))
	mux.Handle("GET /api/v1/kids/classes/{id}/evolution", authed(http.HandlerFunc(app.handleKidsEvolution)))

	// App do membro (publico por token do QR). As tres rotas sao as unicas sem
	// auth: passam pelo `publico`, que aplica noindex e limite por IP. A foto
	// tem endpoint proprio para o Access poder liberar so ela, e nao
	// /api/v1/attachments/{arquivo}, que serve tambem comprovante financeiro.
	mux.Handle("GET /api/v1/public/card/{token}", app.publico(app.handlePublicCard))
	mux.Handle("GET /api/v1/public/card/{token}/print", app.publico(app.handleMemberCardHTML))
	mux.Handle("GET /api/v1/public/card/{token}/photo", app.publico(app.handlePublicCardPhoto))

	return withCORS(mux)
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Branch-Id, X-Tenant-Id, X-Tenant-Slug")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
