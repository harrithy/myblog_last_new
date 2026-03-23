package security

import (
	"errors"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

// HashPassword hashes a plain-text password for storage.
func HashPassword(password string) (string, error) {
	if strings.TrimSpace(password) == "" {
		return "", errors.New("password is required")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}

	return string(hash), nil
}

// CheckPassword compares a stored hash against a plain-text password.
func CheckPassword(hash, password string) bool {
	if hash == "" || strings.TrimSpace(password) == "" {
		return false
	}

	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}
