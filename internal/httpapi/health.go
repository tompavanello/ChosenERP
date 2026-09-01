package httpapi

import (
	"net/http"
	"runtime"
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

func (a *App) handleMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	write := func(name string, v int64) {
		_, _ = w.Write([]byte(name + " " + int64ToStr(v) + "\n"))
	}
	write("chosenerp_mem_alloc_bytes", int64(ms.Alloc))
	write("chosenerp_gc_count", int64(ms.NumGC))
	write("chosenerp_goroutines", int64(runtime.NumGoroutine()))
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
