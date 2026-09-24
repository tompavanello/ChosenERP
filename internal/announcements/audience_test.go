package announcements

import (
	"strings"
	"testing"
)

func intPtr(v int) *int { return &v }

func TestBuildAudienceQuery_MembersWithDemographics(t *testing.T) {
	in := SendInput{
		Audience: "members",
		AudienceFilter: AudienceFilter{
			Genders:            []string{"female"},
			MaritalStatuses:    []string{"married"},
			MembershipStatuses: []string{"active"},
			AgeMin:             intPtr(20),
			AgeMax:             intPtr(30),
			BranchIDs:          []string{"b1"},
		},
	}
	sql, args := buildAudienceQuery(in, "tenant-1")

	for _, want := range []string{
		"m.gender IN", "m.marital_status IN", "m.membership_status IN",
		"m.branch_id IN", "m.tenant_id = $1",
		"date_part('year', age(m.birth_date)) >=", "<=",
	} {
		if !strings.Contains(sql, want) {
			t.Errorf("esperava %q no SQL:\n%s", want, sql)
		}
	}
	if strings.Contains(sql, "visitors") {
		t.Errorf("publico de membros nao deve consultar visitantes:\n%s", sql)
	}
	// tenant + gender, marital, membership, branch + 2 idades = 7 argumentos.
	if len(args) != 7 {
		t.Errorf("esperava 7 argumentos, veio %d: %v", len(args), args)
	}
}

func TestBuildAudienceQuery_EveryoneUnfilteredIncludesVisitors(t *testing.T) {
	sql, _ := buildAudienceQuery(SendInput{Audience: "everyone"}, "tenant-1")
	if !strings.Contains(sql, "FROM members") || !strings.Contains(sql, "FROM visitors") {
		t.Errorf("publico 'todos' sem segmentacao deve unir membros e visitantes:\n%s", sql)
	}
	if !strings.Contains(sql, "UNION ALL") {
		t.Errorf("esperava UNION ALL:\n%s", sql)
	}
	if !strings.Contains(sql, "v.tenant_id = $1") || !strings.Contains(sql, "m.tenant_id = $1") {
		t.Errorf("toda consulta deve filtrar o tenant explicito:\n%s", sql)
	}
}

func TestBuildAudienceQuery_EveryoneWithMemberFilterExcludesVisitors(t *testing.T) {
	sql, _ := buildAudienceQuery(SendInput{
		Audience:       "everyone",
		AudienceFilter: AudienceFilter{Genders: []string{"male"}},
	}, "tenant-1")
	if strings.Contains(sql, "visitors") {
		t.Errorf("segmentacao demografica deve excluir visitantes (nao tem sexo):\n%s", sql)
	}
}

func TestBuildAudienceQuery_GroupsWithoutIDsIsEmpty(t *testing.T) {
	if sql, _ := buildAudienceQuery(SendInput{Audience: "groups"}, "t"); sql != "" {
		t.Errorf("grupos sem ids nao deve gerar consulta, veio: %s", sql)
	}
	if sql, _ := buildAudienceQuery(SendInput{Audience: "ministries"}, "t"); sql != "" {
		t.Errorf("ministerios sem ids nao deve gerar consulta, veio: %s", sql)
	}
}

func TestBuildAudienceQuery_DefaultAudienceIsEveryone(t *testing.T) {
	sql, _ := buildAudienceQuery(SendInput{}, "t")
	if !strings.Contains(sql, "FROM members") || !strings.Contains(sql, "FROM visitors") {
		t.Errorf("publico vazio deve cair em 'todos':\n%s", sql)
	}
}

func TestMemberOnly(t *testing.T) {
	if (AudienceFilter{}).memberOnly() {
		t.Error("filtro vazio nao e member-only")
	}
	if (AudienceFilter{BranchIDs: []string{"b"}}).memberOnly() {
		t.Error("filtro de filial nao e member-only (visitante tambem tem filial)")
	}
	if !(AudienceFilter{AgeMin: intPtr(1)}).memberOnly() {
		t.Error("idade e member-only")
	}
}
