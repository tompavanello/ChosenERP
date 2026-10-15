package httpapi

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/jackc/pgx/v5"

	"chosenerp/internal/backup"
	"chosenerp/internal/store"
)

// ---- Backup por igreja (console da plataforma) ----

// handleListTenantBackups lista os backups de uma igreja.
func (a *App) handleListTenantBackups(w http.ResponseWriter, r *http.Request) {
	if _, ok := a.platformAdminClaims(w, r); !ok {
		return
	}
	id := r.PathValue("id")
	out, err := a.Backup.ListByTenant(r.Context(), id)
	if err != nil {
		writeServerError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"backups": out})
}

// handleCreateTenantBackup gera um backup agora. Com ?download=1 devolve o
// proprio arquivo; caso contrario devolve o metadado.
func (a *App) handleCreateTenantBackup(w http.ResponseWriter, r *http.Request) {
	claims, ok := a.platformAdminClaims(w, r)
	if !ok {
		return
	}
	id := r.PathValue("id")
	b, err := a.Backup.Create(r.Context(), id, "manual", claims.UserID)
	if err != nil {
		writeServerError(w, err)
		return
	}
	if r.URL.Query().Get("download") == "1" {
		a.serveBackupFile(w, r, b)
		return
	}
	writeJSON(w, http.StatusCreated, b)
}

// handleListBackups lista os backups mais recentes de todas as igrejas.
func (a *App) handleListBackups(w http.ResponseWriter, r *http.Request) {
	if _, ok := a.platformAdminClaims(w, r); !ok {
		return
	}
	out, err := a.Backup.ListAll(r.Context(), 200)
	if err != nil {
		writeServerError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"backups": out})
}

// handleDownloadBackup serve o arquivo .zip de um backup.
func (a *App) handleDownloadBackup(w http.ResponseWriter, r *http.Request) {
	if _, ok := a.platformAdminClaims(w, r); !ok {
		return
	}
	b, err := a.Backup.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		if store.IsNotFound(err) {
			writeErr(w, http.StatusNotFound, "backup nao encontrado")
			return
		}
		writeServerError(w, err)
		return
	}
	a.serveBackupFile(w, r, b)
}

func (a *App) serveBackupFile(w http.ResponseWriter, r *http.Request, b *backup.Backup) {
	path := a.Backup.Path(b)
	f, err := os.Open(path)
	if err != nil {
		writeErr(w, http.StatusNotFound, "arquivo de backup indisponivel")
		return
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		writeServerError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", `attachment; filename="`+filepath.Base(b.Filename)+`"`)
	http.ServeContent(w, r, b.Filename, info.ModTime(), f)
}

// handleDeleteBackup remove o arquivo e o metadado.
func (a *App) handleDeleteBackup(w http.ResponseWriter, r *http.Request) {
	if _, ok := a.platformAdminClaims(w, r); !ok {
		return
	}
	if err := a.Backup.Delete(r.Context(), r.PathValue("id")); err != nil {
		if store.IsNotFound(err) {
			writeErr(w, http.StatusNotFound, "backup nao encontrado")
			return
		}
		writeServerError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// maxBackupUpload limita o tamanho total do envio de um .zip de restore.
const maxBackupUpload int64 = 1 << 30 // 1 GiB

// handleRestoreTenant restaura um backup por id OU por upload de arquivo.
// Exige confirmacao explicita com o slug da igreja. Antes de restaurar, gera um
// backup de seguranca (pre_restore) para permitir desfazer.
func (a *App) handleRestoreTenant(w http.ResponseWriter, r *http.Request) {
	claims, ok := a.platformAdminClaims(w, r)
	if !ok {
		return
	}
	id := r.PathValue("id")

	// Limita o corpo total (ParseMultipartForm so limita a parte em memoria).
	r.Body = http.MaxBytesReader(w, r.Body, maxBackupUpload)
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		writeErr(w, http.StatusBadRequest, "envio invalido ou acima do limite")
		return
	}

	// Confirmacao pelo slug da igreja.
	slug, err := a.tenantSlug(r, id)
	if err != nil {
		if store.IsNotFound(err) {
			writeErr(w, http.StatusNotFound, "igreja nao encontrada")
			return
		}
		writeServerError(w, err)
		return
	}
	confirm := strings.TrimSpace(r.FormValue("confirm"))
	if !strings.EqualFold(confirm, slug) {
		writeErr(w, http.StatusBadRequest, "confirmacao invalida: informe o slug da igreja em confirm")
		return
	}

	// Resolve a origem do backup ANTES de gerar o backup de seguranca.
	var restorePath string
	backupID := strings.TrimSpace(r.FormValue("backup_id"))
	if backupID != "" {
		b, err := a.Backup.Get(r.Context(), backupID)
		if err != nil || b.TenantID != id {
			writeErr(w, http.StatusBadRequest, "backup_id invalido para esta igreja")
			return
		}
		restorePath = a.Backup.Path(b)
	} else {
		file, hdr, err := r.FormFile("file")
		if err != nil {
			writeErr(w, http.StatusBadRequest, "envie o arquivo .zip em file ou um backup_id")
			return
		}
		defer file.Close()
		if !strings.HasSuffix(strings.ToLower(hdr.Filename), ".zip") {
			writeErr(w, http.StatusBadRequest, "o arquivo deve ser um .zip de backup")
			return
		}
		tmpDir := filepath.Join(a.Backup.Dir, "uploads")
		if err := os.MkdirAll(tmpDir, 0o755); err != nil {
			writeServerError(w, err)
			return
		}
		tmp, err := os.CreateTemp(tmpDir, "restore-*.zip")
		if err != nil {
			writeServerError(w, err)
			return
		}
		tmpPath := tmp.Name()
		defer func() { _ = os.Remove(tmpPath) }()
		if _, err := io.Copy(tmp, io.LimitReader(file, maxBackupUpload+1)); err != nil {
			_ = tmp.Close()
			writeErr(w, http.StatusRequestEntityTooLarge, "falha ao salvar o envio (limite excedido?)")
			return
		}
		if info, err := tmp.Stat(); err == nil && info.Size() > maxBackupUpload {
			_ = tmp.Close()
			writeErr(w, http.StatusRequestEntityTooLarge, "arquivo de backup acima do limite")
			return
		}
		if err := tmp.Close(); err != nil {
			writeServerError(w, err)
			return
		}
		restorePath = tmpPath
	}

	// Backup de seguranca do estado atual (permite desfazer).
	safety, err := a.Backup.Create(r.Context(), id, "pre_restore", claims.UserID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "nao foi possivel gerar o backup de seguranca: "+err.Error())
		return
	}

	res, err := a.Backup.RestoreFile(r.Context(), id, restorePath)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "restore": res, "safety_backup_id": safety.ID})
}

func (a *App) tenantSlug(r *http.Request, tenantID string) (string, error) {
	var slug string
	err := a.Store.WithSystem(r.Context(), func(tx pgx.Tx) error {
		return tx.QueryRow(r.Context(), `SELECT slug FROM tenants WHERE id = $1::uuid`, tenantID).Scan(&slug)
	})
	return slug, err
}

// ---- Configuracao da rotina agendada ----

func (a *App) handleGetBackupSettings(w http.ResponseWriter, r *http.Request) {
	if _, ok := a.platformAdminClaims(w, r); !ok {
		return
	}
	s, err := a.Backup.GetSettings(r.Context())
	if err != nil {
		writeServerError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, s)
}

func (a *App) handleUpdateBackupSettings(w http.ResponseWriter, r *http.Request) {
	if _, ok := a.platformAdminClaims(w, r); !ok {
		return
	}
	var in struct {
		Enabled       bool `json:"enabled"`
		IntervalHours int  `json:"interval_hours"`
		RetentionDays int  `json:"retention_days"`
	}
	if err := readJSON(r, &in); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	s, err := a.Backup.UpdateSettings(r.Context(), backup.Settings{
		Enabled:       in.Enabled,
		IntervalHours: in.IntervalHours,
		RetentionDays: in.RetentionDays,
	})
	if err != nil {
		writeServerError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, s)
}
