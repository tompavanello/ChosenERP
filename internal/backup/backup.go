// Package backup implementa exportacao e restauracao dos DADOS COMPLETOS de
// uma igreja (tenant). O trabalho pesado de banco e feito por funcoes SQL
// (export_tenant/restore_tenant, migracao 000097); aqui ficam o empacotamento
// em .zip, a coleta dos arquivos de upload, a persistencia dos metadados e a
// rotina agendada.
package backup

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"chosenerp/internal/store"
)

const (
	// FormatVersion identifica a versao do formato do pacote. O restore recusa
	// pacotes com versao diferente para nao interpretar payloads incompativeis.
	FormatVersion = 1

	attachmentPrefix = "/api/v1/attachments/"

	manifestEntry = "manifest.json"
	dataEntry     = "data.json"
	filesDir      = "files/"

	// maxZipEntryBytes limita cada entrada do pacote (anti "zip bomb").
	maxZipEntryBytes int64 = 512 << 20
)

// Manifest descreve o conteudo do pacote.
type Manifest struct {
	FormatVersion int               `json:"format_version"`
	AppVersion    string            `json:"app_version,omitempty"`
	TenantID      string            `json:"tenant_id"`
	TenantSlug    string            `json:"tenant_slug"`
	TenantName    string            `json:"tenant_name"`
	GeneratedAt   time.Time         `json:"generated_at"`
	Kind          string            `json:"kind"`
	DataSHA256    string            `json:"data_sha256"`
	Files         []FileEntry       `json:"files"`
	Counts        map[string]int    `json:"counts"`
	Extra         map[string]string `json:"extra,omitempty"`
}

// FileEntry descreve um arquivo de upload empacotado.
type FileEntry struct {
	Name   string `json:"name"`
	Size   int64  `json:"size"`
	SHA256 string `json:"sha256"`
}

// Backup e o metadado persistido de um backup.
type Backup struct {
	ID         string         `json:"id"`
	TenantID   string         `json:"tenant_id"`
	Filename   string         `json:"filename"`
	SizeBytes  int64          `json:"size_bytes"`
	Kind       string         `json:"kind"`
	Status     string         `json:"status"`
	Error      string         `json:"error,omitempty"`
	Counts     map[string]int `json:"counts"`
	CreatedBy  string         `json:"created_by,omitempty"`
	CreatedAt  time.Time      `json:"created_at"`
}

// Settings e a configuracao global da rotina agendada.
type Settings struct {
	Enabled       bool      `json:"enabled"`
	IntervalHours int       `json:"interval_hours"`
	RetentionDays int       `json:"retention_days"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// Manager orquestra export/restore/persistencia.
type Manager struct {
	Store     *store.Store
	Dir       string
	UploadDir string
	// AppVersion e gravada no manifesto (informativo).
	AppVersion string
}

func (m *Manager) dir() string {
	if m.Dir == "" {
		return "/tmp/chosenerp/backups"
	}
	return m.Dir
}

// ---- Settings ----

func (m *Manager) GetSettings(ctx context.Context) (*Settings, error) {
	var s Settings
	err := m.Store.WithSystem(ctx, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `
			SELECT enabled, interval_hours, retention_days, updated_at
			FROM backup_settings WHERE id = true`).
			Scan(&s.Enabled, &s.IntervalHours, &s.RetentionDays, &s.UpdatedAt)
	})
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (m *Manager) UpdateSettings(ctx context.Context, s Settings) (*Settings, error) {
	if s.IntervalHours <= 0 {
		s.IntervalHours = 24
	}
	if s.RetentionDays < 0 {
		s.RetentionDays = 0
	}
	err := m.Store.WithSystem(ctx, func(tx pgx.Tx) error {
		_, e := tx.Exec(ctx, `
			UPDATE backup_settings
			SET enabled = $1, interval_hours = $2, retention_days = $3, updated_at = now()
			WHERE id = true`, s.Enabled, s.IntervalHours, s.RetentionDays)
		return e
	})
	if err != nil {
		return nil, err
	}
	return m.GetSettings(ctx)
}

// ---- Consulta de metadados ----

func (m *Manager) ListByTenant(ctx context.Context, tenantID string) ([]Backup, error) {
	out := []Backup{}
	err := m.Store.WithSystem(ctx, func(tx pgx.Tx) error {
		rows, e := tx.Query(ctx, `
			SELECT id::text, target_tenant_id::text, filename, size_bytes, kind, status,
			       COALESCE(error, ''), counts, COALESCE(created_by::text, ''), created_at
			FROM tenant_backups
			WHERE target_tenant_id = $1
			ORDER BY created_at DESC`, tenantID)
		if e != nil {
			return e
		}
		defer rows.Close()
		for rows.Next() {
			b, e := scanBackup(rows)
			if e != nil {
				return e
			}
			out = append(out, *b)
		}
		return rows.Err()
	})
	return out, err
}

func (m *Manager) ListAll(ctx context.Context, limit int) ([]Backup, error) {
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	out := []Backup{}
	err := m.Store.WithSystem(ctx, func(tx pgx.Tx) error {
		rows, e := tx.Query(ctx, `
			SELECT b.id::text, b.target_tenant_id::text, b.filename, b.size_bytes, b.kind, b.status,
			       COALESCE(b.error, ''), b.counts, COALESCE(b.created_by::text, ''), b.created_at
			FROM tenant_backups b
			ORDER BY b.created_at DESC
			LIMIT $1`, limit)
		if e != nil {
			return e
		}
		defer rows.Close()
		for rows.Next() {
			b, e := scanBackup(rows)
			if e != nil {
				return e
			}
			out = append(out, *b)
		}
		return rows.Err()
	})
	return out, err
}

func (m *Manager) Get(ctx context.Context, id string) (*Backup, error) {
	var b *Backup
	err := m.Store.WithSystem(ctx, func(tx pgx.Tx) error {
		rows, e := tx.Query(ctx, `
			SELECT id::text, target_tenant_id::text, filename, size_bytes, kind, status,
			       COALESCE(error, ''), counts, COALESCE(created_by::text, ''), created_at
			FROM tenant_backups WHERE id = $1::uuid`, id)
		if e != nil {
			return e
		}
		defer rows.Close()
		if rows.Next() {
			bb, e := scanBackup(rows)
			if e != nil {
				return e
			}
			b = bb
		}
		return rows.Err()
	})
	if err != nil {
		return nil, err
	}
	if b == nil {
		return nil, pgx.ErrNoRows
	}
	return b, nil
}

func scanBackup(rows pgx.Rows) (*Backup, error) {
	var b Backup
	var countsRaw []byte
	if err := rows.Scan(&b.ID, &b.TenantID, &b.Filename, &b.SizeBytes, &b.Kind, &b.Status,
		&b.Error, &countsRaw, &b.CreatedBy, &b.CreatedAt); err != nil {
		return nil, err
	}
	b.Counts = map[string]int{}
	if len(countsRaw) > 0 {
		_ = json.Unmarshal(countsRaw, &b.Counts)
	}
	return &b, nil
}

// Path devolve o caminho absoluto do arquivo de um backup.
func (m *Manager) Path(b *Backup) string { return filepath.Join(m.dir(), b.Filename) }

// Delete remove o arquivo e o metadado.
func (m *Manager) Delete(ctx context.Context, id string) error {
	b, err := m.Get(ctx, id)
	if err != nil {
		return err
	}
	_ = os.Remove(m.Path(b))
	return m.Store.WithSystem(ctx, func(tx pgx.Tx) error {
		_, e := tx.Exec(ctx, `DELETE FROM tenant_backups WHERE id = $1::uuid`, id)
		return e
	})
}

// CleanupExpired remove backups mais antigos que retention_days (0 desliga).
func (m *Manager) CleanupExpired(ctx context.Context, retentionDays int) (int, error) {
	if retentionDays <= 0 {
		return 0, nil
	}
	var names []string
	err := m.Store.WithSystem(ctx, func(tx pgx.Tx) error {
		rows, e := tx.Query(ctx, `
			DELETE FROM tenant_backups
			WHERE created_at < now() - make_interval(days => $1)
			RETURNING filename`, retentionDays)
		if e != nil {
			return e
		}
		defer rows.Close()
		for rows.Next() {
			var n string
			if e := rows.Scan(&n); e != nil {
				return e
			}
			names = append(names, n)
		}
		return rows.Err()
	})
	if err != nil {
		return 0, err
	}
	for _, n := range names {
		_ = os.Remove(filepath.Join(m.dir(), filepath.Base(n)))
	}
	return len(names), nil
}

// ---- Export ----

// Create gera o pacote do tenant e grava o metadado. kind: manual|scheduled|pre_restore.
func (m *Manager) Create(ctx context.Context, tenantID, kind, createdBy string) (*Backup, error) {
	if kind == "" {
		kind = "manual"
	}
	if err := os.MkdirAll(m.dir(), 0o755); err != nil {
		return nil, fmt.Errorf("backups: criar diretorio: %w", err)
	}

	meta, payload, err := m.collect(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	files, err := m.collectFiles(payload, meta.LogoURL, meta.FaviconURL)
	if err != nil {
		return nil, err
	}

	dataBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	sum := sha256.Sum256(dataBytes)

	counts := countRows(payload)
	fileEntries := make([]FileEntry, 0, len(files))
	var totalFiles int64

	manifest := Manifest{
		FormatVersion: FormatVersion,
		AppVersion:    m.AppVersion,
		TenantID:      tenantID,
		TenantSlug:    meta.Slug,
		TenantName:    meta.Name,
		GeneratedAt:   time.Now().UTC(),
		Kind:          kind,
		DataSHA256:    hex.EncodeToString(sum[:]),
		Counts:        counts,
	}

	// Nome do arquivo: slug + timestamp UTC.
	ts := time.Now().UTC().Format("20060102-150405")
	filename := fmt.Sprintf("%s_%s.zip", sanitizeSlug(meta.Slug), ts)
	if kind == "pre_restore" {
		filename = fmt.Sprintf("%s_%s_pre.zip", sanitizeSlug(meta.Slug), ts)
	}

	tmp := filepath.Join(m.dir(), filename+".tmp")
	f, err := os.Create(tmp)
	if err != nil {
		return nil, err
	}

	zw := zip.NewWriter(f)
	// data.json
	if err := writeZipEntry(zw, dataEntry, dataBytes); err != nil {
		_ = f.Close()
		_ = os.Remove(tmp)
		return nil, err
	}
	// arquivos de upload
	for _, name := range files {
		src := filepath.Join(m.UploadDir, name)
		size, sha, err := addFileToZip(zw, src, name)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			_ = f.Close()
			_ = os.Remove(tmp)
			return nil, err
		}
		fileEntries = append(fileEntries, FileEntry{Name: name, Size: size, SHA256: sha})
		totalFiles += size
	}
	// manifest por ultimo (ja com a lista de arquivos)
	manifest.Files = fileEntries
	manifestBytes, _ := json.MarshalIndent(manifest, "", "  ")
	if err := writeZipEntry(zw, manifestEntry, manifestBytes); err != nil {
		_ = f.Close()
		_ = os.Remove(tmp)
		return nil, err
	}
	if err := zw.Close(); err != nil {
		_ = f.Close()
		_ = os.Remove(tmp)
		return nil, err
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(tmp)
		return nil, err
	}

	final := filepath.Join(m.dir(), filename)
	if err := os.Rename(tmp, final); err != nil {
		_ = os.Remove(tmp)
		return nil, err
	}
	info, _ := os.Stat(final)
	var sizeBytes int64
	if info != nil {
		sizeBytes = info.Size()
	}
	_ = totalFiles

	b, err := m.persist(ctx, tenantID, filename, sizeBytes, kind, counts, createdBy)
	if err != nil {
		return nil, err
	}
	return b, nil
}

type tenantMeta struct {
	Name       string
	Slug       string
	LogoURL    string
	FaviconURL string
}

func (m *Manager) collect(ctx context.Context, tenantID string) (*tenantMeta, any, error) {
	meta := &tenantMeta{}
	var payload any
	err := m.Store.WithSystem(ctx, func(tx pgx.Tx) error {
		if e := tx.QueryRow(ctx, `
			SELECT name, slug, COALESCE(logo_url, ''), COALESCE(favicon_url, '')
			FROM tenants WHERE id = $1::uuid`, tenantID).
			Scan(&meta.Name, &meta.Slug, &meta.LogoURL, &meta.FaviconURL); e != nil {
			return e
		}
		var raw []byte
		if e := tx.QueryRow(ctx, `SELECT export_tenant($1::uuid)::text`, tenantID).Scan(&raw); e != nil {
			return e
		}
		return json.Unmarshal(raw, &payload)
	})
	if err != nil {
		return nil, nil, err
	}
	if payload == nil {
		return nil, nil, errors.New("backup: tenant sem dados")
	}
	return meta, payload, nil
}

func (m *Manager) collectFiles(payload any, extraURLs ...string) ([]string, error) {
	set := map[string]struct{}{}
	walkForFiles(payload, set)
	for _, u := range extraURLs {
		collectAttachment(u, set)
	}
	out := make([]string, 0, len(set))
	for n := range set {
		out = append(out, n)
	}
	sort.Strings(out)
	return out, nil
}

func walkForFiles(v any, set map[string]struct{}) {
	switch t := v.(type) {
	case map[string]any:
		for k, val := range t {
			if k == "disk_name" {
				if s, ok := val.(string); ok && s != "" {
					set[filepath.Base(s)] = struct{}{}
				}
			}
			walkForFiles(val, set)
		}
	case []any:
		for _, e := range t {
			walkForFiles(e, set)
		}
	case string:
		collectAttachment(t, set)
	}
}

func collectAttachment(s string, set map[string]struct{}) {
	i := strings.Index(s, attachmentPrefix)
	if i < 0 {
		return
	}
	name := strings.TrimSpace(s[i+len(attachmentPrefix):])
	if name == "" {
		return
	}
	set[filepath.Base(name)] = struct{}{}
}

func countRows(payload any) map[string]int {
	counts := map[string]int{}
	obj, ok := payload.(map[string]any)
	if !ok {
		return counts
	}
	for _, group := range []string{"tables", "children"} {
		g, ok := obj[group].(map[string]any)
		if !ok {
			continue
		}
		total := 0
		for name, rows := range g {
			if arr, ok := rows.([]any); ok {
				counts[name] = len(arr)
				total += len(arr)
			}
		}
		counts["_"+group] = total
	}
	if arr, ok := obj["users"].([]any); ok {
		counts["users"] = len(arr)
	}
	if arr, ok := obj["role_permissions"].([]any); ok {
		counts["role_permissions"] = len(arr)
	}
	return counts
}

func writeZipEntry(zw *zip.Writer, name string, data []byte) error {
	w, err := zw.Create(name)
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}

func addFileToZip(zw *zip.Writer, src, name string) (int64, string, error) {
	f, err := os.Open(src)
	if err != nil {
		return 0, "", err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return 0, "", err
	}
	w, err := zw.Create(filesDir + name)
	if err != nil {
		return 0, "", err
	}
	h := sha256.New()
	if _, err := io.Copy(io.MultiWriter(w, h), f); err != nil {
		return 0, "", err
	}
	return info.Size(), hex.EncodeToString(h.Sum(nil)), nil
}

func (m *Manager) persist(ctx context.Context, tenantID, filename string, size int64, kind string, counts map[string]int, createdBy string) (*Backup, error) {
	countsJSON, _ := json.Marshal(counts)
	var id string
	var created time.Time
	err := m.Store.WithSystem(ctx, func(tx pgx.Tx) error {
		var cb any
		if createdBy != "" {
			cb = createdBy
		}
		return tx.QueryRow(ctx, `
			INSERT INTO tenant_backups (target_tenant_id, filename, size_bytes, kind, status, counts, created_by)
			VALUES ($1::uuid, $2, $3, $4, 'ready', $5::jsonb, $6)
			RETURNING id::text, created_at`,
			tenantID, filename, size, kind, string(countsJSON), cb).Scan(&id, &created)
	})
	if err != nil {
		return nil, err
	}
	return &Backup{
		ID: id, TenantID: tenantID, Filename: filename, SizeBytes: size,
		Kind: kind, Status: "ready", Counts: counts, CreatedBy: createdBy, CreatedAt: created,
	}, nil
}

// ---- Restore ----

// RestoreBytes restaura a partir do conteudo do .zip em memoria.
func (m *Manager) RestoreBytes(ctx context.Context, targetTenantID string, data []byte) (*RestoreResult, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("pacote invalido: %w", err)
	}
	return m.restore(ctx, targetTenantID, zr)
}

// RestoreFile restaura a partir de um arquivo .zip no disco.
func (m *Manager) RestoreFile(ctx context.Context, targetTenantID, path string) (*RestoreResult, error) {
	zr, err := zip.OpenReader(path)
	if err != nil {
		return nil, fmt.Errorf("pacote invalido: %w", err)
	}
	defer zr.Close()
	return m.restore(ctx, targetTenantID, &zr.Reader)
}

// RestoreResult resume o que foi restaurado.
type RestoreResult struct {
	TenantID string         `json:"tenant_id"`
	Counts   map[string]int `json:"counts"`
	Files    int            `json:"files"`
}

func (m *Manager) restore(ctx context.Context, targetTenantID string, zr *zip.Reader) (*RestoreResult, error) {
	manifest, dataBytes, fileReaders, err := readZip(zr)
	if err != nil {
		return nil, err
	}
	defer func() {
		for _, rc := range fileReaders {
			_ = rc.Close()
		}
	}()
	if manifest.FormatVersion != FormatVersion {
		return nil, fmt.Errorf("versao de backup incompativel (pacote=%d, esperado=%d)", manifest.FormatVersion, FormatVersion)
	}
	if manifest.TenantID != "" && manifest.TenantID != targetTenantID {
		return nil, errors.New("o backup pertence a outra igreja")
	}
	sum := sha256.Sum256(dataBytes)
	if manifest.DataSHA256 != "" && hex.EncodeToString(sum[:]) != manifest.DataSHA256 {
		return nil, errors.New("checksum do backup nao confere (arquivo corrompido)")
	}

	// Restore no banco (substituicao total do tenant).
	var result RestoreResult
	err = m.Store.WithSystem(ctx, func(tx pgx.Tx) error {
		var raw []byte
		if e := tx.QueryRow(ctx, `SELECT restore_tenant($1::uuid, $2::jsonb)::text`, targetTenantID, string(dataBytes)).Scan(&raw); e != nil {
			return e
		}
		_ = json.Unmarshal(raw, &result)
		return nil
	})
	if err != nil {
		return nil, err
	}

	// Devolve os arquivos de upload.
	count := 0
	for _, name := range manifestFileNames(manifest, fileReaders) {
		rc, ok := fileReaders[filesDir+name]
		if !ok {
			continue
		}
		if err := m.writeUploadFile(name, rc); err != nil {
			log.Printf("backup: falha ao restaurar arquivo %q: %v", name, err)
			continue
		}
		count++
	}
	result.Files = count
	if result.TenantID == "" {
		result.TenantID = targetTenantID
	}
	if result.Counts == nil {
		result.Counts = map[string]int{}
	}
	return &result, nil
}

// readZip le manifest/data e mantem os readers de arquivos abertos. Em caso de
// erro, fecha tudo o que ja foi aberto. Limita o tamanho de cada entrada para
// evitar "zip bombs".
func readZip(zr *zip.Reader) (*Manifest, []byte, map[string]io.ReadCloser, error) {
	var manifest *Manifest
	var data []byte
	files := map[string]io.ReadCloser{}
	closeAll := func() {
		for _, rc := range files {
			_ = rc.Close()
		}
	}
	for _, f := range zr.File {
		switch {
		case f.Name == manifestEntry:
			b, err := readZipFile(f)
			if err != nil {
				closeAll()
				return nil, nil, nil, err
			}
			var m Manifest
			if err := json.Unmarshal(b, &m); err != nil {
				closeAll()
				return nil, nil, nil, fmt.Errorf("manifest invalido: %w", err)
			}
			manifest = &m
		case f.Name == dataEntry:
			b, err := readZipFile(f)
			if err != nil {
				closeAll()
				return nil, nil, nil, err
			}
			data = b
		case strings.HasPrefix(f.Name, filesDir):
			rc, err := f.Open()
			if err != nil {
				closeAll()
				return nil, nil, nil, err
			}
			files[f.Name] = rc
		}
	}
	if manifest == nil || data == nil {
		closeAll()
		return nil, nil, nil, errors.New("pacote incompleto (manifest/data ausentes)")
	}
	return manifest, data, files, nil
}

func readZipFile(f *zip.File) ([]byte, error) {
	rc, err := f.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	return readLimited(rc, maxZipEntryBytes)
}

// readLimited le no maximo limit bytes; se a fonte tiver mais, devolve erro.
func readLimited(r io.Reader, limit int64) ([]byte, error) {
	b, err := io.ReadAll(io.LimitReader(r, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(b)) > limit {
		return nil, errors.New("entrada do backup excede o limite de tamanho")
	}
	return b, nil
}

func manifestFileNames(m *Manifest, files map[string]io.ReadCloser) []string {
	out := make([]string, 0, len(files))
	for k := range files {
		out = append(out, strings.TrimPrefix(k, filesDir))
	}
	sort.Strings(out)
	return out
}

func (m *Manager) writeUploadFile(name string, rc io.ReadCloser) error {
	defer rc.Close()
	base := filepath.Base(name)
	if base == "" || base == "." || base == string(filepath.Separator) {
		return errors.New("nome de arquivo invalido")
	}
	destDir := m.UploadDir
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return err
	}
	dest := filepath.Join(destDir, base)
	// Confere que o destino permaneceu dentro do diretorio de uploads.
	if !strings.HasPrefix(dest, filepath.Clean(destDir)+string(filepath.Separator)) {
		return errors.New("caminho de arquivo invalido")
	}
	tmp := dest + ".restore.tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	if _, err := io.Copy(f, rc); err != nil {
		_ = f.Close()
		_ = os.Remove(tmp)
		return err
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, dest)
}

func sanitizeSlug(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return "igreja"
	}
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '-':
			b.WriteRune(r)
		default:
			b.WriteByte('-')
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return "igreja"
	}
	return out
}
