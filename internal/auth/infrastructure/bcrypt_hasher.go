package infrastructure

import (
	"golang.org/x/crypto/bcrypt"

	"github.com/trvux/elc-go/internal/auth/domain"
)

// BcryptPasswordHasher uses bcrypt at the library default cost (10) — the
// same cost Supabase Auth (GoTrue) used for the accounts being migrated, so
// existing password hashes remain valid without forcing anyone to reset.
type BcryptPasswordHasher struct{}

var _ domain.PasswordHasher = BcryptPasswordHasher{}

func NewBcryptPasswordHasher() BcryptPasswordHasher {
	return BcryptPasswordHasher{}
}

func (BcryptPasswordHasher) Hash(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func (BcryptPasswordHasher) Compare(hash, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}
