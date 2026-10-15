// Package bible integra uma API externa licenciada da Biblia (api.bible) e
// mantem um cache local dos trechos (tabela bible_cache, dado publico) para
// respeitar o rate limit e acelerar a leitura. Sem chave configurada o recurso
// fica desabilitado.
package bible

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"chosenerp/internal/store"
)

// ErrDisabled indica que a integracao da Biblia nao esta configurada.
var ErrDisabled = errors.New("bible_disabled")

// Version e uma traducao disponivel.
type Version struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Abbreviation string `json:"abbreviation"`
	Language     string `json:"language"`
}

// Book e um livro da Biblia.
type Book struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Chapter e a referencia de um capitulo (id opaco da API).
type Chapter struct {
	ID     string `json:"id"`
	Number string `json:"number"`
}

// Passage e o texto de um capitulo.
type Passage struct {
	Reference string `json:"reference"`
	Content   string `json:"content"`
}

// Service consulta a API e usa o cache local.
type Service struct {
	key     string
	baseURL string
	http    *http.Client
	store   *store.Store
}

// NewService monta o servico. baseURL vazio usa o endpoint da api.bible.
func NewService(st *store.Store, key, baseURL string) *Service {
	if strings.TrimSpace(baseURL) == "" {
		baseURL = "https://api.scripture.api.bible/v1"
	}
	return &Service{
		key:     strings.TrimSpace(key),
		baseURL: strings.TrimRight(baseURL, "/"),
		http:    &http.Client{Timeout: 30 * time.Second},
		store:   st,
	}
}

// Enabled informa se ha chave configurada.
func (s *Service) Enabled() bool { return s != nil && s.key != "" }

// Versions lista as traducoes (cache global por idioma).
func (s *Service) Versions(ctx context.Context, language string) ([]Version, error) {
	if !s.Enabled() {
		return nil, ErrDisabled
	}
	lang := strings.TrimSpace(language)
	if lang == "" {
		lang = "por"
	}
	key := "versions:" + lang
	var out []Version
	if s.cached(ctx, key, &out) {
		return out, nil
	}
	var resp struct {
		Data []struct {
			ID           string `json:"id"`
			Name         string `json:"name"`
			Abbreviation string `json:"abbreviation"`
			Language     struct {
				ID string `json:"id"`
			} `json:"language"`
		} `json:"data"`
	}
	if err := s.get(ctx, "/bibles?language="+url.QueryEscape(lang), &resp); err != nil {
		return nil, err
	}
	out = []Version{}
	for _, b := range resp.Data {
		out = append(out, Version{ID: b.ID, Name: b.Name, Abbreviation: b.Abbreviation, Language: b.Language.ID})
	}
	s.put(ctx, key, out)
	return out, nil
}

// Books lista os livros de uma versao.
func (s *Service) Books(ctx context.Context, bibleID string) ([]Book, error) {
	if !s.Enabled() {
		return nil, ErrDisabled
	}
	key := "books:" + bibleID
	var out []Book
	if s.cached(ctx, key, &out) {
		return out, nil
	}
	var resp struct {
		Data []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"data"`
	}
	if err := s.get(ctx, "/bibles/"+url.PathEscape(bibleID)+"/books", &resp); err != nil {
		return nil, err
	}
	out = []Book{}
	for _, b := range resp.Data {
		out = append(out, Book{ID: b.ID, Name: b.Name})
	}
	s.put(ctx, key, out)
	return out, nil
}

// Chapters lista os capitulos de um livro.
func (s *Service) Chapters(ctx context.Context, bibleID, bookID string) ([]Chapter, error) {
	if !s.Enabled() {
		return nil, ErrDisabled
	}
	key := "chapters:" + bibleID + ":" + bookID
	var out []Chapter
	if s.cached(ctx, key, &out) {
		return out, nil
	}
	var resp struct {
		Data []struct {
			ID     string `json:"id"`
			Number string `json:"number"`
		} `json:"data"`
	}
	if err := s.get(ctx, "/bibles/"+url.PathEscape(bibleID)+"/books/"+url.PathEscape(bookID)+"/chapters", &resp); err != nil {
		return nil, err
	}
	out = []Chapter{}
	for _, c := range resp.Data {
		out = append(out, Chapter{ID: c.ID, Number: c.Number})
	}
	s.put(ctx, key, out)
	return out, nil
}

// Chapter devolve o texto de um capitulo.
func (s *Service) Chapter(ctx context.Context, bibleID, chapterID string) (*Passage, error) {
	if !s.Enabled() {
		return nil, ErrDisabled
	}
	key := "chapter:" + bibleID + ":" + chapterID
	var out Passage
	if s.cached(ctx, key, &out) {
		return &out, nil
	}
	var resp struct {
		Data struct {
			Reference string `json:"reference"`
			Content   string `json:"content"`
		} `json:"data"`
	}
	if err := s.get(ctx, "/bibles/"+url.PathEscape(bibleID)+"/chapters/"+url.PathEscape(chapterID)+"?content-type=text", &resp); err != nil {
		return nil, err
	}
	out = Passage{Reference: resp.Data.Reference, Content: resp.Data.Content}
	s.put(ctx, key, out)
	return &out, nil
}

func (s *Service) get(ctx context.Context, path string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.baseURL+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("api-key", s.key)
	res, err := s.http.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		return fmt.Errorf("biblia respondeu %d", res.StatusCode)
	}
	return json.NewDecoder(res.Body).Decode(out)
}

func (s *Service) cached(ctx context.Context, key string, out any) bool {
	if s.store == nil {
		return false
	}
	var raw string
	err := s.store.Pool().QueryRow(ctx, `SELECT payload::text FROM bible_cache WHERE cache_key = $1`, key).Scan(&raw)
	if err != nil {
		return false
	}
	return json.Unmarshal([]byte(raw), out) == nil
}

func (s *Service) put(ctx context.Context, key string, v any) {
	if s.store == nil {
		return
	}
	b, err := json.Marshal(v)
	if err != nil {
		return
	}
	_, _ = s.store.Pool().Exec(ctx, `
		INSERT INTO bible_cache (cache_key, payload, fetched_at) VALUES ($1, $2::jsonb, now())
		ON CONFLICT (cache_key) DO UPDATE SET payload = EXCLUDED.payload, fetched_at = now()`,
		key, string(b))
}
