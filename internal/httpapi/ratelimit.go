package httpapi

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Limite das rotas públicas (sem auth), por IP.
//
// 60/min é folgado para o uso real — a página da carteirinha faz três chamadas
// (JSON, HTML de impressão e foto) — e apertado o bastante para desencorajar
// varredura de tokens e raspagem de uma lista de carteirinhas.
const (
	publicRateMax    = 60
	publicRateJanela = time.Minute
	// Teto de IPs guardados; acima disso as janelas vencidas são podadas.
	publicRateMaxIPs = 4096
)

type contadorIP struct {
	inicio time.Time
	n      int
}

// limitadorPublico é um limitador de janela fixa por IP. Estado em memória, de
// propósito: são poucas rotas, o volume é baixo, e um Redis só para isso seria
// mais peça móvel do que o problema pede.
type limitadorPublico struct {
	mu       sync.Mutex
	contador map[string]*contadorIP
}

func newLimitadorPublico() *limitadorPublico {
	return &limitadorPublico{contador: make(map[string]*contadorIP)}
}

func (l *limitadorPublico) permitir(ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	agora := time.Now()
	c, ok := l.contador[ip]
	if !ok || agora.Sub(c.inicio) > publicRateJanela {
		if len(l.contador) >= publicRateMaxIPs {
			for k, v := range l.contador {
				if agora.Sub(v.inicio) > publicRateJanela {
					delete(l.contador, k)
				}
			}
		}
		l.contador[ip] = &contadorIP{inicio: agora, n: 1}
		return true
	}
	c.n++
	return c.n <= publicRateMax
}

// ipDoCliente extrai o IP do visitante considerando o proxy do túnel.
//
// CF-Connecting-IP vem primeiro porque é o único que o Cloudflare escreve e
// sobrescreve — o RemoteAddr atrás do túnel é sempre o do cloudflared, então
// usá-lo colocaria todos os visitantes no mesmo balde. Ressalva honesta:
// X-Forwarded-For e o próprio CF-Connecting-IP são falsificáveis por quem
// alcança a API direto (a 38080 escuta em 0.0.0.0), então isto é um dissuasor
// contra abuso casual, não uma fronteira de segurança.
func ipDoCliente(r *http.Request) string {
	if ip := strings.TrimSpace(r.Header.Get("CF-Connecting-IP")); ip != "" {
		return ip
	}
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if i := strings.IndexByte(xff, ','); i > 0 {
			return strings.TrimSpace(xff[:i])
		}
		return strings.TrimSpace(xff)
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// publico envolve as rotas sem autenticação: marca a resposta como não
// indexável e aplica o limite por IP. O noindex é redundante com o robots.txt
// do webadmin de propósito — o token da carteirinha não pode acabar num índice
// de busca.
func (a *App) publico(next http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Robots-Tag", "noindex, nofollow")
		if !a.Limitador.permitir(ipDoCliente(r)) {
			writeErr(w, http.StatusTooManyRequests, "muitas requisições")
			return
		}
		next(w, r)
	})
}
