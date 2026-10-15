package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5"

	"chosenerp/internal/ai"
	"chosenerp/internal/content"
	"chosenerp/internal/store"
)

// ---- Videos (staff) ----

func (a *App) handleListVideos(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	var out []content.Video
	err := a.Store.WithTenant(r.Context(), boundsFromClaims(claims), func(tx pgx.Tx) error {
		var e error
		out, e = a.Content.ListVideos(r.Context(), tx, false)
		return e
	})
	if err != nil {
		writeServerError(w, err)
		return
	}
	if out == nil {
		out = []content.Video{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"videos": out})
}

func (a *App) handleCreateVideo(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	var in content.VideoInput
	if err := readJSON(r, &in); err != nil || strings.TrimSpace(in.Title) == "" {
		writeErr(w, http.StatusBadRequest, "titulo obrigatorio")
		return
	}
	if resolveYT(in) == "" {
		writeErr(w, http.StatusBadRequest, "informe a URL ou o id do video do YouTube")
		return
	}
	var out *content.Video
	err := a.Store.WithTenant(r.Context(), boundsFromClaims(claims), func(tx pgx.Tx) error {
		var e error
		out, e = a.Content.CreateVideo(r.Context(), tx, claims.UserID, in)
		return e
	})
	if err != nil {
		writeServerError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, out)
}

func (a *App) handleUpdateVideo(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	var in content.VideoInput
	if err := readJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	var out *content.Video
	err := a.Store.WithTenant(r.Context(), boundsFromClaims(claims), func(tx pgx.Tx) error {
		var e error
		out, e = a.Content.UpdateVideo(r.Context(), tx, r.PathValue("id"), in)
		return e
	})
	if err != nil {
		if store.IsNotFound(err) {
			writeErr(w, http.StatusNotFound, "video nao encontrado")
			return
		}
		writeServerError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (a *App) handleDeleteVideo(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	err := a.Store.WithTenant(r.Context(), boundsFromClaims(claims), func(tx pgx.Tx) error {
		return a.Content.DeleteVideo(r.Context(), tx, r.PathValue("id"))
	})
	if err != nil {
		if store.IsNotFound(err) {
			writeErr(w, http.StatusNotFound, "video nao encontrado")
			return
		}
		writeServerError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func resolveYT(in content.VideoInput) string {
	if in.YouTubeID != nil && strings.TrimSpace(*in.YouTubeID) != "" {
		return strings.TrimSpace(*in.YouTubeID)
	}
	return content.ExtractYouTubeID(in.URL)
}

// ---- Devocionais (staff) ----

func (a *App) handleListDevotionalPlans(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	var out []content.Plan
	err := a.Store.WithTenant(r.Context(), boundsFromClaims(claims), func(tx pgx.Tx) error {
		var e error
		out, e = a.Content.ListPlans(r.Context(), tx, false)
		return e
	})
	if err != nil {
		writeServerError(w, err)
		return
	}
	if out == nil {
		out = []content.Plan{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"plans": out})
}

func (a *App) handleCreateDevotionalPlan(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	var in content.PlanInput
	if err := readJSON(r, &in); err != nil || strings.TrimSpace(in.Title) == "" {
		writeErr(w, http.StatusBadRequest, "titulo obrigatorio")
		return
	}
	var out *content.Plan
	err := a.Store.WithTenant(r.Context(), boundsFromClaims(claims), func(tx pgx.Tx) error {
		var e error
		out, e = a.Content.CreatePlan(r.Context(), tx, claims.UserID, in)
		return e
	})
	if err != nil {
		writeServerError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, out)
}

func (a *App) handleGetDevotionalPlan(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	id := r.PathValue("id")
	var plan *content.Plan
	var days []content.Day
	err := a.Store.WithTenant(r.Context(), boundsFromClaims(claims), func(tx pgx.Tx) error {
		var e error
		if plan, e = a.Content.GetPlan(r.Context(), tx, id); e != nil {
			return e
		}
		days, e = a.Content.ListDays(r.Context(), tx, id)
		return e
	})
	if err != nil {
		if store.IsNotFound(err) {
			writeErr(w, http.StatusNotFound, "plano nao encontrado")
			return
		}
		writeServerError(w, err)
		return
	}
	if days == nil {
		days = []content.Day{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"plan": plan, "days": days})
}

func (a *App) handleUpdateDevotionalPlan(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	var in content.PlanInput
	if err := readJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	var out *content.Plan
	err := a.Store.WithTenant(r.Context(), boundsFromClaims(claims), func(tx pgx.Tx) error {
		var e error
		out, e = a.Content.UpdatePlan(r.Context(), tx, r.PathValue("id"), in)
		return e
	})
	if err != nil {
		if store.IsNotFound(err) {
			writeErr(w, http.StatusNotFound, "plano nao encontrado")
			return
		}
		writeServerError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (a *App) handleDeleteDevotionalPlan(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	err := a.Store.WithTenant(r.Context(), boundsFromClaims(claims), func(tx pgx.Tx) error {
		return a.Content.DeletePlan(r.Context(), tx, r.PathValue("id"))
	})
	if err != nil {
		if store.IsNotFound(err) {
			writeErr(w, http.StatusNotFound, "plano nao encontrado")
			return
		}
		writeServerError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (a *App) handlePutDevotionalDays(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	var in struct {
		Days []content.DayInput `json:"days"`
	}
	if err := readJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	id := r.PathValue("id")
	var days []content.Day
	err := a.Store.WithTenant(r.Context(), boundsFromClaims(claims), func(tx pgx.Tx) error {
		if _, e := a.Content.GetPlan(r.Context(), tx, id); e != nil {
			return e
		}
		if e := a.Content.ReplaceDays(r.Context(), tx, id, in.Days); e != nil {
			return e
		}
		var e error
		days, e = a.Content.ListDays(r.Context(), tx, id)
		return e
	})
	if err != nil {
		if store.IsNotFound(err) {
			writeErr(w, http.StatusNotFound, "plano nao encontrado")
			return
		}
		writeServerError(w, err)
		return
	}
	if days == nil {
		days = []content.Day{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"days": days})
}

// ---- Geracao de devocional por IA ----

func (a *App) handleGenerateDevotional(w http.ResponseWriter, r *http.Request) {
	if _, ok := claimsFrom(r.Context()); !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	if a.AI == nil || !a.AI.Enabled() {
		writeErr(w, http.StatusServiceUnavailable, "ai_disabled")
		return
	}
	var in struct {
		Theme     string `json:"theme"`
		Reference string `json:"reference"`
		Audience  string `json:"audience"`
		Days      int    `json:"days"`
	}
	if err := readJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	days := in.Days
	if days <= 0 || days > 31 {
		days = 7
	}
	system := "Voce e um assistente pastoral que cria planos devocionais cristaos em portugues do Brasil, " +
		"com linguagem acolhedora, biblica e pratica. Responda SOMENTE com JSON valido, sem markdown."
	user := fmt.Sprintf(
		"Crie um plano devocional com %d dias. Tema: %q. Referencia base: %q. Publico: %q.\n"+
			"Responda apenas JSON no formato: "+
			`{"days":[{"day_number":1,"title":"...","bible_reference":"João 3:16","bible_text":"...","body":"...","prayer":"..."}]}`,
		days, strings.TrimSpace(in.Theme), strings.TrimSpace(in.Reference), strings.TrimSpace(in.Audience))

	txt, err := a.AI.Chat(r.Context(), system, user)
	if err != nil {
		if errors.Is(err, ai.ErrDisabled) {
			writeErr(w, http.StatusServiceUnavailable, "ai_disabled")
			return
		}
		writeErr(w, http.StatusBadGateway, err.Error())
		return
	}
	var draft struct {
		Days []content.DayInput `json:"days"`
	}
	if err := json.Unmarshal([]byte(extractJSON(txt)), &draft); err != nil {
		writeErr(w, http.StatusBadGateway, "a IA retornou um formato inesperado; tente novamente")
		return
	}
	if draft.Days == nil {
		draft.Days = []content.DayInput{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"days": draft.Days})
}

// extractJSON isola o primeiro objeto JSON de um texto (a IA pode envolver em
// cercas de codigo ou comentarios).
func extractJSON(s string) string {
	i := strings.Index(s, "{")
	j := strings.LastIndex(s, "}")
	if i < 0 || j <= i {
		return s
	}
	return s[i : j+1]
}

// ---- Live settings (staff) ----

func (a *App) handleGetLiveSettings(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	var out *content.LiveSettings
	err := a.Store.WithTenant(r.Context(), boundsFromClaims(claims), func(tx pgx.Tx) error {
		var e error
		out, e = a.Content.LiveSettings(r.Context(), tx)
		return e
	})
	if err != nil {
		writeServerError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (a *App) handlePutLiveSettings(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	var in struct {
		ChannelID string `json:"youtube_channel_id"`
		LiveURL   string `json:"live_stream_url"`
	}
	if err := readJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	if s := strings.TrimSpace(in.LiveURL); s != "" && !isHTTPURL(s) {
		writeErr(w, http.StatusBadRequest, "a URL do culto ao vivo deve comecar com http:// ou https://")
		return
	}
	var out *content.LiveSettings
	err := a.Store.WithTenant(r.Context(), boundsFromClaims(claims), func(tx pgx.Tx) error {
		var e error
		out, e = a.Content.SetLiveSettings(r.Context(), tx, in.ChannelID, in.LiveURL)
		return e
	})
	if err != nil {
		writeServerError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (a *App) handleGetBibleSettings(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	var out *content.BibleSettings
	err := a.Store.WithTenant(r.Context(), boundsFromClaims(claims), func(tx pgx.Tx) error {
		var e error
		out, e = a.Content.BibleSettings(r.Context(), tx)
		return e
	})
	if err != nil {
		writeServerError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (a *App) handlePutBibleSettings(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	var in struct {
		BibleVersionID string `json:"bible_version_id"`
	}
	if err := readJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	var out *content.BibleSettings
	err := a.Store.WithTenant(r.Context(), boundsFromClaims(claims), func(tx pgx.Tx) error {
		var e error
		out, e = a.Content.SetBibleVersion(r.Context(), tx, in.BibleVersionID)
		return e
	})
	if err != nil {
		writeServerError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

// ---- App do membro (self, sem permissao) ----
func (a *App) handleMeVideos(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	var out []content.Video
	err := a.Store.WithTenant(r.Context(), boundsFromClaims(claims), func(tx pgx.Tx) error {
		var e error
		out, e = a.Content.ListVideos(r.Context(), tx, true)
		return e
	})
	if err != nil {
		writeServerError(w, err)
		return
	}
	if out == nil {
		out = []content.Video{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"videos": out})
}

func (a *App) handleMeLive(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	var out *content.LiveSettings
	err := a.Store.WithTenant(r.Context(), boundsFromClaims(claims), func(tx pgx.Tx) error {
		var e error
		out, e = a.Content.LiveSettings(r.Context(), tx)
		return e
	})
	if err != nil {
		writeServerError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

// devotionalSummary junta o plano com o progresso do usuario.
type devotionalSummary struct {
	content.Plan
	CompletedCount int `json:"completed_count"`
}

func (a *App) handleMeDevotionalPlans(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	var out []devotionalSummary
	err := a.Store.WithTenant(r.Context(), boundsFromClaims(claims), func(tx pgx.Tx) error {
		plans, e := a.Content.ListPlans(r.Context(), tx, true)
		if e != nil {
			return e
		}
		for _, p := range plans {
			done, e := a.Content.CompletedDays(r.Context(), tx, p.ID, claims.UserID)
			if e != nil {
				return e
			}
			out = append(out, devotionalSummary{Plan: p, CompletedCount: len(done)})
		}
		return nil
	})
	if err != nil {
		writeServerError(w, err)
		return
	}
	if out == nil {
		out = []devotionalSummary{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"plans": out})
}

func (a *App) handleMeDevotionalPlan(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	id := r.PathValue("id")
	var plan *content.Plan
	var days []content.Day
	var completed []int
	err := a.Store.WithTenant(r.Context(), boundsFromClaims(claims), func(tx pgx.Tx) error {
		var e error
		if plan, e = a.Content.GetPlan(r.Context(), tx, id); e != nil {
			return e
		}
		if days, e = a.Content.ListDays(r.Context(), tx, id); e != nil {
			return e
		}
		completed, e = a.Content.CompletedDays(r.Context(), tx, id, claims.UserID)
		return e
	})
	if err != nil {
		if store.IsNotFound(err) {
			writeErr(w, http.StatusNotFound, "plano nao encontrado")
			return
		}
		writeServerError(w, err)
		return
	}
	if days == nil {
		days = []content.Day{}
	}
	if completed == nil {
		completed = []int{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"plan": plan, "days": days, "completed": completed})
}

func (a *App) handleMeDevotionalComplete(w http.ResponseWriter, r *http.Request) {
	a.setDevotionalDay(w, r, true)
}

func (a *App) handleMeDevotionalUncomplete(w http.ResponseWriter, r *http.Request) {
	a.setDevotionalDay(w, r, false)
}

func (a *App) setDevotionalDay(w http.ResponseWriter, r *http.Request, complete bool) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	planID := r.PathValue("id")
	day, err := atoiSafe(r.PathValue("day"))
	if err != nil {
		writeErr(w, http.StatusBadRequest, "dia invalido")
		return
	}
	err = a.Store.WithTenant(r.Context(), boundsFromClaims(claims), func(tx pgx.Tx) error {
		if _, e := a.Content.GetPlan(r.Context(), tx, planID); e != nil {
			return e
		}
		memberID, e := memberIDForClaims(r.Context(), tx, claims)
		if e != nil {
			return e
		}
		return a.Content.SetDayComplete(r.Context(), tx, claims.TenantID, planID, claims.UserID, memberID, day, complete)
	})
	if err != nil {
		if store.IsNotFound(err) {
			writeErr(w, http.StatusNotFound, "plano nao encontrado")
			return
		}
		writeServerError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func atoiSafe(s string) (int, error) {
	n := 0
	if s == "" {
		return 0, errors.New("vazio")
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, errors.New("nao numerico")
		}
		n = n*10 + int(c-'0')
		if n > 100000 {
			return 0, errors.New("grande demais")
		}
	}
	return n, nil
}
