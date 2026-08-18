package domain

import (
	"errors"
	"net/mail"
	"strings"
	"time"
)

const (
	RoleCustomer = "customer"
	RoleAdmin    = "admin"
)

// A5: the harvest-notes toggle's three states, as the settings screen sees
// them. "none" covers never-subscribed AND unsubscribed — both restart the
// same way (a fresh double-opt-in).
const (
	NewsletterNone       = "none"
	NewsletterPending    = "pending"
	NewsletterSubscribed = "subscribed"
)

type User struct {
	ID           int64
	Email        string
	PasswordHash string
	Role         string
	CreatedAt    time.Time

	// A5 (canvas 10): the profile the settings screen edits. Empty strings,
	// not pointers — one kind of absence, and the UI falls back to the
	// email without a nil branch.
	FullName string
	Phone    string
	// The one notification preference with a sender to honour it (decision
	// log #87): F2's status-change mailer checks this before sending.
	NotifyOrderUpdates bool
}

func (u User) IsAdmin() bool { return u.Role == RoleAdmin }

var (
	ErrEmailTaken = errors.New("email already registered")
	// One error for both wrong email and wrong password: the response must
	// not reveal whether an account exists (user enumeration attack).
	ErrInvalidCredentials = errors.New("invalid email or password")
)

// NormalizeEmail makes emails comparable: User@X.com and user@x.com are the
// same account.
func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func ValidateRegistration(email, password string) map[string]string {
	fields := make(map[string]string)
	if !ValidEmail(email) {
		fields["email"] = ValidationEmailFormat
	}
	if len(password) < PasswordMinLength {
		fields["password"] = ValidationPasswordTooShort
	}
	return fields
}

// ValidateProfile checks the settings form's two fields (A5). Both are
// OPTIONAL — an account works without them — so only excess is refused;
// emptiness is a valid answer, not an error.
func ValidateProfile(fullName, phone string) map[string]string {
	fields := make(map[string]string)
	if len(fullName) > 120 {
		fields["full_name"] = ValidationTooLong
	}
	if len(phone) > 40 {
		fields["phone"] = ValidationTooLong
	}
	return fields
}

// ValidEmail is the one address check, shared since E9 by registration and
// the newsletter form. net/mail implements the RFC's grammar — a hand-rolled
// regex would reject real addresses the RFC allows, which is the postal-code
// lesson (ValidateAddress) applied to email.
func ValidEmail(email string) bool {
	_, err := mail.ParseAddress(email)
	return err == nil
}
