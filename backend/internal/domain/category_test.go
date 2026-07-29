package domain

import "testing"

func TestValidateCategory(t *testing.T) {
	tests := []struct {
		name       string
		slug       string
		catName    string
		wantFields []string // field keys expected to be reported
	}{
		{"valid", "herbal-tea", "Herbal Tea", nil},
		{"valid single word", "coffee", "Coffee", nil},
		{"empty name", "coffee", "", []string{"name"}},
		{"whitespace name", "coffee", "   ", []string{"name"}},
		{"empty slug", "", "Coffee", []string{"slug"}},
		{"uppercase slug", "Coffee", "Coffee", []string{"slug"}},
		{"slug with spaces", "her bal", "Herbal", []string{"slug"}},
		{"slug with trailing dash", "herbal-", "Herbal", []string{"slug"}},
		{"both invalid", "BAD SLUG", "", []string{"slug", "name"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := ValidateCategory(tt.slug, tt.catName)
			if len(fields) != len(tt.wantFields) {
				t.Fatalf("got %d field errors (%v), want %d", len(fields), fields, len(tt.wantFields))
			}
			for _, key := range tt.wantFields {
				if _, ok := fields[key]; !ok {
					t.Errorf("expected error for field %q, got %v", key, fields)
				}
			}
		})
	}
}

func TestNormalizeEmail(t *testing.T) {
	if got := NormalizeEmail("  User@Example.COM "); got != "user@example.com" {
		t.Errorf("NormalizeEmail = %q", got)
	}
}

func TestValidateRegistration(t *testing.T) {
	if fields := ValidateRegistration("valid@example.com", "long-enough-pw"); len(fields) != 0 {
		t.Errorf("valid registration reported errors: %v", fields)
	}
	if fields := ValidateRegistration("not-an-email", "short"); len(fields) != 2 {
		t.Errorf("expected email+password errors, got %v", fields)
	}
}
