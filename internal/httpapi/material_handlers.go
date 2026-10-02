package httpapi

import (
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/jackc/pgx/v5"

	"chosenerp/internal/materials"
	"chosenerp/internal/store"
)

// maxMaterialBytes limita o upload de material de estudo (25 MB).
const maxMaterialBytes = 25 << 20

// materialExts e a allowlist de extensoes de material -> MIME. A validacao e por
// extensao (nao por sniff) porque o DetectContentType nao reconhece OOXML
// (docx/xlsx/pptx sao ZIP) e devolveria "application/zip". O download passa por
// endpoint autenticado e nunca executa o arquivo.
var materialExts = map[string]string{
	".pdf":  "application/pdf",
	".doc":  "application/msword",
	".docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
	".ppt":  "application/vnd.ms-powerpoint",
	".pptx": "application/vnd.openxmlformats-officedocument.presentationml.presentation",
	".xls":  "application/vnd.ms-excel",
	".xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
	".odt":  "application/vnd.oasis.opendocument.text",
	".ods":  "application/vnd.oasis.opendocument.spreadsheet",
	".odp":  "application/vnd.oasis.opendocument.presentation",
	".txt":  "text/plain; charset=utf-8",
	".csv":  "text/csv; charset=utf-8",
	".png":  "image/png",
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".webp": "image/webp",
	".gif":  "image/gif",
	".mp3":  "audio/mpeg",
	".m4a":  "audio/mp4",
	".mp4":  "video/mp4",
	".webm": "video/webm",
	".zip":  "application/zip",
}

// handleListMaterials lista os materiais do escopo (admin/secretaria).
func (a *App) handleListMaterials(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	groupFilter := r.URL.Query().Get("group_id")
	b := boundsFromClaims(claims)
	var out []materials.Material
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		var err error
		out, err = a.Materials.List(r.Context(), tx, groupFilter)
		return err
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"materials": out})
}

// handleCreateMaterial cria um material de arquivo (multipart) ou de link (JSON).
func (a *App) handleCreateMaterial(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}

	var (
		in       materials.CreateInput
		diskPath string
		haveFile bool
	)
	if strings.Contains(r.Header.Get("Content-Type"), "application/json") {
		var body struct {
			Title       string  `json:"title"`
			Description *string `json:"description"`
			GroupID     *string `json:"group_id"`
			MinistryID  *string `json:"ministry_id"`
			URL         string  `json:"url"`
			IsPublished *bool   `json:"is_published"`
		}
		if err := readJSON(r, &body); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid body")
			return
		}
		if strings.TrimSpace(body.Title) == "" || !isHTTPURL(body.URL) {
			writeErr(w, http.StatusBadRequest, "titulo e URL (http/https) obrigatorios")
			return
		}
		pub := true
		if body.IsPublished != nil {
			pub = *body.IsPublished
		}
		in = materials.CreateInput{
			Title: strings.TrimSpace(body.Title), Description: body.Description,
			GroupID: body.GroupID, MinistryID: body.MinistryID,
			Kind: "link", URL: &body.URL, IsPublished: pub,
		}
	} else {
		// Upload de arquivo.
		r.Body = http.MaxBytesReader(w, r.Body, maxMaterialBytes)
		if err := r.ParseMultipartForm(8 << 20); err != nil {
			writeErr(w, http.StatusBadRequest, "arquivo muito grande (limite 25MB)")
			return
		}
		title := strings.TrimSpace(r.FormValue("title"))
		if title == "" {
			writeErr(w, http.StatusBadRequest, "titulo obrigatorio")
			return
		}
		file, header, err := r.FormFile("file")
		if err != nil {
			writeErr(w, http.StatusBadRequest, "arquivo nao enviado")
			return
		}
		defer file.Close()

		ext := strings.ToLower(filepath.Ext(header.Filename))
		mimeType, ok := materialExts[ext]
		if !ok {
			writeErr(w, http.StatusBadRequest, "formato nao suportado")
			return
		}
		if err := os.MkdirAll(a.Config.UploadDir, 0o755); err != nil {
			writeErr(w, http.StatusInternalServerError, "nao foi possivel criar diretorio de uploads")
			return
		}
		diskName, err := randomDiskName(ext)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, "erro ao gerar nome do arquivo")
			return
		}
		diskPath = filepath.Join(a.Config.UploadDir, diskName)

		size, err := writeUpload(diskPath, file)
		if err != nil {
			_ = os.Remove(diskPath)
			writeErr(w, http.StatusInternalServerError, "erro ao salvar o arquivo")
			return
		}
		haveFile = true

		original := filepath.Base(header.Filename)
		desc := strings.TrimSpace(r.FormValue("description"))
		groupID := strings.TrimSpace(r.FormValue("group_id"))
		ministryID := strings.TrimSpace(r.FormValue("ministry_id"))
		pub := r.FormValue("is_published") != "false"
		in = materials.CreateInput{
			Title:       title,
			Description: optString(desc),
			GroupID:     optString(groupID),
			MinistryID:  optString(ministryID),
			Kind:        "file",
			DiskName:    &diskName,
			FileName:    &original,
			FileSize:    &size,
			MimeType:    &mimeType,
			IsPublished: pub,
		}
	}

	b := boundsFromClaims(claims)
	var created *materials.Material
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		branchID, err := a.writeBranchID(r.Context(), tx, claims)
		if err != nil {
			return err
		}
		created, err = a.Materials.Create(r.Context(), tx, claims.TenantID, branchID, claims.UserID, in)
		return err
	})
	if err != nil {
		if haveFile {
			_ = os.Remove(diskPath)
		}
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

// handleDeleteMaterial remove o material e o arquivo do disco.
func (a *App) handleDeleteMaterial(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	id := r.PathValue("id")
	b := boundsFromClaims(claims)
	var diskName *string
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		m, err := a.Materials.Get(r.Context(), tx, id)
		if err != nil {
			return err
		}
		diskName = m.DiskName
		return a.Materials.Delete(r.Context(), tx, id)
	})
	if err != nil {
		if store.IsNotFound(err) {
			writeErr(w, http.StatusNotFound, "material nao encontrado")
			return
		}
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	removeMaterialFile(a.Config.UploadDir, diskName)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// handleDownloadMaterial serve o arquivo do material (autenticado + RLS). Nao
// reutiliza /api/v1/attachments porque aquele endpoint e publico.
func (a *App) handleDownloadMaterial(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	id := r.PathValue("id")
	b := boundsFromClaims(claims)
	var m *materials.Material
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		var err error
		m, err = a.Materials.Get(r.Context(), tx, id)
		return err
	})
	if err != nil {
		if store.IsNotFound(err) {
			writeErr(w, http.StatusNotFound, "material nao encontrado")
			return
		}
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if m.Kind != "file" || m.DiskName == nil || *m.DiskName == "" {
		writeErr(w, http.StatusNotFound, "material sem arquivo")
		return
	}
	name := filepath.Base(*m.DiskName)
	diskPath := filepath.Join(a.Config.UploadDir, name)
	if _, err := os.Stat(diskPath); err != nil {
		writeErr(w, http.StatusNotFound, "arquivo nao encontrado")
		return
	}
	downloadName := "material" + filepath.Ext(name)
	if m.FileName != nil && strings.TrimSpace(*m.FileName) != "" {
		downloadName = filepath.Base(*m.FileName)
	}
	ct := "application/octet-stream"
	if m.MimeType != nil && *m.MimeType != "" {
		ct = *m.MimeType
	}
	w.Header().Set("Content-Type", ct)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Cache-Control", "private, max-age=0")
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": downloadName}))
	http.ServeFile(w, r, diskPath)
}

// handleListMyMaterials lista os materiais visiveis para o proprio membro.
func (a *App) handleListMyMaterials(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	var out []materials.Material
	err := a.Store.WithTenant(r.Context(), boundsFromClaims(claims), func(tx pgx.Tx) error {
		m, e := a.memberForClaims(r.Context(), tx, claims)
		if e != nil {
			return e
		}
		out, e = a.Materials.ListForMember(r.Context(), tx, m.ID)
		return e
	})
	if err != nil {
		writeMemberErr(w, err)
		return
	}
	if out == nil {
		out = []materials.Material{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"materials": out})
}

// writeUpload grava o stream no disco e devolve o tamanho em bytes.
func writeUpload(path string, src io.Reader) (int64, error) {
	dst, err := os.Create(path)
	if err != nil {
		return 0, err
	}
	n, err := io.Copy(dst, src)
	closeErr := dst.Close()
	if err == nil {
		err = closeErr
	}
	return n, err
}

// removeMaterialFile apaga o arquivo do disco, garantindo que o caminho fique
// dentro do diretorio de uploads.
func removeMaterialFile(dir string, diskName *string) {
	if dir == "" || diskName == nil || *diskName == "" {
		return
	}
	path := filepath.Join(dir, filepath.Base(*diskName))
	if rel, err := filepath.Rel(dir, path); err != nil || strings.HasPrefix(rel, "..") {
		return
	}
	_ = os.Remove(path)
}

func optString(s string) *string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	return &s
}

func isHTTPURL(s string) bool {
	s = strings.TrimSpace(s)
	return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://")
}
