package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"chosenerp/internal/documents"
	"chosenerp/internal/store"
)

func ptrStr(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

// handleListMemberDocuments lista os certificados/cartas emitidos para o membro.
func (a *App) handleListMemberDocuments(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	id := r.PathValue("id")
	var out []documents.Document
	err := a.Store.WithTenant(r.Context(), boundsFromClaims(claims), func(tx pgx.Tx) error {
		var e error
		out, e = a.Documents.ListByMember(r.Context(), tx, id)
		return e
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if out == nil {
		out = []documents.Document{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"documents": out})
}

// handleIssueMemberDocument emite um certificado ou carta para o membro.
func (a *App) handleIssueMemberDocument(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	id := r.PathValue("id")
	var in struct {
		Kind   string `json:"kind"`
		Type   string `json:"type"`
		Notes  string `json:"notes"`
		City   string `json:"city"`
		Spouse string `json:"spouse"`
	}
	if err := readJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	if in.Kind != documents.KindCertificate && in.Kind != documents.KindLetter {
		writeErr(w, http.StatusBadRequest, "kind invalido (certificate|letter)")
		return
	}
	var doc *documents.Document
	err := a.Store.WithTenant(r.Context(), boundsFromClaims(claims), func(tx pgx.Tx) error {
		m, e := a.Members.Get(r.Context(), tx, id)
		if e != nil {
			return e
		}
		branchID, e := a.writeBranchID(r.Context(), tx, claims)
		if e != nil {
			return e
		}
		var church, churchDoc string
		_ = tx.QueryRow(r.Context(),
			`SELECT COALESCE(NULLIF(legal_name,''), name), COALESCE(cnpj,'') FROM tenants WHERE id = current_tenant()`).
			Scan(&church, &churchDoc)
		refPrefix := "CERT"
		if in.Kind == documents.KindLetter {
			refPrefix = "CARTA"
		}
		ref := documents.NewRef(refPrefix)
		content, _ := json.Marshal(map[string]any{
			"kind": in.Kind, "type": in.Type, "member": m.FullName,
			"member_doc": ptrStr(m.CPF), "birth_date": ptrStr(m.BirthDate),
			"baptism_date": ptrStr(m.BaptismDate), "marriage_date": ptrStr(m.MarriageDate),
			"spouse": in.Spouse, "church": church, "church_doc": churchDoc,
			"city": in.City, "issued_at": time.Now().Format("2006-01-02"),
			"body": strings.TrimSpace(in.Notes), "ref": ref,
		})
		title := documents.DocTitle(in.Kind, in.Type)
		doc, e = a.Documents.IssueMemberDocument(r.Context(), tx, claims.TenantID, branchID, id, in.Kind, title, ref, content)
		return e
	})
	if err != nil {
		if store.IsNotFound(err) {
			writeErr(w, http.StatusNotFound, "membro nao encontrado")
			return
		}
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, doc)
}

// handleRenderDocument renderiza o documento em HTML (certificado/carta/recibo).
func (a *App) handleRenderDocument(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	id := r.PathValue("id")
	var doc *documents.Document
	var tenant string
	err := a.Store.WithTenant(r.Context(), boundsFromClaims(claims), func(tx pgx.Tx) error {
		var e error
		doc, e = a.Documents.GetByID(r.Context(), tx, id)
		if e != nil {
			return e
		}
		tenant, _ = a.Documents.TenantName(r.Context(), tx)
		return nil
	})
	if err != nil {
		if store.IsNotFound(err) {
			writeErr(w, http.StatusNotFound, "documento nao encontrado")
			return
		}
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	var body string
	switch doc.Kind {
	case documents.KindCertificate:
		body, err = documents.RenderCertificateHTML(doc, tenant)
	case documents.KindLetter:
		body, err = documents.RenderLetterHTML(doc, tenant)
	case "receipt":
		body, err = documents.RenderReceiptHTML(doc, tenant)
	default:
		writeErr(w, http.StatusBadRequest, "documento sem visualizacao")
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("X-Robots-Tag", "noindex, nofollow")
	_, _ = w.Write([]byte(body))
}
