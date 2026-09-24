package httpapi

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/jackc/pgx/v5"

	"chosenerp/internal/store"
)

// Extensoes/imagens aceitas na foto do membro. A allowlist e por tipo REAL
// detectado (http.DetectContentType), nunca pela extensao enviada ou pelo
// Content-Type do multipart - ambos sao controlados pelo cliente.
var photoTypes = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
}

// handleUploadMemberPhoto grava a foto do membro no disco local e atualiza
// members.photo_url.
//
// Reaproveita o diretorio de uploads e o endpoint de leitura dos anexos do
// financeiro (GET /api/v1/attachments/{filename}), que ja serve arquivo por nome
// opaco sem exigir sessao. Isso e necessario porque a carteirinha publica
// (/api/v1/public/card/{token}) precisa exibir a foto para quem nao tem login:
// a URL do arquivo e a capability, o nome aleatorio e o que a protege.
func (a *App) handleUploadMemberPhoto(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	memberID := r.PathValue("id")

	// MaxBytesReader corta a leitura no limite antes de bufferizar o corpo -
	// ParseMultipartForm com maxMemory so limita a parte em memoria.
	r.Body = http.MaxBytesReader(w, r.Body, a.Config.MemberPhotoMaxBytes)
	if err := r.ParseMultipartForm(4 << 20); err != nil {
		writeErr(w, http.StatusBadRequest, "imagem muito grande")
		return
	}
	file, _, err := r.FormFile("file")
	if err != nil {
		writeErr(w, http.StatusBadRequest, "arquivo nao enviado")
		return
	}
	defer file.Close()

	// Detecta o tipo pelos primeiros 512 bytes (assinatura do arquivo).
	head := make([]byte, 512)
	n, err := io.ReadFull(file, head)
	if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
		writeErr(w, http.StatusBadRequest, "nao foi possivel ler o arquivo")
		return
	}
	head = head[:n]

	ct := strings.ToLower(strings.TrimSpace(strings.Split(http.DetectContentType(head), ";")[0]))
	ext, ok := photoTypes[ct]
	if !ok {
		writeErr(w, http.StatusBadRequest, "formato nao suportado (use JPG, PNG ou WEBP)")
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
	diskPath := filepath.Join(a.Config.UploadDir, diskName)

	dst, err := os.Create(diskPath)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "nao foi possivel salvar a imagem")
		return
	}
	// Escreve o cabecalho ja lido e o restante do stream.
	if _, err = dst.Write(head); err == nil {
		_, err = io.Copy(dst, file)
	}
	closeErr := dst.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		// Arquivo orfao: a transacao abaixo nem chegou a rodar.
		_ = os.Remove(diskPath)
		writeErr(w, http.StatusInternalServerError, "erro ao salvar a imagem")
		return
	}

	photoURL := "/api/v1/attachments/" + diskName

	b := boundsFromClaims(claims)
	var prev *string
	var m *memberPhotoDTO
	err = a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		atual, err := a.Members.Get(r.Context(), tx, memberID)
		if err != nil {
			return err
		}
		prev = atual.PhotoURL
		updated, err := a.Members.SetPhoto(r.Context(), tx, memberID, &photoURL)
		if err != nil {
			return err
		}
		m = &memberPhotoDTO{ID: updated.ID, PhotoURL: updated.PhotoURL}
		return nil
	})
	if err != nil {
		// Rollback: sem remover o arquivo, cada 404/403 deixaria lixo no disco.
		_ = os.Remove(diskPath)
		if store.IsNotFound(err) {
			writeErr(w, http.StatusNotFound, "member not found")
			return
		}
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}

	removeUploadIfOwned(a.Config.UploadDir, prev)
	writeJSON(w, http.StatusOK, m)
}

// handleDeleteMemberPhoto limpa a foto do membro.
func (a *App) handleDeleteMemberPhoto(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	memberID := r.PathValue("id")
	b := boundsFromClaims(claims)
	var prev *string
	err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
		atual, err := a.Members.Get(r.Context(), tx, memberID)
		if err != nil {
			return err
		}
		prev = atual.PhotoURL
		_, err = a.Members.SetPhoto(r.Context(), tx, memberID, nil)
		return err
	})
	if err != nil {
		if store.IsNotFound(err) {
			writeErr(w, http.StatusNotFound, "member not found")
			return
		}
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	removeUploadIfOwned(a.Config.UploadDir, prev)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// memberPhotoDTO e o recorte devolvido ao front apos mexer na foto: o objeto
// Member inteiro seria caro e o front so precisa saber onde a imagem ficou.
type memberPhotoDTO struct {
	ID       string  `json:"id"`
	PhotoURL *string `json:"photo_url,omitempty"`
}

// randomDiskName gera "<32 hex><ext>" - nome opaco, sem relacao com o nome
// original nem com o id do membro (nao vaza dado nenhum na URL publica).
func randomDiskName(ext string) (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b) + ext, nil
}

// removeUploadIfOwned apaga o arquivo anterior, mas SO se ele estiver dentro do
// diretorio de uploads e tiver o formato de nome que nos mesmos geramos. Sem
// essa checagem, um photo_url manipulado no banco (ou um valor legado apontando
// para outro caminho) viraria remocao de arquivo arbitrario.
func removeUploadIfOwned(dir string, url *string) {
	if url == nil || dir == "" {
		return
	}
	const prefix = "/api/v1/attachments/"
	if !strings.HasPrefix(*url, prefix) {
		return
	}
	name := filepath.Base(strings.TrimPrefix(*url, prefix))
	if !isOwnUploadName(name) {
		return
	}
	path := filepath.Join(dir, name)
	// filepath.Join limpa "..", entao confere que o resultado ficou dentro de dir.
	if rel, err := filepath.Rel(dir, path); err != nil || strings.HasPrefix(rel, "..") {
		return
	}
	_ = os.Remove(path)
}

// isOwnUploadName reconhece "<24+ hex><ext conhecida>".
func isOwnUploadName(name string) bool {
	ext := filepath.Ext(name)
	if _, ok := photoTypes[contentTypeForExt(ext)]; !ok {
		return false
	}
	stem := strings.TrimSuffix(name, ext)
	if len(stem) < 16 {
		return false
	}
	for _, c := range stem {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			return false
		}
	}
	return true
}

// contentTypeForExt inverte photoTypes (ext -> mime).
func contentTypeForExt(ext string) string {
	for ct, e := range photoTypes {
		if e == ext {
			return ct
		}
	}
	return ""
}
