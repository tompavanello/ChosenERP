package httpapi

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Limite das rotas publicas (sem auth), por IP.
//
// 60/min e folgado para o uso real - a pagina da carteirinha faz tres chamadas
// (JSON, HTML de impressao e foto) - e apertado o bastante para desencorajar
// varredura de tokens e raspagem de uma lista de carteirinhas.
const (
	publicRateMax    = 60
	publicRateJanela = time.Minute
	// Teto de IPs guardados; acima disso as janelas vencidas sao podadas.
	publicRateMaxIPs = 4096
)

type contadorIP struct {
	inicio time.Time
	n      int
}

// limitadorPublico e um limitador de janela fixa por IP. Estado em memoria, de
// proposito: sao poucas rotas, o volume e baixo, e um Redis so para isso seria
// mais peca movel do que o problema pede.
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

// ipDoCliente extrai o IP do visitante considerando o proxy do tunel.
//
// CF-Connecting-IP vem primeiro porque e o unico que o Cloudflare escreve e
// sobrescreve - o RemoteAddr atras do tunel e sempre o do cloudflared, entao
// usa-lo colocaria todos os visitantes no mesmo balde. Ressalva honesta:
// X-Forwarded-For e o proprio CF-Connecting-IP sao falsificaveis por quem
// alcanca a API direto (a 38080 escuta em 0.0.0.0), entao isto e um dissuasor
// contra abuso casual, nao uma fronteira de seguranca.
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

// publico envolve as rotas sem autenticacao: marca a resposta como nao
// indexavel e aplica o limite por IP. O noindex e redundante com o robots.txt
// do webadmin de proposito - o token da carteirinha nao pode acabar num indice
// de busca.
func (a *App) publico(next http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Robots-Tag", "noindex, nofollow")
		if !a.Limitador.permitir(ipDoCliente(r)) {
			writeErr(w, http.StatusTooManyRequests, "muitas requisicoes")
			return
		}
		next(w, r)
	})
}
