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

// GetCSVEnv returns a trimmed CSV environment variable.
func GetCSVEnv(key string, defaultValues []string) []string {
	raw := GetEnv(key, "")
	if raw == "" {
		return append([]string(nil), defaultValues...)
	}

	parts := strings.Split(raw, ",")
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		if value := strings.TrimSpace(part); value != "" {
			values = append(values, value)
		}
	}

	if len(values) == 0 {
		return append([]string(nil), defaultValues...)
	}

	return values
}

// GetInt64Env returns an int64 environment variable or the default value.
func GetInt64Env(key string, defaultValue int64) int64 {
	value := GetEnv(key, "")
	if value == "" {
		return defaultValue
	}

	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return defaultValue
	}

	return parsed
}

// JWTSecret returns the signing secret used for JWT tokens.
func JWTSecret() string {
	return GetEnv("JWT_SECRET", "dev-only-jwt-secret-change-me")
}

// CORSAllowedOrigins returns the configured allowed CORS origins.
func CORSAllowedOrigins() []string {
	return GetCSVEnv("CORS_ALLOW_ORIGINS", []string{"*"})
}

// UploadProxyURL returns the remote upload endpoint used by the proxy.
func UploadProxyURL() string {
	return GetEnv("IMAGE_HOST_URL", "https://image.harrio.xyz/upload")
}

// UploadMaxBytes returns the maximum accepted upload payload size.
func UploadMaxBytes() int64 {
	return GetInt64Env("UPLOAD_MAX_BYTES", 10<<20)
}

// UploadAllowedTypes returns the accepted upload MIME types.
func UploadAllowedTypes() []string {
	return GetCSVEnv("UPLOAD_ALLOWED_TYPES", []string{
		"image/jpeg",
		"image/png",
		"image/gif",
		"image/webp",
	})
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
