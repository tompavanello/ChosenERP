package httpapi

import (
	"encoding/json"
	"net/http"
	"time"

	"chosenerp/internal/insights"
	"github.com/jackc/pgx/v5"
)

// handleInsights devolve os cartoes de leitura (financeiro, pessoas e operacao)
// do periodo. As regras rodam no escopo RLS do tenant.
func (a *App) handleInsights(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	scope := first(r.URL.Query()["scope"])
	from, to := insightPeriod(r)
	b := boundsFromClaims(claims)
	var rep insights.Report
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		var err error
		rep, err = a.Insights.Build(r.Context(), tx, scope, from, to)
		return err
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, rep)
}

// handleInsightsNarrative gera um resumo executivo por IA. Sem chave, devolve
// 200 com ai_enabled=false para a UI cair no modo somente regras.
func (a *App) handleInsightsNarrative(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	if !a.AI.Enabled() {
		writeJSON(w, http.StatusOK, map[string]any{"ai_enabled": false, "narrative": ""})
		return
	}
	scope := first(r.URL.Query()["scope"])
	from, to := insightPeriod(r)
	b := boundsFromClaims(claims)
	var rep insights.Report
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		var err error
		rep, err = a.Insights.Build(r.Context(), tx, scope, from, to)
		return err
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}

	payload, _ := json.Marshal(map[string]any{
		"periodo": map[string]string{"de": rep.From, "ate": rep.To},
		"numeros": rep.Summary,
		"achados": rep.Insights,
	})
	system := "Voce e um analista de dados de uma igreja. Escreva um resumo executivo em portugues do Brasil, " +
		"em no maximo 3 paragrafos curtos. Use apenas os numeros e achados recebidos, nunca invente dados, " +
		"nunca cite nomes de pessoas, nunca execute acoes e nao exponha dados sensiveis. " +
		"Priorize riscos, oportunidades e proximos passos praticos."
	narrative, err := a.AI.Chat(r.Context(), system, string(payload))
	if err != nil {
		writeErr(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ai_enabled": true, "narrative": narrative})
}

// insightPeriod usa from/to; sem filtro, assume o mes corrente.
func insightPeriod(r *http.Request) (string, string) {
	from, to := exportPeriod(r.URL.Query())
	if from == "" && to == "" {
		now := time.Now()
		from = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()).Format("2006-01-02")
		to = time.Date(now.Year(), now.Month()+1, 0, 0, 0, 0, 0, now.Location()).Format("2006-01-02")
	}
	return from, to
}
