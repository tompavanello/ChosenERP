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
	mux.Handle("GET /api/v1/me/materials/{id}/file", authed(http.HandlerFunc(app.handleDownloadMyMaterial)))
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
	mux.Handle("GET /api/v1/members", authed(app.perm("members.read", app.handleListMembers)))
	mux.Handle("POST /api/v1/members", authed(app.perm("members.write", app.handleCreateMember)))
	mux.Handle("GET /api/v1/members/{id}", authed(app.perm("members.read", app.handleGetMember)))
	mux.Handle("PATCH /api/v1/members/{id}", authed(app.perm("members.write", app.handleUpdateMember)))
	mux.Handle("DELETE /api/v1/members/{id}", authed(app.perm("members.delete", app.handleDeleteMember)))
	mux.Handle("POST /api/v1/members/{id}/transfer", authed(app.perm("members.write", app.handleTransferMember)))
	mux.Handle("GET /api/v1/members/{id}/documents", authed(app.perm("members.read", app.handleListMemberDocuments)))
	mux.Handle("POST /api/v1/members/{id}/documents", authed(app.perm("members.write", app.handleIssueMemberDocument)))
	mux.Handle("GET /api/v1/members/{id}/tree", authed(app.perm("members.read", app.handleMemberTree)))
	mux.Handle("POST /api/v1/members/{id}/relationships", authed(app.perm("members.write", app.handleAddRelationship)))
	mux.Handle("POST /api/v1/members/{id}/card", authed(app.perm("members.write", app.handleIssueCard)))
	mux.Handle("GET /api/v1/members/{id}/card", authed(app.perm("members.read", app.handleGetCard)))
	mux.Handle("POST /api/v1/members/{id}/photo", authed(app.perm("members.write", app.handleUploadMemberPhoto)))
	mux.Handle("DELETE /api/v1/members/{id}/photo", authed(app.perm("members.write", app.handleDeleteMemberPhoto)))
	mux.Handle("GET /api/v1/members/{id}/families", authed(app.perm("members.read", app.handleMemberFamilies)))

	// Acesso do membro ao app (Sede): criar/redefinir senha provisoria, editar
	// identificador e ativar/desativar.
	mux.Handle("GET /api/v1/members/{id}/access", authed(app.adminOnly(app.handleGetMemberAccess)))
	mux.Handle("POST /api/v1/members/{id}/access", authed(app.adminOnly(app.handleCreateMemberAccess)))
	mux.Handle("PATCH /api/v1/members/{id}/access", authed(app.adminOnly(app.handleUpdateMemberAccess)))
	mux.Handle("POST /api/v1/members/{id}/access/password", authed(app.adminOnly(app.handleResetMemberAccessPassword)))

	// Historico eclesiastico do membro (requisito 1.8)
	mux.Handle("GET /api/v1/members/{id}/history", authed(app.perm("members.read", app.handleListMemberHistory)))
	mux.Handle("POST /api/v1/members/{id}/history", authed(app.perm("members.write", app.handleAddMemberHistory)))

	// Catalogo configuravel de eventos da vida eclesiastica (000065)
	mux.Handle("GET /api/v1/member-event-kinds", authed(app.perm("members.read", app.handleListMemberEventKinds)))
	mux.Handle("POST /api/v1/member-event-kinds", authed(app.perm("members.write", app.handleCreateMemberEventKind)))
	mux.Handle("PATCH /api/v1/member-event-kinds/{id}", authed(app.perm("members.write", app.handleUpdateMemberEventKind)))
	mux.Handle("DELETE /api/v1/member-event-kinds/{id}", authed(app.perm("members.write", app.handleDeleteMemberEventKind)))

	// Cargos (funcoes/ministerios) e mandatos do membro
	mux.Handle("GET /api/v1/cargos", authed(app.perm("members.read", app.handleListCargos)))
	mux.Handle("POST /api/v1/cargos", authed(app.perm("members.write", app.handleCreateCargo)))
	mux.Handle("PATCH /api/v1/cargos/{id}", authed(app.perm("members.write", app.handleUpdateCargo)))
	mux.Handle("DELETE /api/v1/cargos/{id}", authed(app.perm("members.write", app.handleDeleteCargo)))
	mux.Handle("GET /api/v1/members/{id}/cargos", authed(app.perm("members.read", app.handleListMemberCargos)))
	mux.Handle("POST /api/v1/members/{id}/cargos", authed(app.perm("members.write", app.handleAssignCargo)))
	mux.Handle("PATCH /api/v1/members/{id}/cargos/{linkId}", authed(app.perm("members.write", app.handleUpdateMemberCargo)))
	mux.Handle("DELETE /api/v1/members/{id}/cargos/{linkId}", authed(app.perm("members.write", app.handleUnassignCargo)))

	// Familias
	mux.Handle("GET /api/v1/families", authed(app.perm("families.read", app.handleListFamilies)))
	mux.Handle("POST /api/v1/families", authed(app.perm("members.write", app.handleCreateFamily)))
	mux.Handle("GET /api/v1/families/{id}", authed(app.perm("families.read", app.handleGetFamily)))
	mux.Handle("PATCH /api/v1/families/{id}", authed(app.perm("members.write", app.handleUpdateFamily)))
	mux.Handle("DELETE /api/v1/families/{id}", authed(app.perm("members.write", app.handleDeleteFamily)))
	mux.Handle("GET /api/v1/families/{id}/members", authed(app.perm("families.read", app.handleFamilyMembers)))
	mux.Handle("POST /api/v1/families/{id}/members", authed(app.perm("members.write", app.handleAddFamilyMember)))
	mux.Handle("DELETE /api/v1/families/{id}/members/{memberId}", authed(app.perm("members.write", app.handleUnlinkFamilyMember)))

	// Visitantes
	mux.Handle("GET /api/v1/visitors", authed(app.perm("visitors.read", app.handleListVisitors)))
	mux.Handle("POST /api/v1/visitors", authed(app.perm("members.write", app.handleCreateVisitor)))
	mux.Handle("PATCH /api/v1/visitors/{id}/stage", authed(app.perm("members.write", app.handleUpdateVisitorStage)))
	mux.Handle("POST /api/v1/visitors/{id}/convert", authed(app.perm("members.write", app.handleConvertVisitor)))
	mux.Handle("POST /api/v1/visitors/{id}/revert", authed(app.perm("members.write", app.handleRevertVisitorConversion)))

	// Benfeitores
	mux.Handle("GET /api/v1/benefactors", authed(app.perm("members.read", app.handleListBenefactors)))
	mux.Handle("POST /api/v1/benefactors", authed(app.perm("members.write", app.handleCreateBenefactor)))

	// Fornecedores
	mux.Handle("GET /api/v1/suppliers", authed(app.perm("finance.read", app.handleListSuppliers)))
	mux.Handle("POST /api/v1/suppliers", authed(app.perm("finance.write", app.handleCreateSupplier)))
	mux.Handle("PATCH /api/v1/suppliers/{id}", authed(app.perm("finance.write", app.handleUpdateSupplier)))
	mux.Handle("DELETE /api/v1/suppliers/{id}", authed(app.perm("finance.write", app.handleDeleteSupplier)))

	// Financeiro
	mux.Handle("GET /api/v1/finance/categories", authed(app.perm("finance.read", app.handleListCategories)))
	mux.Handle("POST /api/v1/finance/categories", authed(app.perm("finance.write", app.handleCreateCategory)))
	mux.Handle("PATCH /api/v1/finance/categories/{id}", authed(app.perm("finance.write", app.handleUpdateCategory)))
	mux.Handle("DELETE /api/v1/finance/categories/{id}", authed(app.perm("finance.write", app.handleDeleteCategory)))
	mux.Handle("GET /api/v1/finance/category-groups", authed(app.perm("finance.read", app.handleListCategoryGroups)))
	mux.Handle("POST /api/v1/finance/category-groups", authed(app.perm("finance.write", app.handleCreateCategoryGroup)))
	mux.Handle("PATCH /api/v1/finance/category-groups/{id}", authed(app.perm("finance.write", app.handleUpdateCategoryGroup)))
	mux.Handle("DELETE /api/v1/finance/category-groups/{id}", authed(app.perm("finance.write", app.handleDeleteCategoryGroup)))
	mux.Handle("GET /api/v1/finance/accounts", authed(app.perm("finance.read", app.handleListAccounts)))
	mux.Handle("POST /api/v1/finance/accounts", authed(app.perm("finance.write", app.handleCreateAccount)))
	mux.Handle("PATCH /api/v1/finance/accounts/{id}", authed(app.perm("finance.write", app.handleUpdateAccount)))
	mux.Handle("DELETE /api/v1/finance/accounts/{id}", authed(app.perm("finance.write", app.handleDeleteAccount)))
	mux.Handle("GET /api/v1/finance/transactions", authed(app.perm("finance.read", app.handleListTxn)))
	mux.Handle("POST /api/v1/finance/transactions", authed(app.perm("finance.write", app.handleCreateTxn)))
	mux.Handle("POST /api/v1/finance/transactions/batch", authed(app.perm("finance.write", app.handleCreateTxnBatch)))
	mux.Handle("POST /api/v1/finance/transactions/import", authed(app.perm("finance.write", app.handleImportTransactions)))
	mux.Handle("POST /api/v1/finance/transactions/import/preview", authed(app.perm("finance.write", app.handlePreviewTransactions)))
	mux.Handle("POST /api/v1/finance/transactions/reorder", authed(app.perm("finance.write", app.handleReorderTxns)))
	mux.Handle("POST /api/v1/finance/transactions/{id}/void", authed(app.perm("finance.write", app.handleVoidTxn)))
	mux.Handle("POST /api/v1/finance/transactions/{id}/settle", authed(app.perm("finance.write", app.handleSettleTxn)))
	mux.Handle("GET /api/v1/finance/split", authed(app.perm("finance.read", app.handleGetSplit)))
	mux.Handle("PUT /api/v1/finance/split", authed(app.perm("finance.write", app.handlePutSplit)))
	mux.Handle("DELETE /api/v1/finance/transactions/{id}", authed(app.perm("finance.write", app.handleDeleteTxn)))
	mux.Handle("GET /api/v1/finance/transactions/{id}/events", authed(app.perm("finance.read", app.handleListTxnEvents)))
	mux.Handle("GET /api/v1/finance/transactions/{id}/attachments", authed(app.perm("finance.read", app.handleListAttachments)))
	mux.Handle("POST /api/v1/finance/transactions/{id}/attachments", authed(app.perm("finance.write", app.handleUploadAttachment)))
	mux.Handle("GET /api/v1/finance/balance", authed(app.perm("finance.read", app.handleBalance)))

	// Auditoria analitica do financeiro (documento imutavel apos fechamento)
	mux.Handle("GET /api/v1/finance/audits", authed(app.perm("finance.read", app.handleListAudits)))
	mux.Handle("POST /api/v1/finance/audits", authed(app.perm("finance.write", app.handleCreateAudit)))
	mux.Handle("GET /api/v1/finance/audits/{id}", authed(app.perm("finance.read", app.handleGetAudit)))
	mux.Handle("POST /api/v1/finance/audits/{id}/mark", authed(app.perm("finance.write", app.handleMarkAudit)))
	mux.Handle("POST /api/v1/finance/audits/{id}/close", authed(app.perm("finance.write", app.handleCloseAudit)))
	mux.Handle("GET /api/v1/finance/audits/{id}/export", authed(app.perm("finance.read", app.handleExportAudit)))
	mux.Handle("DELETE /api/v1/finance/audits/{id}", authed(app.perm("finance.write", app.handleDeleteAudit)))

	// Conciliacao financeira (trava o periodo ao conciliar)
	mux.Handle("GET /api/v1/finance/reconciliations", authed(app.perm("finance.read", app.handleListReconciliations)))
	mux.Handle("POST /api/v1/finance/reconciliations", authed(app.perm("finance.write", app.handleCreateReconciliation)))
	mux.Handle("GET /api/v1/finance/reconciliations/{id}", authed(app.perm("finance.read", app.handleGetReconciliation)))
	mux.Handle("POST /api/v1/finance/reconciliations/{id}/conciliate", authed(app.perm("finance.write", app.handleConciliateReconciliation)))
	mux.Handle("DELETE /api/v1/finance/reconciliations/{id}", authed(app.perm("finance.write", app.handleDeleteReconciliation)))

	// Conciliacao BANCARIA (importa extrato OFX/CSV e casa com os lancamentos)
	mux.Handle("GET /api/v1/finance/bank-imports", authed(app.perm("finance.read", app.handleListBankImports)))
	mux.Handle("POST /api/v1/finance/bank-imports", authed(app.perm("finance.write", app.handleImportBankStatement)))
	mux.Handle("GET /api/v1/finance/bank-imports/{id}", authed(app.perm("finance.read", app.handleGetBankImport)))
	mux.Handle("POST /api/v1/finance/bank-imports/{id}/entries/{entryId}/ignore", authed(app.perm("finance.write", app.handleIgnoreBankEntry)))
	mux.Handle("POST /api/v1/finance/bank-imports/{id}/entries/{entryId}/generate", authed(app.perm("finance.write", app.handleGenerateBankEntry)))
	mux.Handle("DELETE /api/v1/finance/bank-imports/{id}", authed(app.perm("finance.write", app.handleDeleteBankImport)))

	// Download de anexos (arquivo por nome - opaco, nao requer auth)
	mux.Handle("GET /api/v1/attachments/{filename}", http.HandlerFunc(app.handleDownloadAttachment))

	// Documentos digitais (leitura por token do QR)
	mux.Handle("GET /api/v1/documents/by-token/{token}", authed(http.HandlerFunc(app.handleGetDocumentByToken)))
	mux.Handle("GET /api/v1/member-documents/{id}/html", authed(app.perm("members.read", app.handleRenderDocument)))

	// Recibos: renderizacao e envio (prefixo proprio evita ambiguidade de rotas)
	mux.Handle("GET /api/v1/receipts/{id}", authed(app.perm("finance.read", app.handleRenderReceipt)))
	mux.Handle("POST /api/v1/receipts/{id}/send", authed(app.perm("finance.write", app.handleSendDocument)))
	mux.Handle("GET /api/v1/receipts/{id}/deliveries", authed(app.perm("finance.read", app.handleListDeliveries)))

	// Relatorios
	mux.Handle("GET /api/v1/reports/balance", authed(app.perm("reports.read", app.handleMonthlyBalance)))
	mux.Handle("GET /api/v1/reports/dre", authed(app.perm("reports.read", app.handleDRE)))
	mux.Handle("GET /api/v1/reports/balance/export", authed(app.perm("reports.read", app.handleExportBalance)))
	mux.Handle("GET /api/v1/reports/dre/export", authed(app.perm("reports.read", app.handleExportDRE)))
	mux.Handle("GET /api/v1/reports/birthdays", authed(app.perm("reports.read", app.handleBirthdays)))
	mux.Handle("GET /api/v1/reports/contribution-drop", authed(app.perm("reports.read", app.handleContributionDrops)))
	mux.Handle("GET /api/v1/reports/birthdays/export", authed(app.perm("reports.read", app.handleExportBirthdays)))
	mux.Handle("GET /api/v1/reports/demographics", authed(app.perm("reports.read", app.handleDemographics)))
	mux.Handle("GET /api/v1/reports/demographics/export", authed(app.perm("reports.read", app.handleExportDemographics)))
	mux.Handle("GET /api/v1/reports/monthly-statement", authed(app.perm("reports.read", app.handleMonthlyStatement)))
	mux.Handle("GET /api/v1/reports/monthly-statement/export", authed(app.perm("reports.read", app.handleExportMonthlyStatement)))
	mux.Handle("GET /api/v1/reports/consolidated", authed(app.perm("reports.read", app.handleConsolidatedReport)))
	mux.Handle("GET /api/v1/reports/assembly", authed(app.perm("reports.read", app.handleAssemblyReport)))
	mux.Handle("GET /api/v1/reports/assembly/export", authed(app.perm("reports.read", app.handleExportAssembly)))
	mux.Handle("GET /api/v1/reports/attendance", authed(app.perm("reports.read", app.handleAttendanceReport)))
	mux.Handle("GET /api/v1/reports/attendance/export", authed(app.perm("reports.read", app.handleExportAttendance)))

	// Branches (selecao de filial em repasses) + configuracao do tenant
	mux.Handle("GET /api/v1/branches", authed(http.HandlerFunc(app.handleListBranches)))
	mux.Handle("POST /api/v1/branches", authed(app.perm("settings.write", app.handleCreateBranch)))
	mux.Handle("PATCH /api/v1/branches/{id}", authed(app.perm("settings.write", app.handleUpdateBranch)))
	mux.Handle("DELETE /api/v1/branches/{id}", authed(app.perm("settings.write", app.handleDeleteBranch)))

	// Canais de envio por filial (WhatsApp via Evolution + SMTP).
	mux.Handle("GET /api/v1/branches/{id}/channels", authed(app.perm("settings.read", app.handleGetBranchChannels)))
	mux.Handle("PATCH /api/v1/branches/{id}/channels", authed(app.perm("settings.write", app.handleUpdateBranchChannels)))
	mux.Handle("POST /api/v1/branches/{id}/whatsapp/connect", authed(app.perm("settings.write", app.handleConnectBranchWhatsApp)))
	mux.Handle("GET /api/v1/branches/{id}/whatsapp/state", authed(app.perm("settings.read", app.handleBranchWhatsAppState)))
	mux.Handle("POST /api/v1/branches/{id}/whatsapp/logout", authed(app.perm("settings.write", app.handleDisconnectBranchWhatsApp)))
	mux.Handle("GET /api/v1/tenant", authed(app.perm("settings.read", app.handleGetTenant)))
	mux.Handle("PATCH /api/v1/tenant", authed(app.perm("settings.write", app.handleUpdateTenant)))

	// Repasses entre filiais
	mux.Handle("GET /api/v1/finance/transfers", authed(app.perm("finance.read", app.handleListTransfers)))
	mux.Handle("POST /api/v1/finance/transfers", authed(app.perm("finance.write", app.handleCreateTransfer)))

	// Doacoes recorrentes
	mux.Handle("GET /api/v1/finance/recurring", authed(app.perm("finance.read", app.handleListRecurring)))
	mux.Handle("POST /api/v1/finance/recurring", authed(app.perm("finance.write", app.handleCreateRecurring)))
	mux.Handle("PATCH /api/v1/finance/recurring/{id}", authed(app.perm("finance.write", app.handleUpdateRecurring)))

	// Ministerios / escalas
	mux.Handle("GET /api/v1/ministries", authed(app.perm("ministries.read", app.handleListMinistries)))
	mux.Handle("POST /api/v1/ministries", authed(app.perm("ministries.write", app.handleCreateMinistry)))
	mux.Handle("PATCH /api/v1/ministries/{id}", authed(app.perm("ministries.write", app.handleUpdateMinistry)))
	mux.Handle("DELETE /api/v1/ministries/{id}", authed(app.perm("ministries.write", app.handleDeleteMinistry)))
	mux.Handle("GET /api/v1/ministries/{id}/members", authed(app.perm("ministries.read", app.handleListMinistryMembers)))
	mux.Handle("POST /api/v1/ministries/{id}/members", authed(app.perm("ministries.write", app.handleAddMinistryMember)))
	mux.Handle("POST /api/v1/ministries/{id}/members/batch", authed(app.perm("ministries.write", app.handleAddMinistryMembersBatch)))
	mux.Handle("DELETE /api/v1/ministries/{id}/members/{memberId}", authed(app.perm("ministries.write", app.handleRemoveMinistryMember)))

	// Escalas de voluntarios
	mux.Handle("GET /api/v1/rosters", authed(app.perm("ministries.read", app.handleListRosters)))
	mux.Handle("POST /api/v1/rosters", authed(app.perm("ministries.write", app.handleCreateRoster)))
	mux.Handle("GET /api/v1/rosters/suggestions", authed(app.perm("ministries.read", app.handleRosterSuggestions)))
	mux.Handle("GET /api/v1/rosters/{id}", authed(app.perm("ministries.read", app.handleGetRoster)))
	mux.Handle("PATCH /api/v1/rosters/{id}", authed(app.perm("ministries.write", app.handleUpdateRoster)))
	mux.Handle("DELETE /api/v1/rosters/{id}", authed(app.perm("ministries.write", app.handleDeleteRoster)))
	mux.Handle("POST /api/v1/rosters/{id}/assignments", authed(app.perm("ministries.write", app.handleSetRosterAssignments)))
	mux.Handle("PATCH /api/v1/rosters/{id}/assignments/{assignmentId}", authed(app.perm("ministries.write", app.handleRespondRosterAssignment)))
	mux.Handle("GET /api/v1/rosters/{id}/conflicts", authed(app.perm("ministries.read", app.handleRosterConflicts)))

	// Pequenos grupos / celulas + check-in
	mux.Handle("GET /api/v1/groups", authed(app.perm("ministries.read", app.handleListGroups)))
	mux.Handle("POST /api/v1/groups", authed(app.perm("ministries.write", app.handleCreateGroup)))
	mux.Handle("PATCH /api/v1/groups/{id}", authed(app.perm("ministries.write", app.handleUpdateGroup)))
	mux.Handle("DELETE /api/v1/groups/{id}", authed(app.perm("ministries.write", app.handleDeleteGroup)))
	mux.Handle("POST /api/v1/groups/{id}/attendance", authed(app.perm("ministries.write", app.handleCheckIn)))
	mux.Handle("GET /api/v1/groups/{id}/attendance", authed(app.perm("ministries.read", app.handleListEventAttendance)))
	mux.Handle("GET /api/v1/groups/{id}/members", authed(app.perm("ministries.read", app.handleListGroupMembers)))
	mux.Handle("POST /api/v1/groups/{id}/members", authed(app.perm("ministries.write", app.handleAddGroupMember)))
	mux.Handle("DELETE /api/v1/groups/{id}/members/{memberId}", authed(app.perm("ministries.write", app.handleRemoveGroupMember)))

	// ---- Materiais de estudo (arquivo/link) ----
	mux.Handle("GET /api/v1/materials", authed(app.perm("ministries.read", app.handleListMaterials)))
	mux.Handle("POST /api/v1/materials", authed(app.perm("ministries.write", app.handleCreateMaterial)))
	mux.Handle("DELETE /api/v1/materials/{id}", authed(app.perm("ministries.write", app.handleDeleteMaterial)))
	mux.Handle("GET /api/v1/materials/{id}/file", authed(app.perm("ministries.read", app.handleDownloadMaterial)))

	// Avisos (app do membro) - gestao + disparo
	mux.Handle("GET /api/v1/announcements", authed(app.perm("announcements.read", app.handleListAnnouncements)))
	mux.Handle("POST /api/v1/announcements", authed(app.perm("announcements.write", app.handleCreateAnnouncement)))
	mux.Handle("PATCH /api/v1/announcements/{id}", authed(app.perm("announcements.write", app.handleUpdateAnnouncement)))
	mux.Handle("DELETE /api/v1/announcements/{id}", authed(app.perm("announcements.write", app.handleDeleteAnnouncement)))
	mux.Handle("POST /api/v1/announcements/audience/preview", authed(app.perm("announcements.read", app.handlePreviewAudience)))
	mux.Handle("POST /api/v1/announcements/{id}/send", authed(app.perm("announcements.write", app.handleSendAnnouncement)))
	mux.Handle("GET /api/v1/announcements/{id}/deliveries", authed(app.perm("announcements.read", app.handleListAnnouncementDeliveries)))
	mux.Handle("POST /api/v1/announcements/send-test", authed(app.perm("announcements.write", app.handleSendTestMessage)))

	// Historico de disparos agendados (tela Execucoes)
	mux.Handle("GET /api/v1/notification-runs", authed(app.perm("announcements.read", app.handleListNotificationRuns)))

	// Automacoes de WhatsApp (#31): aniversario, lembrete de escala e boas-vindas
	mux.Handle("GET /api/v1/notifications/settings", authed(app.perm("announcements.read", app.handleGetNotificationSettings)))
	mux.Handle("PATCH /api/v1/notifications/settings", authed(app.perm("announcements.write", app.handleUpdateNotificationSettings)))
	mux.Handle("POST /api/v1/notifications/run", authed(app.perm("announcements.write", app.handleRunNotifications)))

	// Usuarios e acessos (autorizacao por papel no handler; RLS isola o tenant)
	mux.Handle("GET /api/v1/users", authed(app.perm("users.read", app.handleListUsers)))
	mux.Handle("POST /api/v1/users", authed(app.perm("users.write", app.handleCreateUser)))
	mux.Handle("PATCH /api/v1/users/{id}", authed(app.perm("users.write", app.handleUpdateUser)))
	mux.Handle("POST /api/v1/users/{id}/password", authed(app.perm("users.write", app.handleResetUserPassword)))
	mux.Handle("GET /api/v1/roles", authed(app.perm("users.read", app.handleListRoles)))
	mux.Handle("GET /api/v1/permissions", authed(app.perm("users.read", app.handleListPermissions)))

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
	mux.Handle("GET /api/v1/event-kinds", authed(app.perm("members.read", app.handleListEventKinds)))
	mux.Handle("GET /api/v1/events", authed(app.perm("members.read", app.handleListEvents)))
	mux.Handle("POST /api/v1/events", authed(app.perm("members.write", app.handleCreateEvent)))
	mux.Handle("GET /api/v1/events/{id}", authed(app.perm("members.read", app.handleGetEvent)))
	mux.Handle("PATCH /api/v1/events/{id}", authed(app.perm("members.write", app.handleUpdateEvent)))
	mux.Handle("DELETE /api/v1/events/{id}", authed(app.perm("members.write", app.handleDeleteEvent)))
	mux.Handle("GET /api/v1/events/{id}/attendance", authed(app.perm("members.read", app.handleListEventAttendance)))
	mux.Handle("POST /api/v1/events/{id}/attendance", authed(app.perm("members.write", app.handleSaveEventAttendance)))
	mux.Handle("GET /api/v1/events/{id}/invitees", authed(app.perm("members.read", app.handleListInvitees)))
	mux.Handle("POST /api/v1/events/{id}/invitees", authed(app.perm("members.write", app.handleSetInvitees)))
	mux.Handle("GET /api/v1/members/{id}/frequency", authed(app.perm("members.read", app.handleListFrequency)))
	mux.Handle("POST /api/v1/members/{id}/frequency", authed(app.perm("members.write", app.handleSetFrequency)))

	// Programacao: grade de horarios recorrentes e publicacao na agenda
	mux.Handle("GET /api/v1/programacoes", authed(app.perm("members.read", app.handleListProgramacao)))
	mux.Handle("POST /api/v1/programacoes", authed(app.perm("members.write", app.handleCreateProgramacao)))
	mux.Handle("POST /api/v1/programacoes/generate", authed(app.perm("members.write", app.handleGenerateProgramacao)))
	mux.Handle("PATCH /api/v1/programacoes/{id}", authed(app.perm("members.write", app.handleUpdateProgramacao)))
	mux.Handle("DELETE /api/v1/programacoes/{id}", authed(app.perm("members.write", app.handleDeleteProgramacao)))

	// LGPD: consentimento, portabilidade e anonimizacao
	mux.Handle("GET /api/v1/consent-terms", authed(app.perm("members.read", app.handleListConsentTerms)))
	mux.Handle("POST /api/v1/consent-terms", authed(app.perm("members.write", app.handleCreateConsentTerm)))
	mux.Handle("GET /api/v1/members/{id}/consents", authed(app.perm("members.read", app.handleListMemberConsents)))
	mux.Handle("POST /api/v1/members/{id}/consents", authed(app.perm("members.write", app.handleRecordConsent)))
	mux.Handle("GET /api/v1/members/{id}/export", authed(app.perm("members.read", app.handleExportMemberData)))
	mux.Handle("POST /api/v1/members/{id}/anonymize", authed(app.adminOnly(app.handleAnonymizeMember)))

	// Governanca (Modulo 6): atas, assinatura interna, votacao e convenios
	mux.Handle("GET /api/v1/minutes", authed(app.perm("governance.read", app.handleListMinutes)))
	mux.Handle("POST /api/v1/minutes", authed(app.perm("governance.write", app.handleCreateMinute)))
	mux.Handle("GET /api/v1/minutes/{id}", authed(app.perm("governance.read", app.handleGetMinute)))
	mux.Handle("PATCH /api/v1/minutes/{id}", authed(app.perm("governance.write", app.handleUpdateMinute)))
	mux.Handle("DELETE /api/v1/minutes/{id}", authed(app.perm("governance.write", app.handleDeleteMinute)))
	mux.Handle("POST /api/v1/minutes/{id}/sign", authed(app.perm("governance.write", app.handleSignMinute)))
	mux.Handle("GET /api/v1/minutes/{id}/signatures", authed(app.perm("governance.read", app.handleListSignatures)))
	mux.Handle("GET /api/v1/votes", authed(app.perm("governance.read", app.handleListVotes)))
	mux.Handle("POST /api/v1/votes", authed(app.perm("governance.write", app.handleCreateVote)))
	mux.Handle("GET /api/v1/votes/{id}", authed(app.perm("governance.read", app.handleGetVote)))
	mux.Handle("PATCH /api/v1/votes/{id}", authed(app.perm("governance.write", app.handleUpdateVote)))
	mux.Handle("DELETE /api/v1/votes/{id}", authed(app.perm("governance.write", app.handleDeleteVote)))
	mux.Handle("POST /api/v1/votes/{id}/open", authed(app.perm("governance.write", app.handleOpenVote)))
	mux.Handle("POST /api/v1/votes/{id}/close", authed(app.perm("governance.write", app.handleCloseVote)))
	mux.Handle("POST /api/v1/votes/{id}/ballot", authed(app.perm("governance.read", app.handleCastBallot)))
	mux.Handle("GET /api/v1/votes/{id}/result", authed(app.perm("governance.read", app.handleVoteResult)))
	mux.Handle("GET /api/v1/legal-documents", authed(app.perm("governance.read", app.handleListLegalDocuments)))
	mux.Handle("POST /api/v1/legal-documents", authed(app.perm("governance.write", app.handleCreateLegalDocument)))
	mux.Handle("PATCH /api/v1/legal-documents/{id}", authed(app.perm("governance.write", app.handleUpdateLegalDocument)))
	mux.Handle("DELETE /api/v1/legal-documents/{id}", authed(app.perm("governance.write", app.handleDeleteLegalDocument)))
	mux.Handle("GET /api/v1/governance/mandates", authed(app.perm("governance.read", app.handleListMandates)))

	// Ministerio Infantil (Kids): trilha/conteudo, turmas, participantes,
	// encontros, check-in e evolucao.
	mux.Handle("GET /api/v1/kids/tracks", authed(app.perm("members.read", app.handleListKidsTracks)))
	mux.Handle("POST /api/v1/kids/tracks", authed(app.perm("members.write", app.handleCreateKidsTrack)))
	mux.Handle("PATCH /api/v1/kids/tracks/{id}", authed(app.perm("members.write", app.handleUpdateKidsTrack)))
	mux.Handle("DELETE /api/v1/kids/tracks/{id}", authed(app.perm("members.write", app.handleDeleteKidsTrack)))
	mux.Handle("GET /api/v1/kids/tracks/{id}/lessons", authed(app.perm("members.read", app.handleListKidsLessons)))
	mux.Handle("POST /api/v1/kids/tracks/{id}/lessons", authed(app.perm("members.write", app.handleCreateKidsLesson)))
	mux.Handle("PATCH /api/v1/kids/lessons/{id}", authed(app.perm("members.write", app.handleUpdateKidsLesson)))
	mux.Handle("DELETE /api/v1/kids/lessons/{id}", authed(app.perm("members.write", app.handleDeleteKidsLesson)))
	mux.Handle("GET /api/v1/kids/classes", authed(app.perm("members.read", app.handleListKidsClasses)))
	mux.Handle("POST /api/v1/kids/classes", authed(app.perm("members.write", app.handleCreateKidsClass)))
	mux.Handle("PATCH /api/v1/kids/classes/{id}", authed(app.perm("members.write", app.handleUpdateKidsClass)))
	mux.Handle("DELETE /api/v1/kids/classes/{id}", authed(app.perm("members.write", app.handleDeleteKidsClass)))
	mux.Handle("GET /api/v1/kids/classes/{id}/enrollments", authed(app.perm("members.read", app.handleListKidsEnrollments)))
	mux.Handle("POST /api/v1/kids/classes/{id}/enrollments", authed(app.perm("members.write", app.handleCreateKidsEnrollment)))
	mux.Handle("PATCH /api/v1/kids/enrollments/{id}", authed(app.perm("members.write", app.handleUpdateKidsEnrollment)))
	mux.Handle("DELETE /api/v1/kids/enrollments/{id}", authed(app.perm("members.write", app.handleDeleteKidsEnrollment)))
	mux.Handle("GET /api/v1/kids/enrollments/{id}/guardians", authed(app.perm("members.read", app.handleListKidsGuardians)))
	mux.Handle("POST /api/v1/kids/enrollments/{id}/guardians", authed(app.perm("members.write", app.handleAddKidsGuardian)))
	mux.Handle("DELETE /api/v1/kids/guardians/{id}", authed(app.perm("members.write", app.handleDeleteKidsGuardian)))
	mux.Handle("GET /api/v1/kids/sessions", authed(app.perm("members.read", app.handleListKidsSessions)))
	mux.Handle("POST /api/v1/kids/sessions", authed(app.perm("members.write", app.handleCreateKidsSession)))
	mux.Handle("PATCH /api/v1/kids/sessions/{id}", authed(app.perm("members.write", app.handleUpdateKidsSession)))
	mux.Handle("DELETE /api/v1/kids/sessions/{id}", authed(app.perm("members.write", app.handleDeleteKidsSession)))
	mux.Handle("GET /api/v1/kids/sessions/{id}/roster", authed(app.perm("members.read", app.handleKidsRoster)))
	mux.Handle("POST /api/v1/kids/sessions/{id}/checkin", authed(app.perm("members.write", app.handleKidsCheckin)))
	mux.Handle("POST /api/v1/kids/sessions/{id}/checkout", authed(app.perm("members.write", app.handleKidsCheckout)))
	mux.Handle("POST /api/v1/kids/sessions/{id}/absence", authed(app.perm("members.write", app.handleKidsAbsence)))
	mux.Handle("GET /api/v1/kids/classes/{id}/evolution", authed(app.perm("members.read", app.handleKidsEvolution)))

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
