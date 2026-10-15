// Package ai e um cliente minimo de LLM compativel com a API da OpenAI
// (funciona com OpenAI, Azure, Groq, Ollama e afins via base URL). Usado para
// gerar RASCUNHOS de planos devocionais; a equipe revisa antes de publicar.
//
// Sem chave configurada, o recurso fica desabilitado (ErrDisabled) e o resto da
// area de conteudo continua funcionando.
package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// ErrDisabled indica que a IA nao esta configurada (sem API key).
var ErrDisabled = errors.New("ai_disabled")

// Client chama um endpoint /chat/completions compativel com OpenAI.
type Client struct {
	BaseURL string
	APIKey  string
	Model   string
	HTTP    *http.Client
}

// NewClient monta o cliente. baseURL vazio usa o endpoint publico da OpenAI.
func NewClient(baseURL, apiKey, model string) *Client {
	if strings.TrimSpace(baseURL) == "" {
		baseURL = "https://api.openai.com/v1"
	}
	if strings.TrimSpace(model) == "" {
		model = "gpt-4o-mini"
	}
	return &Client{
		BaseURL: strings.TrimRight(baseURL, "/"),
		APIKey:  strings.TrimSpace(apiKey),
		Model:   model,
		HTTP:    &http.Client{Timeout: 90 * time.Second},
	}
}

// Enabled informa se ha credencial configurada.
func (c *Client) Enabled() bool { return c != nil && c.APIKey != "" }

type chatRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	Temperature float64       `json:"temperature"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatResponse struct {
	Choices []struct {
		Message chatMessage `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

// Chat envia system+user e devolve o texto da resposta.
func (c *Client) Chat(ctx context.Context, system, user string) (string, error) {
	if !c.Enabled() {
		return "", ErrDisabled
	}
	payload, err := json.Marshal(chatRequest{
		Model: c.Model,
		Messages: []chatMessage{
			{Role: "system", Content: system},
			{Role: "user", Content: user},
		},
		Temperature: 0.7,
	})
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.APIKey)

	res, err := c.HTTP.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	var out chatResponse
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		return "", fmt.Errorf("resposta invalida da IA: %w", err)
	}
	if out.Error != nil {
		return "", errors.New(out.Error.Message)
	}
	if res.StatusCode >= 300 {
		return "", fmt.Errorf("IA respondeu %d", res.StatusCode)
	}
	if len(out.Choices) == 0 {
		return "", errors.New("IA nao retornou conteudo")
	}
	return out.Choices[0].Message.Content, nil
}
