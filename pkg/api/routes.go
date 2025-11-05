package api

import (
	"database/sql"
	"myblog_last_new/pkg/auth"
	"net/http"
)

// RegisterRoutes registers all routes for the application
func RegisterRoutes(mux *http.ServeMux, db *sql.DB) {
	mux.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		Login(w, r, db)
	})
	mux.HandleFunc("/users", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			GetUsers(w, r, db)
		case http.MethodPost:
			auth.Middleware(func(w http.ResponseWriter, r *http.Request) {
				AddUser(w, r, db)
			})(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/visits", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			GetVisitLogs(w, r, db)
		case http.MethodPost:
			LogVisit(w, r, db)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/guest", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			LogGuestRecord(w, r, db)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})
}
