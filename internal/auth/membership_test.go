package auth

import "testing"

func TestActiveMembershipsAndOptions(t *testing.T) {
	all := []Membership{
		{TenantID: "t1", TenantName: "Igreja A", TenantSlug: "a", Role: "super_admin", IsActive: true},
		{TenantID: "t2", TenantName: "Igreja B", TenantSlug: "b", Role: "secretario", IsActive: false},
		{TenantID: "t3", TenantName: "Igreja C", TenantSlug: "c", Role: "tesoureiro", IsActive: true},
	}
	active := activeMemberships(all)
	if len(active) != 2 {
		t.Fatalf("activeMemberships: got %d, queria 2", len(active))
	}
	opts := tenantOptions(active)
	if len(opts) != 2 || opts[0].Slug != "a" || opts[1].ID != "t3" {
		t.Fatalf("tenantOptions inesperado: %+v", opts)
	}
}

func TestFindMembershipIgnoresInactive(t *testing.T) {
	all := []Membership{
		{TenantID: "t1", IsActive: true},
		{TenantID: "t2", IsActive: false},
	}
	if _, ok := findMembership(all, "t2"); ok {
		t.Errorf("findMembership nao deveria aceitar vinculo inativo")
	}
	if m, ok := findMembership(all, "t1"); !ok || m.TenantID != "t1" {
		t.Errorf("findMembership deveria achar t1")
	}
	if _, ok := findMembership(all, "t9"); ok {
		t.Errorf("findMembership nao deveria achar tenant inexistente")
	}
}
