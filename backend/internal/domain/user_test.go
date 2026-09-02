package domain_test

import (
	"testing"

	"github.com/Nerses01/Mountain_Breath/backend/internal/domain"
)

// F2 (decision #96): the role vocabulary. Two real roles; everything else
// — including the tempting "" and the SQL-ish "ADMIN" — is refused.
func TestValidRole(t *testing.T) {
	for _, valid := range []string{domain.RoleCustomer, domain.RoleAdmin} {
		if !domain.ValidRole(valid) {
			t.Errorf("ValidRole(%q) = false, want true", valid)
		}
	}
	for _, invalid := range []string{"", "ADMIN", "superuser", "Customer"} {
		if domain.ValidRole(invalid) {
			t.Errorf("ValidRole(%q) = true, want false", invalid)
		}
	}
}
