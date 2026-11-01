package httpapi

import (
	"context"
	"net/http"
	"runtime"
	"strconv"
	"strings"
	"time"
)

func (a *App) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status":  "ok",
		"service": "chosenerp-api",
		"version": "0.1.0",
		"time":    time.Now().UTC().Format(time.RFC3339),
	})
}

// handleReady indica se a API esta pronta para receber trafego (o banco
// responde). Diferente de /healthz, que e liveness puro.
func (a *App) handleReady(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	if err := a.Store.Pool().Ping(ctx); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{
			"status": "unavailable",
			"db":     "down",
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ready", "db": "up"})
}

func (a *App) handleMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4")

	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)

	var b strings.Builder
	write := func(name string, v int64) {
		b.WriteString(name)
		b.WriteByte(' ')
		b.WriteString(strconv.FormatInt(v, 10))
		b.WriteByte('\n')
	}
	write("chosenerp_mem_alloc_bytes", int64(ms.Alloc))
	write("chosenerp_gc_count", int64(ms.NumGC))
	write("chosenerp_goroutines", int64(runtime.NumGoroutine()))

	// Estado do pool de conexoes (saturação aparece aqui antes de virar 500).
	stat := a.Store.Pool().Stat()
	write("chosenerp_db_pool_max_conns", int64(stat.MaxConns()))
	write("chosenerp_db_pool_total_conns", int64(stat.TotalConns()))
	write("chosenerp_db_pool_idle_conns", int64(stat.IdleConns()))
	write("chosenerp_db_pool_acquired_conns", int64(stat.AcquiredConns()))

	a.Metrics.writeTo(&b)
	_, _ = w.Write([]byte(b.String()))
}

func int64ToStr(v int64) string {
	if v == 0 {
		return "0"
	}
	neg := v < 0
	if neg {
		v = -v
	}
	var b [20]byte
	i := len(b)
	for v > 0 {
		i--
		b[i] = byte('0' + v%10)
		v /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}
