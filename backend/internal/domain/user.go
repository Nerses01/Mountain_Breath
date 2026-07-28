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

type User struct {
	ID           int64
	Email        string
	PasswordHash string
	Role         string
	CreatedAt    time.Time
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
	if _, err := mail.ParseAddress(email); err != nil {
		fields["email"] = "must be a valid email address"
	}
	if len(password) < 8 {
		fields["password"] = "must be at least 8 characters"
	}
	return fields
}
