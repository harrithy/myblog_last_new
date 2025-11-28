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

	// 博客列表和详情接口
	mux.HandleFunc("/blogs", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			GetBlogs(w, r, db)
			return
		}
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	})

	mux.HandleFunc("/blogs/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			GetBlogDetail(w, r, db)
			return
		}
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	})

	// 博客主人访问统计接口
	mux.HandleFunc("/owner/visits", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			GetOwnerVisitStats(w, r, db)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/owner/today-visits", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			GetOwnerTodayVisits(w, r, db)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// 分类接口
	mux.HandleFunc("/categories", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			GetCategories(w, r, db)
		case http.MethodPost:
			CreateCategory(w, r, db)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/categories/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			GetCategoryByID(w, r, db)
		case http.MethodPut:
			UpdateCategory(w, r, db)
		case http.MethodDelete:
			DeleteCategory(w, r, db)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})
}
