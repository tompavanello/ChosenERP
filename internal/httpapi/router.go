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
	"chosenerp/internal/members"
	"chosenerp/internal/ministries"
	"chosenerp/internal/org"
	"chosenerp/internal/rosters"
	"chosenerp/internal/store"
	"chosenerp/internal/suppliers"
	"chosenerp/internal/users"
	"chosenerp/internal/visitors"
)

// App agrega as dependências dos handlers HTTP.
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
	Users         *users.Repo
	Events        *events.Repo
	LGPD          *lgpd.Repo
	Governance    *governance.Repo
	Org           *org.Repo
	Rosters       *rosters.Repo
	Kids          *kids.Repo
	Dispatch      *delivery.Dispatcher
	// Limitador protege as rotas /api/v1/public/* (sem auth).
	Limitador *limitadorPublico
}

// NewRouter monta o gateway HTTP e suas rotas.
func NewRouter(cfg *Config, st *store.Store, authSvc *auth.Service, membersRepo *members.Repo, finRepo *finance.Repo, auditRepo *audit.Repo, docRepo *documents.Repo, famRepo *families.Repo, visitRepo *visitors.Repo, benefRepo *benefactors.Repo, suppliersRepo *suppliers.Repo, ministRepo *ministries.Repo, groupsRepo *groups.Repo, annRepo *announcements.Repo, cargosRepo *cargos.Repo, usersRepo *users.Repo, eventsRepo *events.Repo, lgpdRepo *lgpd.Repo, govRepo *governance.Repo, orgRepo *org.Repo, rostersRepo *rosters.Repo, kidsRepo *kids.Repo, dispatcher *delivery.Dispatcher) http.Handler {
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
		Users:         usersRepo,
		Events:        eventsRepo,
		LGPD:          lgpdRepo,
		Governance:    govRepo,
		Org:           orgRepo,
		Rosters:       rostersRepo,
		Kids:          kidsRepo,
		Dispatch:      dispatcher,
		Limitador:     newLimitadorPublico(),
	}
	mux := http.NewServeMux()

	// Público
	mux.HandleFunc("GET /healthz", app.handleHealth)
	mux.HandleFunc("GET /metrics", app.handleMetrics)
	mux.HandleFunc("POST /api/v1/auth/login", app.handleLogin)
	mux.HandleFunc("POST /api/v1/auth/refresh", app.handleRefresh)
	mux.HandleFunc("POST /api/v1/auth/select-tenant", app.handleSelectTenant)
	// Branding da igreja por slug (tela de login do subdomínio).
	mux.Handle("GET /api/v1/public/tenant/{slug}", app.publico(app.handlePublicTenant))

	// Autenticado
	authed := app.Authenticator
	mux.Handle("GET /api/v1/me", authed(http.HandlerFunc(app.handleMe)))
	mux.Handle("GET /api/v1/me/tenants", authed(http.HandlerFunc(app.handleMeTenants)))
	mux.Handle("PATCH /api/v1/me", authed(http.HandlerFunc(app.handleUpdateProfile)))
	mux.Handle("POST /api/v1/me/password", authed(http.HandlerFunc(app.handleChangePassword)))
	mux.Handle("POST /api/v1/auth/switch-tenant", authed(http.HandlerFunc(app.handleSwitchTenant)))
	mux.Handle("GET /api/v1/members", authed(http.HandlerFunc(app.handleListMembers)))
	mux.Handle("POST /api/v1/members", authed(http.HandlerFunc(app.handleCreateMember)))
	mux.Handle("GET /api/v1/members/{id}", authed(http.HandlerFunc(app.handleGetMember)))
	mux.Handle("PATCH /api/v1/members/{id}", authed(http.HandlerFunc(app.handleUpdateMember)))
	mux.Handle("GET /api/v1/members/{id}/tree", authed(http.HandlerFunc(app.handleMemberTree)))
	mux.Handle("POST /api/v1/members/{id}/relationships", authed(http.HandlerFunc(app.handleAddRelationship)))
	mux.Handle("POST /api/v1/members/{id}/card", authed(http.HandlerFunc(app.handleIssueCard)))
	mux.Handle("GET /api/v1/members/{id}/card", authed(http.HandlerFunc(app.handleGetCard)))
	mux.Handle("POST /api/v1/members/{id}/photo", authed(http.HandlerFunc(app.handleUploadMemberPhoto)))
	mux.Handle("DELETE /api/v1/members/{id}/photo", authed(http.HandlerFunc(app.handleDeleteMemberPhoto)))
	mux.Handle("GET /api/v1/members/{id}/families", authed(http.HandlerFunc(app.handleMemberFamilies)))

	// Histórico eclesiástico do membro (requisito 1.8)
	mux.Handle("GET /api/v1/members/{id}/history", authed(http.HandlerFunc(app.handleListMemberHistory)))
	mux.Handle("POST /api/v1/members/{id}/history", authed(http.HandlerFunc(app.handleAddMemberHistory)))

	// Cargos (funções/ministérios) e mandatos do membro
	mux.Handle("GET /api/v1/cargos", authed(http.HandlerFunc(app.handleListCargos)))
	mux.Handle("POST /api/v1/cargos", authed(http.HandlerFunc(app.handleCreateCargo)))
	mux.Handle("PATCH /api/v1/cargos/{id}", authed(http.HandlerFunc(app.handleUpdateCargo)))
	mux.Handle("DELETE /api/v1/cargos/{id}", authed(http.HandlerFunc(app.handleDeleteCargo)))
	mux.Handle("GET /api/v1/members/{id}/cargos", authed(http.HandlerFunc(app.handleListMemberCargos)))
	mux.Handle("POST /api/v1/members/{id}/cargos", authed(http.HandlerFunc(app.handleAssignCargo)))
	mux.Handle("PATCH /api/v1/members/{id}/cargos/{linkId}", authed(http.HandlerFunc(app.handleUpdateMemberCargo)))
	mux.Handle("DELETE /api/v1/members/{id}/cargos/{linkId}", authed(http.HandlerFunc(app.handleUnassignCargo)))

	// Famílias
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
	mux.Handle("GET /api/v1/finance/accounts", authed(http.HandlerFunc(app.handleListAccounts)))
	mux.Handle("POST /api/v1/finance/accounts", authed(http.HandlerFunc(app.handleCreateAccount)))
	mux.Handle("PATCH /api/v1/finance/accounts/{id}", authed(http.HandlerFunc(app.handleUpdateAccount)))
	mux.Handle("DELETE /api/v1/finance/accounts/{id}", authed(http.HandlerFunc(app.handleDeleteAccount)))
	mux.Handle("GET /api/v1/finance/transactions", authed(http.HandlerFunc(app.handleListTxn)))
	mux.Handle("POST /api/v1/finance/transactions", authed(http.HandlerFunc(app.handleCreateTxn)))
	mux.Handle("POST /api/v1/finance/transactions/import", authed(http.HandlerFunc(app.handleImportTransactions)))
	mux.Handle("POST /api/v1/finance/transactions/import/preview", authed(http.HandlerFunc(app.handlePreviewTransactions)))
	mux.Handle("POST /api/v1/finance/transactions/{id}/void", authed(http.HandlerFunc(app.handleVoidTxn)))
	mux.Handle("GET /api/v1/finance/transactions/{id}/events", authed(http.HandlerFunc(app.handleListTxnEvents)))
	mux.Handle("GET /api/v1/finance/transactions/{id}/attachments", authed(http.HandlerFunc(app.handleListAttachments)))
	mux.Handle("POST /api/v1/finance/transactions/{id}/attachments", authed(http.HandlerFunc(app.handleUploadAttachment)))
	mux.Handle("GET /api/v1/finance/balance", authed(http.HandlerFunc(app.handleBalance)))

	// Auditoria analítica do financeiro (documento imutável após fechamento)
	mux.Handle("GET /api/v1/finance/audits", authed(http.HandlerFunc(app.handleListAudits)))
	mux.Handle("POST /api/v1/finance/audits", authed(http.HandlerFunc(app.handleCreateAudit)))
	mux.Handle("GET /api/v1/finance/audits/{id}", authed(http.HandlerFunc(app.handleGetAudit)))
	mux.Handle("POST /api/v1/finance/audits/{id}/mark", authed(http.HandlerFunc(app.handleMarkAudit)))
	mux.Handle("POST /api/v1/finance/audits/{id}/close", authed(http.HandlerFunc(app.handleCloseAudit)))
	mux.Handle("GET /api/v1/finance/audits/{id}/export", authed(http.HandlerFunc(app.handleExportAudit)))
	mux.Handle("DELETE /api/v1/finance/audits/{id}", authed(http.HandlerFunc(app.handleDeleteAudit)))

	// Download de anexos (arquivo por nome — opaco, não requer auth)
	mux.Handle("GET /api/v1/attachments/{filename}", http.HandlerFunc(app.handleDownloadAttachment))

	// Documentos digitais (leitura por token do QR)
	mux.Handle("GET /api/v1/documents/by-token/{token}", authed(http.HandlerFunc(app.handleGetDocumentByToken)))

	// Recibos: renderização e envio (prefixo próprio evita ambiguidade de rotas)
	mux.Handle("GET /api/v1/receipts/{id}", authed(http.HandlerFunc(app.handleRenderReceipt)))
	mux.Handle("POST /api/v1/receipts/{id}/send", authed(http.HandlerFunc(app.handleSendDocument)))
	mux.Handle("GET /api/v1/receipts/{id}/deliveries", authed(http.HandlerFunc(app.handleListDeliveries)))

	// Relatórios
	mux.Handle("GET /api/v1/reports/balance", authed(http.HandlerFunc(app.handleMonthlyBalance)))
	mux.Handle("GET /api/v1/reports/dre", authed(http.HandlerFunc(app.handleDRE)))
	mux.Handle("GET /api/v1/reports/balance/export", authed(http.HandlerFunc(app.handleExportBalance)))
	mux.Handle("GET /api/v1/reports/dre/export", authed(http.HandlerFunc(app.handleExportDRE)))
	mux.Handle("GET /api/v1/reports/birthdays", authed(http.HandlerFunc(app.handleBirthdays)))
	mux.Handle("GET /api/v1/reports/birthdays/export", authed(http.HandlerFunc(app.handleExportBirthdays)))
	mux.Handle("GET /api/v1/reports/demographics", authed(http.HandlerFunc(app.handleDemographics)))
	mux.Handle("GET /api/v1/reports/demographics/export", authed(http.HandlerFunc(app.handleExportDemographics)))
	mux.Handle("GET /api/v1/reports/monthly-statement", authed(http.HandlerFunc(app.handleMonthlyStatement)))
	mux.Handle("GET /api/v1/reports/monthly-statement/export", authed(http.HandlerFunc(app.handleExportMonthlyStatement)))
	mux.Handle("GET /api/v1/reports/consolidated", authed(http.HandlerFunc(app.handleConsolidatedReport)))
	mux.Handle("GET /api/v1/reports/assembly", authed(http.HandlerFunc(app.handleAssemblyReport)))
	mux.Handle("GET /api/v1/reports/assembly/export", authed(http.HandlerFunc(app.handleExportAssembly)))

	// Branches (seleção de filial em repasses) + configuração do tenant
	mux.Handle("GET /api/v1/branches", authed(http.HandlerFunc(app.handleListBranches)))
	mux.Handle("POST /api/v1/branches", authed(http.HandlerFunc(app.handleCreateBranch)))
	mux.Handle("PATCH /api/v1/branches/{id}", authed(http.HandlerFunc(app.handleUpdateBranch)))
	mux.Handle("DELETE /api/v1/branches/{id}", authed(http.HandlerFunc(app.handleDeleteBranch)))
	mux.Handle("GET /api/v1/tenant", authed(http.HandlerFunc(app.handleGetTenant)))
	mux.Handle("PATCH /api/v1/tenant", authed(http.HandlerFunc(app.handleUpdateTenant)))

	// Repasses entre filiais
	mux.Handle("GET /api/v1/finance/transfers", authed(http.HandlerFunc(app.handleListTransfers)))
	mux.Handle("POST /api/v1/finance/transfers", authed(http.HandlerFunc(app.handleCreateTransfer)))

	// Doações recorrentes
	mux.Handle("GET /api/v1/finance/recurring", authed(http.HandlerFunc(app.handleListRecurring)))
	mux.Handle("POST /api/v1/finance/recurring", authed(http.HandlerFunc(app.handleCreateRecurring)))
	mux.Handle("PATCH /api/v1/finance/recurring/{id}", authed(http.HandlerFunc(app.handleUpdateRecurring)))

	// Ministérios / escalas
	mux.Handle("GET /api/v1/ministries", authed(http.HandlerFunc(app.handleListMinistries)))
	mux.Handle("POST /api/v1/ministries", authed(http.HandlerFunc(app.handleCreateMinistry)))
	mux.Handle("PATCH /api/v1/ministries/{id}", authed(http.HandlerFunc(app.handleUpdateMinistry)))
	mux.Handle("DELETE /api/v1/ministries/{id}", authed(http.HandlerFunc(app.handleDeleteMinistry)))
	mux.Handle("GET /api/v1/ministries/{id}/members", authed(http.HandlerFunc(app.handleListMinistryMembers)))
	mux.Handle("POST /api/v1/ministries/{id}/members", authed(http.HandlerFunc(app.handleAddMinistryMember)))
	mux.Handle("POST /api/v1/ministries/{id}/members/batch", authed(http.HandlerFunc(app.handleAddMinistryMembersBatch)))
	mux.Handle("DELETE /api/v1/ministries/{id}/members/{memberId}", authed(http.HandlerFunc(app.handleRemoveMinistryMember)))

	// Escalas de voluntários
	mux.Handle("GET /api/v1/rosters", authed(http.HandlerFunc(app.handleListRosters)))
	mux.Handle("POST /api/v1/rosters", authed(http.HandlerFunc(app.handleCreateRoster)))
	mux.Handle("GET /api/v1/rosters/suggestions", authed(http.HandlerFunc(app.handleRosterSuggestions)))
	mux.Handle("GET /api/v1/rosters/{id}", authed(http.HandlerFunc(app.handleGetRoster)))
	mux.Handle("PATCH /api/v1/rosters/{id}", authed(http.HandlerFunc(app.handleUpdateRoster)))
	mux.Handle("DELETE /api/v1/rosters/{id}", authed(http.HandlerFunc(app.handleDeleteRoster)))
	mux.Handle("POST /api/v1/rosters/{id}/assignments", authed(http.HandlerFunc(app.handleSetRosterAssignments)))
	mux.Handle("PATCH /api/v1/rosters/{id}/assignments/{assignmentId}", authed(http.HandlerFunc(app.handleRespondRosterAssignment)))
	mux.Handle("GET /api/v1/rosters/{id}/conflicts", authed(http.HandlerFunc(app.handleRosterConflicts)))

	// Pequenos grupos / células + check-in
	mux.Handle("GET /api/v1/groups", authed(http.HandlerFunc(app.handleListGroups)))
	mux.Handle("POST /api/v1/groups", authed(http.HandlerFunc(app.handleCreateGroup)))
	mux.Handle("PATCH /api/v1/groups/{id}", authed(http.HandlerFunc(app.handleUpdateGroup)))
	mux.Handle("DELETE /api/v1/groups/{id}", authed(http.HandlerFunc(app.handleDeleteGroup)))
	mux.Handle("POST /api/v1/groups/{id}/attendance", authed(http.HandlerFunc(app.handleCheckIn)))
	mux.Handle("GET /api/v1/groups/{id}/attendance", authed(http.HandlerFunc(app.handleListEventAttendance)))

	// Avisos (app do membro) — gestão + disparo
	mux.Handle("GET /api/v1/announcements", authed(http.HandlerFunc(app.handleListAnnouncements)))
	mux.Handle("POST /api/v1/announcements", authed(http.HandlerFunc(app.handleCreateAnnouncement)))
	mux.Handle("PATCH /api/v1/announcements/{id}", authed(http.HandlerFunc(app.handleUpdateAnnouncement)))
	mux.Handle("DELETE /api/v1/announcements/{id}", authed(http.HandlerFunc(app.handleDeleteAnnouncement)))
	mux.Handle("POST /api/v1/announcements/audience/preview", authed(http.HandlerFunc(app.handlePreviewAudience)))
	mux.Handle("POST /api/v1/announcements/{id}/send", authed(http.HandlerFunc(app.handleSendAnnouncement)))
	mux.Handle("GET /api/v1/announcements/{id}/deliveries", authed(http.HandlerFunc(app.handleListAnnouncementDeliveries)))
	mux.Handle("POST /api/v1/announcements/send-test", authed(http.HandlerFunc(app.handleSendTestMessage)))

	// Histórico de disparos agendados (tela Execuções)
	mux.Handle("GET /api/v1/notification-runs", authed(http.HandlerFunc(app.handleListNotificationRuns)))

	// Automações de WhatsApp (#31): aniversário, lembrete de escala e boas-vindas
	mux.Handle("GET /api/v1/notifications/settings", authed(http.HandlerFunc(app.handleGetNotificationSettings)))
	mux.Handle("PATCH /api/v1/notifications/settings", authed(http.HandlerFunc(app.handleUpdateNotificationSettings)))
	mux.Handle("POST /api/v1/notifications/run", authed(http.HandlerFunc(app.handleRunNotifications)))

	// Usuários e acessos (autorização por papel no handler; RLS isola o tenant)
	mux.Handle("GET /api/v1/users", authed(http.HandlerFunc(app.handleListUsers)))
	mux.Handle("POST /api/v1/users", authed(http.HandlerFunc(app.handleCreateUser)))
	mux.Handle("PATCH /api/v1/users/{id}", authed(http.HandlerFunc(app.handleUpdateUser)))
	mux.Handle("POST /api/v1/users/{id}/password", authed(http.HandlerFunc(app.handleResetUserPassword)))
	mux.Handle("GET /api/v1/roles", authed(http.HandlerFunc(app.handleListRoles)))
	mux.Handle("GET /api/v1/permissions", authed(http.HandlerFunc(app.handleListPermissions)))

	// MFA (TOTP) do próprio usuário
	mux.Handle("GET /api/v1/auth/mfa", authed(http.HandlerFunc(app.handleMFAStatus)))
	mux.Handle("POST /api/v1/auth/mfa/setup", authed(http.HandlerFunc(app.handleMFASetup)))
	mux.Handle("POST /api/v1/auth/mfa/enable", authed(http.HandlerFunc(app.handleMFAEnable)))
	mux.Handle("POST /api/v1/auth/mfa/disable", authed(http.HandlerFunc(app.handleMFADisable)))

	// Eventos, tipos de evento, chamada nominal e frequência do membro
	mux.Handle("GET /api/v1/event-kinds", authed(http.HandlerFunc(app.handleListEventKinds)))
	mux.Handle("POST /api/v1/event-kinds", authed(http.HandlerFunc(app.handleCreateEventKind)))
	mux.Handle("PATCH /api/v1/event-kinds/{id}", authed(http.HandlerFunc(app.handleUpdateEventKind)))
	mux.Handle("DELETE /api/v1/event-kinds/{id}", authed(http.HandlerFunc(app.handleDeleteEventKind)))
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

	// LGPD: consentimento, portabilidade e anonimização
	mux.Handle("GET /api/v1/consent-terms", authed(http.HandlerFunc(app.handleListConsentTerms)))
	mux.Handle("POST /api/v1/consent-terms", authed(http.HandlerFunc(app.handleCreateConsentTerm)))
	mux.Handle("GET /api/v1/members/{id}/consents", authed(http.HandlerFunc(app.handleListMemberConsents)))
	mux.Handle("POST /api/v1/members/{id}/consents", authed(http.HandlerFunc(app.handleRecordConsent)))
	mux.Handle("GET /api/v1/members/{id}/export", authed(http.HandlerFunc(app.handleExportMemberData)))
	mux.Handle("POST /api/v1/members/{id}/anonymize", authed(http.HandlerFunc(app.handleAnonymizeMember)))

	// Governança (Módulo 6): atas, assinatura interna, votação e convênios
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

	// Ministério Infantil (Kids): trilha/conteúdo, turmas, participantes,
	// encontros, check-in e evolução.
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

	// App do membro (público por token do QR). As três rotas são as únicas sem
	// auth: passam pelo `publico`, que aplica noindex e limite por IP. A foto
	// tem endpoint próprio para o Access poder liberar só ela, e não
	// /api/v1/attachments/{arquivo}, que serve também comprovante financeiro.
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
