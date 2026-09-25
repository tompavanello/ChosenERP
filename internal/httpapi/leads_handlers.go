package httpapi

import (
	"net/http"
	"strings"
)

// leadInput e o corpo do formulario de contato do site institucional.
// As tags devem bater com o JSON enviado pelo app apps/marketing.
type leadInput struct {
	Name       string `json:"name"`
	Email      string `json:"email"`
	Phone      string `json:"phone"`
	ChurchName string `json:"church_name"`
	ChurchSize string `json:"church_size"`
	Message    string `json:"message"`
	Source     string `json:"source"`
}

// handleCreateLead registra um lead vindo do site (rota publica, sob rate limit).
// Nao devolve dado sensivel; so confirma o recebimento.
func (a *App) handleCreateLead(w http.ResponseWriter, r *http.Request) {
	var in leadInput
	if err := readJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "dados invalidos")
		return
	}

	in.Name = strings.TrimSpace(in.Name)
	in.Email = strings.ToLower(strings.TrimSpace(in.Email))
	in.Phone = strings.TrimSpace(in.Phone)
	in.ChurchName = strings.TrimSpace(in.ChurchName)
	in.ChurchSize = strings.TrimSpace(in.ChurchSize)
	in.Message = strings.TrimSpace(in.Message)
	in.Source = strings.TrimSpace(in.Source)
	if in.Source == "" {
		in.Source = "site"
	}

	switch {
	case in.Name == "" || len(in.Name) > 120:
		writeErr(w, http.StatusBadRequest, "informe o seu nome")
		return
	case !validEmail(in.Email) || len(in.Email) > 160:
		writeErr(w, http.StatusBadRequest, "informe um e-mail valido")
		return
	case len(in.Phone) > 40 || len(in.ChurchName) > 160 ||
		len(in.ChurchSize) > 60 || len(in.Message) > 2000:
		writeErr(w, http.StatusBadRequest, "conteudo muito longo")
		return
	}

	_, err := a.Store.Pool().Exec(r.Context(),
		`INSERT INTO marketing_leads
		     (name, email, phone, church_name, church_size, message, source)
		 VALUES ($1, $2, NULLIF($3, ''), NULLIF($4, ''), NULLIF($5, ''),
		         NULLIF($6, ''), $7)`,
		in.Name, in.Email, in.Phone, in.ChurchName, in.ChurchSize, in.Message, in.Source)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "nao foi possivel registrar agora")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{"ok": true})
}

// validEmail faz uma validacao minima e sem regex (barata e suficiente para um
// campo de contato): exige um "@" central e um ponto no dominio.
func validEmail(s string) bool {
	if strings.ContainsAny(s, " \t\r\n") {
		return false
	}
	at := strings.IndexByte(s, '@')
	if at <= 0 || at == len(s)-1 {
		return false
	}
	dot := strings.LastIndexByte(s, '.')
	return dot > at+1 && dot < len(s)-1
}
