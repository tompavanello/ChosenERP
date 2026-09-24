package httpapi

import (
	"encoding/base64"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/jackc/pgx/v5"
	qrcode "github.com/skip2/go-qrcode"

	"chosenerp/internal/announcements"
	"chosenerp/internal/delivery"
	"chosenerp/internal/documents"
	"chosenerp/internal/members"
	"chosenerp/internal/store"
)

// resolvePublicCard resolve a carteirinha (sem auth) e o membro dono dela.
//
// Existe para os tres endpoints publicos compartilharem a resolucao: o token
// vem do QR impresso no cartao, entao nada aqui pode assumir sessao.
// Devolve ok=false ja tendo escrito a resposta de erro.
func (a *App) resolvePublicCard(w http.ResponseWriter, r *http.Request, token string) (
	*documents.CardInfo, *members.Member, []announcements.Announcement, bool,
) {
	var card *documents.CardInfo
	err := a.Store.WithSystem(r.Context(), func(tx pgx.Tx) error {
		var err error
		card, err = a.Documents.ResolveCard(r.Context(), tx, token)
		return err
	})
	if err != nil {
		if store.IsNotFound(err) {
			writeErr(w, http.StatusNotFound, "carteirinha nao encontrada")
			return nil, nil, nil, false
		}
		writeErr(w, http.StatusInternalServerError, err.Error())
		return nil, nil, nil, false
	}

	bounds := store.Bounds{TenantID: card.TenantID, BranchID: card.BranchID, Role: "membro"}
	var member *members.Member
	var anns []announcements.Announcement
	err = a.Store.WithTenant(r.Context(), bounds, func(tx pgx.Tx) error {
		var err error
		if member, err = a.Members.Get(r.Context(), tx, card.MemberID); err != nil {
			return err
		}
		anns, err = a.Announcements.List(r.Context(), tx)
		return err
	})
	if err != nil {
		if store.IsNotFound(err) {
			writeErr(w, http.StatusNotFound, "membro nao encontrado")
			return nil, nil, nil, false
		}
		writeErr(w, http.StatusInternalServerError, err.Error())
		return nil, nil, nil, false
	}
	return card, member, anns, true
}

// handlePublicCard devolve a carteirinha do membro pelo token do QR (sem auth).
//
// ATENCAO: a resposta e uma PROJECAO MINIMA de proposito. O token que endereca
// este endpoint esta impresso no proprio cartao (o QR o carrega), entao tudo o
// que for devolvido aqui e alcancavel por qualquer pessoa que fotografar uma
// carteirinha - sem rate limit e sem rotacao de token, para sempre. Antes esta
// funcao devolvia o members.Member inteiro, o que vazava CPF, RG, data de
// nascimento, e-mail, telefone e WhatsApp de quem tivesse o link.
func (a *App) handlePublicCard(w http.ResponseWriter, r *http.Request) {
	card, member, anns, ok := a.resolvePublicCard(w, r, r.PathValue("token"))
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"member": map[string]any{
			"full_name": member.FullName,
		},
		"card": map[string]any{
			"token":       card.Token,
			"card_ref":    card.Ref,
			"member":      card.MemberName,
			"branch_name": card.BranchName,
		},
		"announcements": anns,
	})
}

// handlePublicCardPhoto serve a foto do membro dono da carteirinha.
//
// Existe para a pagina publica e a impressao nao dependerem de
// /api/v1/attachments/{arquivo}: aquele endpoint nao tem auth nenhuma e serve
// tambem comprovantes financeiros, entao libera-lo atras do Access publicaria
// as duas coisas juntas. Aqui o acesso e amarrado ao token da carteirinha.
func (a *App) handlePublicCardPhoto(w http.ResponseWriter, r *http.Request) {
	_, member, _, ok := a.resolvePublicCard(w, r, r.PathValue("token"))
	if !ok {
		return
	}
	if member.PhotoURL == nil || *member.PhotoURL == "" {
		writeErr(w, http.StatusNotFound, "membro sem foto cadastrada")
		return
	}
	// O banco guarda "/api/v1/attachments/<nome>"; filepath.Base garante que so
	// o nome do arquivo chegue ao disco, sem travessia de caminho.
	filename := filepath.Base(*member.PhotoURL)
	diskPath := filepath.Join(a.Config.UploadDir, filename)
	if _, err := os.Stat(diskPath); err != nil {
		writeErr(w, http.StatusNotFound, "arquivo nao encontrado")
		return
	}
	ct := mime.TypeByExtension(filepath.Ext(filename))
	if ct == "" {
		ct = "application/octet-stream"
	}
	w.Header().Set("Content-Type", ct)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Cache-Control", "private, max-age=86400")
	http.ServeFile(w, r, diskPath)
}

func (a *App) handleListAnnouncements(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	b := boundsFromClaims(claims)
	var out []announcements.Announcement
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		var err error
		out, err = a.Announcements.List(r.Context(), tx)
		return err
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"announcements": out})
}

func (a *App) handleCreateAnnouncement(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	var in announcements.UpsertInput
	if err := readJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	if in.Title == nil || strings.TrimSpace(*in.Title) == "" {
		writeErr(w, http.StatusBadRequest, "title required")
		return
	}
	b := boundsFromClaims(claims)
	var an *announcements.Announcement
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		var err error
		an, err = a.Announcements.Create(r.Context(), tx, claims.TenantID, claims.BranchID, in)
		return err
	})
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, an)
}

func (a *App) handleMemberCardHTML(w http.ResponseWriter, r *http.Request) {
	card, _, _, ok := a.resolvePublicCard(w, r, r.PathValue("token"))
	if !ok {
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("X-Robots-Tag", "noindex, nofollow")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(memberCardHTML(card, a.Config.AppBaseURL)))
}

// memberCardHTML monta a carteirinha para impressao. A foto e opcional: quando
// o membro nao tem foto cadastrada, o bloco dela simplesmente nao e renderizado
// (antes a carteirinha saia sem foto e sem numero de identificacao).
//
// A foto vem de /api/v1/public/card/{token}/photo, e nao de /api/v1/attachments:
// o caminho publico acompanha o token da carteirinha e e o unico que pode ficar
// liberado no Access sem arrastar os anexos do financeiro junto.
func memberCardHTML(c *documents.CardInfo, baseURL string) string {
	foto := ""
	if c.PhotoURL != nil && *c.PhotoURL != "" {
		src := baseURL + "/api/v1/public/card/" + c.Token + "/photo"
		foto = `<img class="foto" src="` + htmlEscape(src) + `" alt="Foto de ` + htmlEscape(c.MemberName) + `">`
	}
	filial := ""
	if c.BranchName != nil && *c.BranchName != "" {
		filial = `<div class="linha"><span>Filial</span><strong>` + htmlEscape(*c.BranchName) + `</strong></div>`
	}
	// O QR e o que liga o cartao fisico ao link publico. Antes esta folha dizia
	// "Validacao: QR Code" sem desenhar QR nenhum - o cartao impresso nao levava
	// a lugar nenhum. Usamos delivery.PublicLink (o mesmo helper do e-mail) para
	// o QR e o link enviado por WhatsApp nunca apontarem para lugares diferentes.
	qr := ""
	if link := delivery.PublicLink(baseURL, documents.KindMembershipCard, c.Token); link != "" {
		if uri := qrDataURI(link, 240); uri != "" {
			qr = `<div class="qr"><img src="` + uri + `" alt="QR Code da carteirinha" width="104" height="104"><span>Aponte a camera para abrir a<br>carteirinha digital e ver os avisos</span></div>`
		}
	}
	// O token em texto saiu daqui: ele e a credencial da carteirinha e ja viaja
	// no QR; imprimi-lo em claro so facilitava copia-lo de uma foto do cartao.
	return `<!doctype html><html lang="pt-BR"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><meta name="robots" content="noindex,nofollow"><title>Carteirinha - ` + htmlEscape(c.MemberName) + `</title>
<style>body{font-family:ui-sans-serif,system-ui,sans-serif;background:#f1f5f9;margin:0;padding:24px;color:#0b1020}.card{max-width:460px;margin:0 auto;background:linear-gradient(135deg,#0369a1,#0ea5e9);color:#fff;border-radius:20px;padding:26px;box-shadow:0 12px 32px rgba(3,105,161,.35)}.brand{font-size:12px;letter-spacing:.1em;text-transform:uppercase;opacity:.85;margin-bottom:18px}.top{display:flex;gap:18px;align-items:center}.foto{width:96px;height:96px;border-radius:14px;object-fit:cover;border:3px solid rgba(255,255,255,.55);background:rgba(255,255,255,.18)}.nome{font-size:24px;font-weight:700;line-height:1.15}.num{margin-top:6px;font-size:13px;font-family:ui-monospace,SFMono-Regular,monospace;letter-spacing:.08em;background:rgba(255,255,255,.18);display:inline-block;padding:3px 9px;border-radius:999px}.linhas{margin-top:20px;border-top:1px solid rgba(255,255,255,.28);padding-top:14px;font-size:13px}.linha{display:flex;justify-content:space-between;gap:12px;padding:3px 0}.linha span{opacity:.8}.qr{display:flex;align-items:center;gap:14px;margin-top:14px;background:#fff;border-radius:14px;padding:12px}.qr img{display:block;border-radius:6px}.qr span{color:#475569;font-size:11px;line-height:1.45}@media print{body{background:#fff;padding:0}.card{box-shadow:none}}</style></head><body>
<div class="card"><div class="brand">Carteirinha de Membro</div>
<div class="top">` + foto + `<div><div class="nome">` + htmlEscape(c.MemberName) + `</div><div class="num">` + htmlEscape(c.Ref) + `</div></div></div>
<div class="linhas">` + filial + `</div>` + qr + `
</div>
</body></html>`
}

// qrDataURI gera o QR em PNG e devolve como data: URI, para a folha impressa
// nao depender de rede nem de servico externo - um gerador remoto receberia o
// token da carteirinha, que e justamente a credencial que o QR carrega.
//
// Devolve "" se a geracao falhar: um QR ausente e melhor que uma carteirinha
// que nao renderiza (o chamador simplesmente omite o bloco).
func qrDataURI(conteudo string, tamanho int) string {
	png, err := qrcode.Encode(conteudo, qrcode.Medium, tamanho)
	if err != nil {
		return ""
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(png)
}

func htmlEscape(s string) string {
	if s == "" {
		return ""
	}
	out := make([]rune, 0, len(s))
	for _, r := range s {
		switch r {
		case '<':
			out = append(out, '&', 'l', 't', ';')
		case '>':
			out = append(out, '&', 'g', 't', ';')
		case '&':
			out = append(out, '&', 'a', 'm', 'p', ';')
		default:
			out = append(out, r)
		}
	}
	return string(out)
}
