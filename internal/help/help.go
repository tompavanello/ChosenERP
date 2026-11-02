// Package help carrega o Manual do Usuario embutido no binario e resolve os
// artigos por tela, perfil e plano. O conteudo fica em content/ como Markdown
// com frontmatter simples (sem dependencia externa), servido pela API para o
// webadmin e para o app do membro.
package help

import (
	"embed"
	"errors"
	"io/fs"
	"path"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

//go:embed content
var contentFS embed.FS

// ErrNotFound indica artigo/tour inexistente.
var ErrNotFound = errors.New("artigo de ajuda nao encontrado")

// Article e um documento do manual.
type Article struct {
	Slug        string   `json:"slug"`
	App         string   `json:"app"`
	Title       string   `json:"title"`
	Category    string   `json:"category"`
	Summary     string   `json:"summary"`
	Body        string   `json:"body,omitempty"`
	Routes      []string `json:"routes,omitempty"`
	Permissions []string `json:"permissions,omitempty"`
	Feature     string   `json:"feature,omitempty"`
	Roles       []string `json:"roles,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	Order       int      `json:"order"`
	Tour        string   `json:"tour,omitempty"`
	Platform    bool     `json:"platform,omitempty"`
}

// Summary e a versao curta usada em listagens.
type Summary struct {
	Slug        string   `json:"slug"`
	App         string   `json:"app"`
	Title       string   `json:"title"`
	Category    string   `json:"category"`
	Summary     string   `json:"summary"`
	Tags        []string `json:"tags,omitempty"`
	Order       int      `json:"order"`
	Tour        string   `json:"tour,omitempty"`
	Routes      []string `json:"routes,omitempty"`
	Permissions []string `json:"permissions,omitempty"`
	Feature     string   `json:"feature,omitempty"`
}

// Access descreve o que a sessao atual pode ver.
type Access struct {
	App           string
	Role          string
	Path          string
	PlatformAdmin bool
	// CanPerm/HasFeature sao injetados pelo gateway (RBAC e entitlement).
	CanPerm    func(string) bool
	HasFeature func(string) bool
}

// Context e o resultado da ajuda contextual de uma rota.
type Context struct {
	Path     string    `json:"path"`
	Guide    *Article  `json:"guide,omitempty"`
	Articles []Summary `json:"articles"`
	Tour     string    `json:"tour,omitempty"`
}

type tokenized struct {
	tokens map[string]int
}

// Book mantem os artigos e o indice de busca em memoria.
type Book struct {
	articles []Article
	index    map[string]tokenized
}

// Load le os artigos embutidos. E tolerante a arquivos invalidos: registra o
// problema e continua (o restante da ajuda continua disponivel).
func Load() *Book {
	b := &Book{index: map[string]tokenized{}}
	_ = fs.WalkDir(contentFS, "content", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(strings.ToLower(p), ".md") {
			return nil
		}
		raw, err := contentFS.ReadFile(p)
		if err != nil {
			return nil
		}
		a, err := parseArticle(string(raw))
		if err != nil {
			return nil
		}
		if a.Slug == "" {
			a.Slug = strings.TrimSuffix(path.Base(p), ".md")
		}
		if a.App == "" {
			a.App = appFromPath(p)
		}
		if a.Order == 0 {
			a.Order = 100
		}
		b.articles = append(b.articles, a)
		return nil
	})
	sort.SliceStable(b.articles, func(i, j int) bool {
		if b.articles[i].App != b.articles[j].App {
			return b.articles[i].App < b.articles[j].App
		}
		if b.articles[i].Category != b.articles[j].Category {
			return b.articles[i].Category < b.articles[j].Category
		}
		if b.articles[i].Order != b.articles[j].Order {
			return b.articles[i].Order < b.articles[j].Order
		}
		return b.articles[i].Title < b.articles[j].Title
	})
	for i := range b.articles {
		b.index[b.articles[i].Slug] = tokenized{tokens: tokenize(b.articles[i].Title + " " + strings.Join(b.articles[i].Tags, " ") + " " + b.articles[i].Summary + " " + b.articles[i].Body)}
	}
	return b
}

// Articles lista os artigos visiveis, opcionalmente filtrando por termo e
// categoria. Sem query devolve todos (ordenados por categoria/ordem).
func (b *Book) Articles(acc Access, query, category string) []Summary {
	if b == nil {
		return []Summary{}
	}
	out := make([]Summary, 0, len(b.articles))
	for _, a := range b.articles {
		if !visible(a, acc) {
			continue
		}
		if category != "" && !strings.EqualFold(a.Category, category) {
			continue
		}
		out = append(out, a.summary())
	}
	if q := strings.TrimSpace(query); q != "" {
		out = b.rank(out, q)
	}
	return out
}

// Categories lista as categorias visiveis para a sessao.
func (b *Book) Categories(acc Access) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, a := range b.articles {
		if visible(a, acc) && !seen[a.Category] {
			seen[a.Category] = true
			out = append(out, a.Category)
		}
	}
	sort.Strings(out)
	return out
}

// Get devolve um artigo pelo slug, respeitando o acesso.
func (b *Book) Get(acc Access, slug string) (*Article, error) {
	for i := range b.articles {
		if b.articles[i].Slug == slug {
			if !visible(b.articles[i], acc) {
				return nil, ErrNotFound
			}
			a := b.articles[i]
			return &a, nil
		}
	}
	return nil, ErrNotFound
}

// Context resolve a ajuda da rota atual. Sempre devolve os artigos gerais da
// app como fallback, mesmo quando nenhuma rota casa.
func (b *Book) Context(acc Access) Context {
	ctx := Context{Path: acc.Path, Articles: []Summary{}}
	type hit struct {
		a       Article
		specLen int
	}
	var hits []hit
	for _, a := range b.articles {
		if !visible(a, acc) {
			continue
		}
		for _, r := range a.Routes {
			if routeMatch(r, acc.Path) {
				hits = append(hits, hit{a: a, specLen: len(strings.TrimSuffix(r, "*"))})
				break
			}
		}
	}
	sort.SliceStable(hits, func(i, j int) bool { return hits[i].specLen > hits[j].specLen })
	for _, h := range hits {
		ctx.Articles = append(ctx.Articles, h.a.summary())
	}
	if len(hits) > 0 {
		a := hits[0].a
		ctx.Guide = &a
		ctx.Tour = a.Tour
	}
	// Mantem a lista curta: o guia + ate 4 relacionados.
	if len(ctx.Articles) > 5 {
		ctx.Articles = ctx.Articles[:5]
	}
	return ctx
}

// Ranked devolve os artigos visiveis mais relevantes para a pergunta.
func (b *Book) Ranked(acc Access, query string, limit int) []Article {
	if b == nil {
		return nil
	}
	type scored struct {
		a Article
		s int
	}
	var all []scored
	for _, a := range b.articles {
		if !visible(a, acc) {
			continue
		}
		all = append(all, scored{a: a, s: score(a, b.index[a.Slug], query)})
	}
	sort.SliceStable(all, func(i, j int) bool { return all[i].s > all[j].s })
	if limit <= 0 || limit > len(all) {
		limit = len(all)
	}
	out := make([]Article, 0, limit)
	for _, x := range all[:limit] {
		if x.s == 0 {
			continue
		}
		out = append(out, x.a)
	}
	return out
}

func (b *Book) rank(list []Summary, query string) []Summary {
	type scored struct {
		s Summary
		n int
	}
	q := tokenize(query)
	var all []scored
	for _, item := range list {
		a := b.find(item.Slug)
		if a == nil {
			continue
		}
		all = append(all, scored{s: item, n: score(*a, b.index[a.Slug], query)})
	}
	sort.SliceStable(all, func(i, j int) bool { return all[i].n > all[j].n })
	out := make([]Summary, 0, len(all))
	for _, x := range all {
		if len(q) > 0 && x.n == 0 {
			continue
		}
		out = append(out, x.s)
	}
	return out
}

func (b *Book) find(slug string) *Article {
	for i := range b.articles {
		if b.articles[i].Slug == slug {
			return &b.articles[i]
		}
	}
	return nil
}

func (a Article) summary() Summary {
	return Summary{
		Slug: a.Slug, App: a.App, Title: a.Title, Category: a.Category,
		Summary: a.Summary, Tags: a.Tags, Order: a.Order, Tour: a.Tour,
		Routes: a.Routes, Permissions: a.Permissions, Feature: a.Feature,
	}
}

func visible(a Article, acc Access) bool {
	if a.App != acc.App {
		return false
	}
	if acc.PlatformAdmin != a.Platform {
		return false
	}
	if len(a.Roles) > 0 && !contains(a.Roles, acc.Role) {
		return false
	}
	if acc.Role == "super_admin" || acc.Role == "admin_sede" {
		// Sede tem todas as permissoes, mas o gate de plano continua valendo.
	} else if len(a.Permissions) > 0 {
		ok := false
		for _, p := range a.Permissions {
			if acc.CanPerm != nil && acc.CanPerm(p) {
				ok = true
				break
			}
		}
		if !ok {
			return false
		}
	}
	if a.Feature != "" && acc.HasFeature != nil && !acc.HasFeature(a.Feature) {
		return false
	}
	return true
}

func contains(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

// routeMatch suporta rota exata e prefixo com "*" no final.
func routeMatch(pattern, p string) bool {
	pattern = strings.TrimSpace(pattern)
	if pattern == "" || p == "" {
		return false
	}
	if strings.HasSuffix(pattern, "*") {
		return strings.HasPrefix(p, strings.TrimSuffix(pattern, "*"))
	}
	return p == pattern
}

func score(a Article, idx tokenized, query string) int {
	qt := tokenize(query)
	if len(qt) == 0 {
		return 0
	}
	total := 0
	title := tokenize(a.Title)
	tags := tokenize(strings.Join(a.Tags, " "))
	for t := range qt {
		if title[t] > 0 {
			total += 6
		}
		if tags[t] > 0 {
			total += 4
		}
		if idx.tokens[t] > 0 {
			total += 1
		}
	}
	return total
}

func tokenize(s string) map[string]int {
	out := map[string]int{}
	var b strings.Builder
	flush := func() {
		if b.Len() == 0 {
			return
		}
		t := b.String()
		b.Reset()
		if len(t) >= 2 {
			out[t]++
		}
	}
	for _, r := range fold(s) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			continue
		}
		flush()
	}
	flush()
	return out
}

// fold remove acentos e coloca em minusculas, para busca tolerante.
func fold(s string) string {
	s = strings.ToLower(s)
	repl := strings.NewReplacer(
		"á", "a", "à", "a", "â", "a", "ã", "a", "ä", "a",
		"é", "e", "è", "e", "ê", "e", "ë", "e",
		"í", "i", "ì", "i", "î", "i", "ï", "i",
		"ó", "o", "ò", "o", "ô", "o", "õ", "o", "ö", "o",
		"ú", "u", "ù", "u", "û", "u", "ü", "u",
		"ç", "c", "ñ", "n",
	)
	return repl.Replace(s)
}

func parseArticle(raw string) (Article, error) {
	raw = strings.TrimPrefix(raw, "\ufeff")
	a := Article{Order: 100}
	body := raw
	if strings.HasPrefix(strings.TrimSpace(raw), "---") {
		text := strings.TrimSpace(raw)
		text = strings.TrimPrefix(text, "---")
		parts := strings.SplitN(text, "\n---", 2)
		if len(parts) == 2 {
			meta := parts[0]
			body = strings.TrimSpace(parts[1])
			for _, line := range strings.Split(meta, "\n") {
				line = strings.TrimSpace(line)
				if line == "" || strings.HasPrefix(line, "#") {
					continue
				}
				k, v, ok := strings.Cut(line, ":")
				if !ok {
					continue
				}
				k = strings.ToLower(strings.TrimSpace(k))
				v = cleanValue(v)
				switch k {
				case "id", "slug":
					a.Slug = v
				case "app":
					a.App = v
				case "title":
					a.Title = v
				case "category":
					a.Category = v
				case "summary":
					a.Summary = v
				case "routes":
					a.Routes = list(v)
				case "permissions", "permission":
					a.Permissions = list(v)
				case "feature":
					a.Feature = v
				case "roles", "role":
					a.Roles = list(v)
				case "tags":
					a.Tags = list(v)
				case "order":
					if n, err := strconv.Atoi(v); err == nil {
						a.Order = n
					}
				case "tour":
					a.Tour = v
				case "platform":
					a.Platform = v == "true" || v == "1" || v == "sim"
				}
			}
		}
	}
	if a.Title == "" {
		return Article{}, errors.New("artigo sem titulo")
	}
	if a.Category == "" {
		a.Category = "Geral"
	}
	a.Body = body
	return a, nil
}

func cleanValue(v string) string {
	v = strings.TrimSpace(v)
	v = strings.Trim(v, `"'`)
	return strings.TrimSpace(v)
}

func list(v string) []string {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil
	}
	v = strings.Trim(v, "[]")
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = cleanValue(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func appFromPath(p string) string {
	if strings.Contains(p, "/member/") {
		return "member"
	}
	return "webadmin"
}

