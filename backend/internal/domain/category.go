package domain

import (
	"errors"
	"regexp"
	"strings"
	"time"
)

type Category struct {
	ID        int64
	Slug      string
	Name      string
	SortOrder int
	CreatedAt time.Time
}

// ErrSlugTaken is a sentinel error: the store returns it, the API layer
// translates it to 409 Conflict. Neither layer needs to know the other's
// details — they only share this value.
var ErrSlugTaken = errors.New("slug already taken")

// slugRe: lowercase words of letters/digits separated by single dashes,
// e.g. "herbal-tea", "coffee", "gift-sets-2026".
var slugRe = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

// ValidateCategory returns a field->problem map; empty map means valid.
func ValidateCategory(slug, name string) map[string]string {
	fields := make(map[string]string)
	if strings.TrimSpace(name) == "" {
		fields["name"] = "required"
	}
	switch {
	case slug == "":
		fields["slug"] = "required"
	case !slugRe.MatchString(slug):
		fields["slug"] = "must be lowercase letters/digits separated by dashes, e.g. \"herbal-tea\""
	}
	return fields
}
