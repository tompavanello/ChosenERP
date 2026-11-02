package httpapi

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5"

	"chosenerp/internal/ai"
	"chosenerp/internal/auth"
	"chosenerp/internal/help"
	"chosenerp/internal/org"
)

// helpAIClient usa a chave/modelo dedicados da ajuda e, quando ausentes, cai
// para a IA global (devocionais).
func helpAIClient(cfg *Config) *ai.Client {
	if strings.TrimSpace(cfg.HelpAIAPIKey) != "" {
		return ai.NewClient(cfg.HelpAIBaseURL, cfg.HelpAIAPIKey, cfg.HelpAIModel)
	}
	return ai.NewClient(cfg.AIBaseURL, cfg.AIAPIKey, cfg.AIModel)
}

func helpApp(v string) string {
	if strings.EqualFold(strings.TrimSpace(v), "member") {
		return "member"
	}
	return "webadmin"
}

func helpAccess(ctx context.Context, a *App, claims *auth.Claims, app, path string) help.Access {
	acc := help.Access{
		App:           app,
		Role:          claims.Role,
		Path:          path,
		PlatformAdmin: claims.Role == "platform_admin",
	}
	if claims.TenantID == "" {
		return acc
	}
	perms, err := a.helpPermissions(ctx, claims)
	if err == nil && perms != nil {
		acc.CanPerm = func(p string) bool { return perms[p] }
	}
	if ent, err := a.helpEntitlements(ctx, claims); err == nil && ent != nil {
		acc.HasFeature = func(f string) bool { return ent.Has(f) }
	}
	return acc
}

// helpPermissions carrega as permissoes do papel em uma unica consulta. A Sede
// tem todas (nil); papeis sem permissao recebem mapa vazio.
func (a *App) helpPermissions(ctx context.Context, claims *auth.Claims) (map[string]bool, error) {
	if isAdmin(claims.Role) {
		return nil, nil
	}
	perms := map[string]bool{}
	err := a.Store.WithTenant(ctx, boundsFromClaims(claims), func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `
			SELECT p.key
			FROM role_permissions rp
			JOIN roles r ON r.id = rp.role_id
			JOIN permissions p ON p.id = rp.permission_id
			WHERE r.tenant_id = $1::uuid AND r.key = $2`, claims.TenantID, claims.Role)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var k string
			if err := rows.Scan(&k); err != nil {
				return err
			}
			perms[k] = true
		}
		return rows.Err()
	})
	return perms, err
}

// helpEntitlements usa o mesmo cache do gate de plano.
func (a *App) helpEntitlements(ctx context.Context, claims *auth.Claims) (*org.Entitlements, error) {
	if ent, ok := a.Entitlements.Get(claims.TenantID); ok {
		return ent, nil
	}
	var ent *org.Entitlements
	err := a.Store.WithTenant(ctx, boundsFromClaims(claims), func(tx pgx.Tx) error {
		var e error
		ent, e = a.Org.EffectiveForTenant(ctx, tx, claims.TenantID)
		return e
	})
	if err != nil {
		return nil, err
	}
	a.Entitlements.Set(claims.TenantID, ent)
	return ent, nil
}

func writeHelpError(w http.ResponseWriter, err error) {
	if errors.Is(err, help.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "artigo nao encontrado")
		return
	}
	writeServerError(w, err)
}

func (a *App) handleHelpArticles(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	app := helpApp(r.URL.Query().Get("app"))
	acc := helpAccess(r.Context(), a, claims, app, r.URL.Query().Get("path"))
	items := a.Help.Articles(acc, r.URL.Query().Get("q"), r.URL.Query().Get("category"))
	writeJSON(w, http.StatusOK, map[string]any{
		"articles":   items,
		"categories": a.Help.Categories(acc),
		"ai_enabled": a.HelpAI != nil && a.HelpAI.Enabled(),
	})
}

func (a *App) handleHelpCategories(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	app := helpApp(r.URL.Query().Get("app"))
	acc := helpAccess(r.Context(), a, claims, app, "")
	writeJSON(w, http.StatusOK, map[string]any{"categories": a.Help.Categories(acc)})
}

func (a *App) handleHelpContext(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	app := helpApp(r.URL.Query().Get("app"))
	path := strings.TrimSpace(r.URL.Query().Get("path"))
	if path == "" {
		if app == "member" {
			path = "/home"
		} else {
			path = "/dashboard"
		}
	}
	acc := helpAccess(r.Context(), a, claims, app, path)
	ctx := a.Help.Context(acc)
	writeJSON(w, http.StatusOK, map[string]any{
		"context":    ctx,
		"ai_enabled": a.HelpAI != nil && a.HelpAI.Enabled(),
	})
}

func (a *App) handleHelpArticle(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	app := helpApp(r.URL.Query().Get("app"))
	acc := helpAccess(r.Context(), a, claims, app, "")
	art, err := a.Help.Get(acc, r.PathValue("slug"))
	if err != nil {
		writeHelpError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, art)
}

type helpChatIn struct {
	App      string       `json:"app"`
	Path     string       `json:"path"`
	Messages []ai.Message `json:"messages"`
}

func (a *App) handleHelpChat(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	if a.HelpAI == nil || !a.HelpAI.Enabled() {
		writeErr(w, http.StatusServiceUnavailable, "ai_disabled")
		return
	}
	// Limite por identidade (nao por IP): evita abuso/custo de IA.
	if a.Limitador != nil && !a.Limitador.permitir("help-ai:"+claims.UserID) {
		writeErr(w, http.StatusTooManyRequests, "muitas perguntas em pouco tempo")
		return
	}
	var in helpChatIn
	if err := readJSONMax(r, &in, 64<<10); err != nil {
		writeErr(w, http.StatusBadRequest, "payload invalido")
		return
	}
	app := helpApp(in.App)
	path := strings.TrimSpace(in.Path)
	if path == "" {
		if app == "member" {
			path = "/home"
		} else {
			path = "/dashboard"
		}
	}
	msgs := sanitizeChat(in.Messages)
	if len(msgs) == 0 {
		writeErr(w, http.StatusBadRequest, "mensagem obrigatoria")
		return
	}
	question := ""
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i].Role == "" || msgs[i].Role == "user" {
			question = msgs[i].Content
			break
		}
	}
	acc := helpAccess(r.Context(), a, claims, app, path)
	ctx := a.Help.Context(acc)
	docs := a.Help.Ranked(acc, question, 5)

	var b strings.Builder
	b.WriteString("Voce e o assistente de ajuda do Chosen ERP. Responda em portugues do Brasil, de forma objetiva e passo a passo. ")
	b.WriteString("Use SOMENTE a documentacao fornecida como base. Se a documentacao nao resolver, oriente abrir um chamado no Suporte. ")
	b.WriteString("Nunca execute acoes, nunca invente botoes/rotas e nunca revele dados de membros, financas, credenciais ou de outras igrejas. ")
	b.WriteString("Quando a resposta vier da documentacao, cite o titulo do artigo entre parenteses. Se nao houver base suficiente, comece a resposta com [ESCALAR].\n\n")
	fmt.Fprintf(&b, "Aplicacao: %s\nRota atual: %s\nPapel do usuario: %s\n\n", app, path, claims.Role)
	if ctx.Guide != nil {
		fmt.Fprintf(&b, "Guia da tela atual:\n# %s\n%s\n\n", ctx.Guide.Title, truncateRunes(ctx.Guide.Body, 2500))
	}
	if len(docs) > 0 {
		b.WriteString("Artigos relevantes:\n")
		for i, d := range docs {
			fmt.Fprintf(&b, "--- Artigo %d: %s (slug: %s) ---\n%s\n\n", i+1, d.Title, d.Slug, truncateRunes(d.Body, 2200))
		}
	}
	answer, err := a.HelpAI.ChatMessages(r.Context(), b.String(), msgs, ai.Options{Temperature: 0.2, MaxTokens: 900})
	if err != nil {
		if errors.Is(err, ai.ErrDisabled) {
			writeErr(w, http.StatusServiceUnavailable, "ai_disabled")
			return
		}
		writeErr(w, http.StatusBadGateway, "ai_unavailable")
		return
	}
	escalate := strings.Contains(strings.ToUpper(answer), "[ESCALAR]")
	answer = strings.ReplaceAll(answer, "[ESCALAR]", "")
	answer = strings.ReplaceAll(answer, "[escalar]", "")
	answer = strings.TrimSpace(answer)

	sources := make([]help.Summary, 0, len(docs))
	for _, d := range docs {
		sources = append(sources, help.Summary{Slug: d.Slug, App: d.App, Title: d.Title, Category: d.Category, Summary: d.Summary})
	}
	if len(docs) == 0 {
		escalate = true
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"answer":   answer,
		"sources":  sources,
		"escalate": escalate,
	})
}

func sanitizeChat(in []ai.Message) []ai.Message {
	out := make([]ai.Message, 0, len(in))
	for _, m := range in {
		role := strings.ToLower(strings.TrimSpace(m.Role))
		if role != "user" && role != "assistant" {
			continue
		}
		content := strings.TrimSpace(m.Content)
		if content == "" {
			continue
		}
		content = truncateRunes(content, 2000)
		out = append(out, ai.Message{Role: role, Content: content})
	}
	if len(out) > 10 {
		out = out[len(out)-10:]
	}
	return out
}

func truncateRunes(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "..."
}

type helpFeedbackIn struct {
	App     string `json:"app"`
	Slug    string `json:"slug"`
	Helpful bool   `json:"helpful"`
	Comment string `json:"comment"`
}

func (a *App) handleHelpFeedback(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	var in helpFeedbackIn
	if err := readJSONMax(r, &in, 16<<10); err != nil {
		writeErr(w, http.StatusBadRequest, "payload invalido")
		return
	}
	app := helpApp(in.App)
	var err error
	if claims.TenantID == "" {
		err = a.Store.WithSystem(r.Context(), func(tx pgx.Tx) error {
			return a.Support.AddFeedback(r.Context(), tx, "", "", claims.UserID, app, in.Slug, in.Helpful, in.Comment)
		})
	} else {
		err = a.Store.WithTenant(r.Context(), boundsFromClaims(claims), func(tx pgx.Tx) error {
			return a.Support.AddFeedback(r.Context(), tx, claims.TenantID, claims.BranchID, claims.UserID, app, in.Slug, in.Helpful, in.Comment)
		})
	}
	if err != nil {
		writeServerError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}
