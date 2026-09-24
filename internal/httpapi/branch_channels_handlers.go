package httpapi

import (
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5"

	"chosenerp/internal/delivery"
	"chosenerp/internal/org"
	"chosenerp/internal/store"
)

// evolution devolve um cliente da Evolution API com as credenciais globais.
func (a *App) evolution() *delivery.EvolutionClient {
	return &delivery.EvolutionClient{
		BaseURL: a.Config.EvolutionAPIURL,
		APIKey:  a.Config.EvolutionAPIKey,
	}
}

// handleGetBranchChannels le a configuracao de canais (WhatsApp/SMTP) da filial.
func (a *App) handleGetBranchChannels(w http.ResponseWriter, r *http.Request) {
	claims, ok := a.adminClaims(w, r)
	if !ok {
		return
	}
	id := r.PathValue("id")
	b := boundsFromClaims(claims)
	var out *org.BranchChannels
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		var err error
		out, err = a.Org.GetBranchChannels(r.Context(), tx, id)
		return err
	})
	if err != nil {
		if store.IsNotFound(err) {
			writeErr(w, http.StatusNotFound, "filial nao encontrada")
			return
		}
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, out)
}

// handleUpdateBranchChannels grava a configuracao de canais da filial.
func (a *App) handleUpdateBranchChannels(w http.ResponseWriter, r *http.Request) {
	claims, ok := a.adminClaims(w, r)
	if !ok {
		return
	}
	id := r.PathValue("id")
	var in org.BranchChannelsInput
	if err := readJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	b := boundsFromClaims(claims)
	var out *org.BranchChannels
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		var err error
		out, err = a.Org.UpdateBranchChannels(r.Context(), tx, id, in)
		return err
	})
	if err != nil {
		if store.IsNotFound(err) {
			writeErr(w, http.StatusNotFound, "filial nao encontrada")
			return
		}
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, out)
}

// handleConnectBranchWhatsApp cria (ou reconecta) a instancia Evolution cujo
// nome e o id da filial e devolve o QR Code para leitura no WhatsApp.
func (a *App) handleConnectBranchWhatsApp(w http.ResponseWriter, r *http.Request) {
	claims, ok := a.adminClaims(w, r)
	if !ok {
		return
	}
	if a.Config.EvolutionAPIURL == "" || a.Config.EvolutionAPIKey == "" {
		writeErr(w, http.StatusBadRequest, "integracao WhatsApp (Evolution) nao configurada no servidor")
		return
	}
	id := r.PathValue("id")
	b := boundsFromClaims(claims)

	// Confirma que a filial existe no escopo e usa o id como nome da instancia.
	if err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		_, err := a.Org.GetBranchChannels(r.Context(), tx, id)
		return err
	}); err != nil {
		if store.IsNotFound(err) {
			writeErr(w, http.StatusNotFound, "filial nao encontrada")
			return
		}
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}

	client := a.evolution()
	qr, err := client.CreateInstance(r.Context(), id)
	if err != nil {
		// Instancia provavelmente ja existe: tenta apenas reconectar.
		qr, err = client.Connect(r.Context(), id)
		if err != nil {
			writeErr(w, http.StatusBadGateway, "nao foi possivel conectar ao WhatsApp: "+err.Error())
			return
		}
	}

	// Se a instancia ja estava conectada (ex.: "Reconectar"), captura o numero.
	number := ""
	status := "connecting"
	if st, err := client.State(r.Context(), id); err == nil && st == "connected" {
		status = "connected"
		if info, err := client.FetchInstance(r.Context(), id); err == nil {
			number = delivery.ConnectedNumber(info)
		}
	}
	_ = a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		return a.Org.SetBranchWhatsApp(r.Context(), tx, id, id, status, number)
	})
	writeJSON(w, http.StatusOK, map[string]any{
		"instance":      id,
		"status":        status,
		"number":        number,
		"qrcode_base64": qr.Base64,
		"code":          qr.Code,
	})
}

// handleBranchWhatsAppState consulta o estado da conexao e sincroniza o banco.
func (a *App) handleBranchWhatsAppState(w http.ResponseWriter, r *http.Request) {
	claims, ok := a.adminClaims(w, r)
	if !ok {
		return
	}
	id := r.PathValue("id")
	b := boundsFromClaims(claims)
	state := "disconnected"
	number := ""
	if a.Config.EvolutionAPIURL != "" && a.Config.EvolutionAPIKey != "" {
		if s, err := a.evolution().State(r.Context(), id); err == nil {
			state = s
		}
		if state == "connected" {
			if info, err := a.evolution().FetchInstance(r.Context(), id); err == nil {
				number = delivery.ConnectedNumber(info)
			}
		}
	}
	_ = a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		return a.Org.SetBranchWhatsApp(r.Context(), tx, id, id, state, number)
	})
	writeJSON(w, http.StatusOK, map[string]any{
		"instance":  id,
		"status":    state,
		"connected": state == "connected",
		"number":    number,
	})
}

// handleDisconnectBranchWhatsApp desconecta a instancia da filial.
func (a *App) handleDisconnectBranchWhatsApp(w http.ResponseWriter, r *http.Request) {
	claims, ok := a.adminClaims(w, r)
	if !ok {
		return
	}
	id := r.PathValue("id")
	b := boundsFromClaims(claims)
	if a.Config.EvolutionAPIURL != "" && a.Config.EvolutionAPIKey != "" {
		if err := a.evolution().Logout(r.Context(), id); err != nil {
			// Mesmo falhando na Evolution, refletimos o estado local.
			_ = err
		}
	}
	if err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		return a.Org.SetBranchWhatsApp(r.Context(), tx, id, id, "disconnected", "")
	}); err != nil && !errors.Is(err, pgx.ErrNoRows) {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "status": "disconnected"})
}
