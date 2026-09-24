package delivery

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// EvolutionClient fala com a Evolution API (WhatsApp Web self-hosted) para
// gerenciar instancias por filial: criar, conectar (QR Code), consultar estado
// e desconectar. A URL/chave do servidor Evolution sao globais (env); a
// instancia e identificada pelo id da filial.
type EvolutionClient struct {
	BaseURL string
	APIKey  string
	HTTP    *http.Client
}

// EvolutionQR e o retorno do QR Code (base64 pronto para <img> e o codigo
// textual, quando houver).
type EvolutionQR struct {
	Base64 string `json:"base64"`
	Code   string `json:"code"`
}

func (c *EvolutionClient) client() *http.Client {
	if c.HTTP != nil {
		return c.HTTP
	}
	return &http.Client{Timeout: 20 * time.Second}
}

func (c *EvolutionClient) do(ctx context.Context, method, path string, body any) ([]byte, int, error) {
	var reader *bytes.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return nil, 0, err
		}
		reader = bytes.NewReader(raw)
	} else {
		reader = bytes.NewReader(nil)
	}
	url := strings.TrimRight(c.BaseURL, "/") + path
	req, err := http.NewRequestWithContext(ctx, method, url, reader)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("apikey", c.APIKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.client().Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return buf.Bytes(), resp.StatusCode, fmt.Errorf("evolution api status=%d: %s", resp.StatusCode, truncate(buf.String(), 300))
	}
	return buf.Bytes(), resp.StatusCode, nil
}

// parseQR extrai o QR de respostas que variam entre versoes da Evolution
// (v1 costuma devolver base64/code no topo; v2 aninha em "qrcode").
func parseQR(raw []byte) EvolutionQR {
	var top map[string]json.RawMessage
	if err := json.Unmarshal(raw, &top); err != nil {
		return EvolutionQR{}
	}
	var out EvolutionQR
	if q, ok := top["qrcode"]; ok {
		var n struct {
			Base64 string `json:"base64"`
			Code   string `json:"code"`
		}
		if json.Unmarshal(q, &n) == nil {
			out.Base64, out.Code = n.Base64, n.Code
		}
	}
	if out.Base64 == "" {
		_ = json.Unmarshal(top["base64"], &out.Base64)
	}
	// `code` no topo pode ser o status HTTP (numero) em algumas versoes; o
	// unmarshal para string simplesmente falha e o campo fica vazio.
	if out.Code == "" {
		_ = json.Unmarshal(top["code"], &out.Code)
	}
	return out
}

// CreateInstance cria a instancia com o nome dado e ja pede o QR Code.
func (c *EvolutionClient) CreateInstance(ctx context.Context, instance string) (EvolutionQR, error) {
	raw, _, err := c.do(ctx, http.MethodPost, "/instance/create", map[string]any{
		"instanceName": instance,
		"qrcode":       true,
		"integration":  "WHATSAPP-BAILEYS",
	})
	if err != nil {
		return EvolutionQR{}, err
	}
	return parseQR(raw), nil
}

// Connect (re)obtem o QR Code de conexao de uma instancia existente.
func (c *EvolutionClient) Connect(ctx context.Context, instance string) (EvolutionQR, error) {
	raw, _, err := c.do(ctx, http.MethodGet, "/instance/connect/"+instance, nil)
	if err != nil {
		return EvolutionQR{}, err
	}
	return parseQR(raw), nil
}

// State devolve o estado normalizado: connected | connecting | disconnected.
func (c *EvolutionClient) State(ctx context.Context, instance string) (string, error) {
	raw, _, err := c.do(ctx, http.MethodGet, "/instance/connectionState/"+instance, nil)
	if err != nil {
		return "disconnected", err
	}
	var s struct {
		State    string `json:"state"`
		Instance struct {
			State string `json:"state"`
		} `json:"instance"`
	}
	if err := json.Unmarshal(raw, &s); err != nil {
		return "disconnected", err
	}
	st := s.State
	if st == "" {
		st = s.Instance.State
	}
	return normalizeEvolutionState(st), nil
}

func normalizeEvolutionState(s string) string {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "open", "connected":
		return "connected"
	case "connecting", "qr", "pairing":
		return "connecting"
	default:
		return "disconnected"
	}
}

// Logout desconecta a instancia (mantem a instancia criada).
func (c *EvolutionClient) Logout(ctx context.Context, instance string) error {
	_, _, err := c.do(ctx, http.MethodDelete, "/instance/logout/"+instance, nil)
	return err
}

// EvolutionInstance e um item de /instance/fetchInstances.
type EvolutionInstance struct {
	Name             string `json:"name"`
	ConnectionStatus string `json:"connectionStatus"`
	OwnerJid         string `json:"ownerJid"`
	Number           string `json:"number"`
	ProfileName      string `json:"profileName"`
	ProfilePicUrl    string `json:"profilePicUrl"`
}

// FetchInstance devolve os dados da instancia (inclui ownerJid do numero
// conectado). A rota varia entre versoes: v2 aceita ?instanceName=.
func (c *EvolutionClient) FetchInstance(ctx context.Context, instance string) (EvolutionInstance, error) {
	raw, _, err := c.do(ctx, http.MethodGet,
		"/instance/fetchInstances?instanceName="+url.QueryEscape(instance), nil)
	if err != nil {
		return EvolutionInstance{}, err
	}
	var arr []EvolutionInstance
	if err := json.Unmarshal(raw, &arr); err != nil || len(arr) == 0 {
		return EvolutionInstance{}, nil
	}
	return arr[0], nil
}

// ConnectedNumber extrai o numero (E.164 com "+") do ownerJid da instancia.
func ConnectedNumber(inst EvolutionInstance) string {
	jid := inst.OwnerJid
	if jid == "" {
		jid = inst.Number
	}
	if jid == "" {
		return ""
	}
	num := strings.SplitN(jid, "@", 2)[0]
	num = strings.TrimSpace(num)
	if num == "" {
		return ""
	}
	if strings.HasPrefix(num, "+") {
		return num
	}
	return "+" + num
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
