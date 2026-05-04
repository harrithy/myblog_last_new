package middleware

import (
	"myblog_last_new/internal/config"
	"net/http"
	"strings"
)

// CORS adds the configured CORS headers to responses.
func CORS(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		origin := strings.TrimSpace(r.Header.Get("Origin"))
		if allowedOrigin := resolveAllowedOrigin(origin, config.CORSAllowedOrigins()); allowedOrigin != "" {
			w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
			w.Header().Add("Vary", "Origin")
		}

		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next(w, r)
	}
}

func resolveAllowedOrigin(origin string, allowedOrigins []string) string {
	for _, allowedOrigin := range allowedOrigins {
		if allowedOrigin == "*" {
			return "*"
		}

		if origin != "" && strings.EqualFold(origin, allowedOrigin) {
			return origin
		}
	}

	return ""
}
