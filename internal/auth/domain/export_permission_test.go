package domain

import "testing"

func TestCanExportLeads_AdminOnly(t *testing.T) {
	cases := map[string]bool{
		string(RoleSuperAdmin): true,
		string(RoleAdmin):      true,
		string(RoleUser):       false,
		"":                     false,
	}
	for role, want := range cases {
		if got := CanExportLeads(role); got != want {
			t.Errorf("CanExportLeads(%q) = %v, want %v", role, got, want)
		}
	}
}
