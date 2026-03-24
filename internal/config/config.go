package config

import (
	"os"
	"strconv"
	"strings"
)

// OwnerSettings stores owner/admin identity and login settings.
type OwnerSettings struct {
	Name           string
	Account        string
	Password       string
	Email          string
	GitHubID       int64
	GitHubUsername string
}

// GetEnv returns an environment variable or a default value when missing.
func GetEnv(key, defaultValue string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return defaultValue
}

// JWTSecret returns the signing secret used for JWT tokens.
func JWTSecret() string {
	return GetEnv("JWT_SECRET", "dev-only-jwt-secret-change-me")
}

// LoadOwnerSettings loads owner/admin settings from the environment.
func LoadOwnerSettings() OwnerSettings {
	githubID, err := strconv.ParseInt(GetEnv("OWNER_GITHUB_ID", "156180607"), 10, 64)
	if err != nil {
		githubID = 156180607
	}

	account := GetEnv("OWNER_ACCOUNT", "harrio")
	email := GetEnv("OWNER_EMAIL", "harrithy@github.com")

	return OwnerSettings{
		Name:           GetEnv("OWNER_NAME", account),
		Account:        account,
		Password:       strings.TrimSpace(os.Getenv("OWNER_PASSWORD")),
		Email:          email,
		GitHubID:       githubID,
		GitHubUsername: GetEnv("OWNER_GITHUB_USERNAME", "harrithy"),
	}
}

// MatchesPasswordLogin returns true when the provided credentials match the owner login.
func (s OwnerSettings) MatchesPasswordLogin(login, password string) bool {
	login = strings.TrimSpace(login)
	password = strings.TrimSpace(password)

	if login == "" || s.Password == "" || password != s.Password {
		return false
	}

	return login == s.Account || login == s.Email
}

// IsOwnerIdentity returns true when the username belongs to the configured owner.
func (s OwnerSettings) IsOwnerIdentity(username string) bool {
	if username == "" {
		return false
	}

	if username == s.Account || username == s.Email {
		return true
	}

	if s.GitHubID > 0 {
		return username == "github_"+strconv.FormatInt(s.GitHubID, 10)
	}

	return false
}
