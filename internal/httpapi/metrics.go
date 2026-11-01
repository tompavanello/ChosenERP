package httpapi

import (
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Métricas HTTP em memoria, no formato de texto do Prometheus. Evitam uma
// dependencia nova e cobrem o essencial: taxa por rota/status, latencia
// (histograma) e estado do pool de conexoes.
type httpMetrics struct {
	mu        sync.Mutex
	requests  map[requestKey]int64
	durations map[string]*durationHist
}

type requestKey struct {
	method string
	status string
	route  string
}

type durationHist struct {
	buckets []int64 // cumulativo por faixa
	sum     float64
	count   int64
}

// faixas em segundos; a ultima e +Inf.
var durationBuckets = []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10}

func newHTTPMetrics() *httpMetrics {
	return &httpMetrics{
		requests:  make(map[requestKey]int64),
		durations: make(map[string]*durationHist),
	}
}

func (m *httpMetrics) observe(method, status, route string, d time.Duration) {
	if m == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	m.requests[requestKey{method: method, status: status, route: route}]++

	h := m.durations[route]
	if h == nil {
		h = &durationHist{buckets: make([]int64, len(durationBuckets)+1)}
		m.durations[route] = h
	}
	secs := d.Seconds()
	h.sum += secs
	h.count++
	idx := len(durationBuckets) // +Inf
	for i, b := range durationBuckets {
		if secs <= b {
			idx = i
			break
		}
	}
	h.buckets[idx]++
}

// writeTo emite as linhas Prometheus. Chamado por /metrics.
func (m *httpMetrics) writeTo(w *strings.Builder) {
	if m == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	keys := make([]requestKey, 0, len(m.requests))
	for k := range m.requests {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].route != keys[j].route {
			return keys[i].route < keys[j].route
		}
		if keys[i].method != keys[j].method {
			return keys[i].method < keys[j].method
		}
		return keys[i].status < keys[j].status
	})
	for _, k := range keys {
		w.WriteString(`chosenerp_http_requests_total{method="` + k.method +
			`",status="` + k.status + `",route="` + k.route + `"} ` +
			strconv.FormatInt(m.requests[k], 10) + "\n")
	}

	routes := make([]string, 0, len(m.durations))
	for r := range m.durations {
		routes = append(routes, r)
	}
	sort.Strings(routes)
	for _, r := range routes {
		h := m.durations[r]
		for i, b := range durationBuckets {
			w.WriteString(`chosenerp_http_request_duration_seconds_bucket{route="` + r +
				`",le="` + strconv.FormatFloat(b, 'g', -1, 64) + `"} ` +
				strconv.FormatInt(h.buckets[i], 10) + "\n")
		}
		w.WriteString(`chosenerp_http_request_duration_seconds_bucket{route="` + r +
			`",le="+Inf"} ` + strconv.FormatInt(h.buckets[len(durationBuckets)], 10) + "\n")
		w.WriteString(`chosenerp_http_request_duration_seconds_sum{route="` + r + `"} ` +
			strconv.FormatFloat(h.sum, 'f', -1, 64) + "\n")
		w.WriteString(`chosenerp_http_request_duration_seconds_count{route="` + r + `"} ` +
			strconv.FormatInt(h.count, 10) + "\n")
	}
}

// statusWriter captura o codigo de status sem alterar a resposta.
type statusWriter struct {
	http.ResponseWriter
	status int
	wrote  bool
}

func (s *statusWriter) WriteHeader(code int) {
	if !s.wrote {
		s.status = code
		s.wrote = true
	}
	s.ResponseWriter.WriteHeader(code)
}

func (s *statusWriter) Write(b []byte) (int, error) {
	if !s.wrote {
		s.status = http.StatusOK
		s.wrote = true
	}
	return s.ResponseWriter.Write(b)
}

var (
	uuidPathRe = regexp.MustCompile(`[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}`)
	numPathRe  = regexp.MustCompile(`/[0-9]+`)
)

// normalizeRoute reduz a cardinalidade (ids viram :id) para as labels.
func normalizeRoute(path string) string {
	if path == "" {
		return "/"
	}
	p := uuidPathRe.ReplaceAllString(path, ":id")
	p = numPathRe.ReplaceAllString(p, "/:n")
	return p
}

// withMetrics envolve o mux e registra cada request.
func (a *App) withMetrics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		start := time.Now()
		next.ServeHTTP(sw, r)
		a.Metrics.observe(r.Method, strconv.Itoa(sw.status), normalizeRoute(r.URL.Path), time.Since(start))
	})
}
