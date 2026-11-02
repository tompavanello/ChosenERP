package help

import "testing"

func testAccess(app, role, path string, perms []string, features []string, platform bool) Access {
	p := map[string]bool{}
	for _, k := range perms {
		p[k] = true
	}
	f := map[string]bool{}
	for _, k := range features {
		f[k] = true
	}
	return Access{
		App: app, Role: role, Path: path, PlatformAdmin: platform,
		CanPerm:    func(k string) bool { return p[k] },
		HasFeature: func(k string) bool { return f[k] },
	}
}

func TestLoadAndSearch(t *testing.T) {
	b := Load()
	if len(b.articles) < 70 {
		t.Fatalf("esperava ao menos 70 artigos carregados, obtive %d", len(b.articles))
	}
	acc := testAccess("webadmin", "tesoureiro", "/dashboard/finance", []string{"finance.read"}, []string{"finance", "reports"}, false)
	items := b.Articles(acc, "conciliacao", "")
	if len(items) == 0 {
		t.Fatal("busca por conciliacao nao retornou artigos")
	}
	if items[0].Slug != "conciliacao" && items[0].Slug != "conciliacao-bancaria" {
		t.Fatalf("resultado inesperado: %s", items[0].Slug)
	}
}

func TestContextPrefersSpecificRoute(t *testing.T) {
	b := Load()
	acc := testAccess("webadmin", "secretario", "/dashboard/members/abc", []string{"members.read"}, nil, false)
	ctx := b.Context(acc)
	if ctx.Guide == nil || ctx.Guide.Slug != "ficha-membro" {
		t.Fatalf("guia esperado ficha-membro, obtive %+v", ctx.Guide)
	}
}

func TestPermissionAndFeatureFilter(t *testing.T) {
	b := Load()
	// Sem permissao de financeiro e com o modulo desabilitado, nada de financeiro.
	acc := testAccess("webadmin", "membro", "/dashboard", nil, nil, false)
	for _, a := range b.Articles(acc, "", "") {
		if a.Slug == "financeiro-lancamentos" || a.Slug == "conciliacao" {
			t.Fatalf("artigo de financeiro vazou sem permissao: %s", a.Slug)
		}
	}
	// Plataforma nao ve artigos operacionais.
	pacc := testAccess("webadmin", "platform_admin", "/dashboard/platform", nil, nil, true)
	for _, a := range b.Articles(pacc, "", "") {
		if a.Slug == "financeiro-lancamentos" {
			t.Fatal("admin de plataforma nao deveria ver artigos operacionais")
		}
	}
	if len(b.Articles(pacc, "plataforma", "")) == 0 {
		t.Fatal("admin de plataforma deveria ver artigos da plataforma")
	}
}

func TestMemberArticles(t *testing.T) {
	b := Load()
	acc := testAccess("member", "membro", "/home", nil, nil, false)
	ctx := b.Context(acc)
	if ctx.Guide == nil || ctx.Guide.App != "member" {
		t.Fatalf("guia do membro nao encontrado: %+v", ctx.Guide)
	}
}
