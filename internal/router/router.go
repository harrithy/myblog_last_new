package router

import (
	"database/sql"
	"myblog_last_new/internal/handler"
	"myblog_last_new/internal/middleware"
	"myblog_last_new/internal/repository"
	"net/http"
)

// Router 封装 http.ServeMux 并提供额外功能
type Router struct {
	mux *http.ServeMux
}

// New 创建新的 Router
func New() *Router {
	return &Router{mux: http.NewServeMux()}
}

// ServeHTTP 实现 http.Handler 接口
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.mux.ServeHTTP(w, req)
}

// Handle 注册带 CORS 中间件的处理器
func (r *Router) Handle(pattern string, handler http.HandlerFunc) {
	r.mux.HandleFunc(pattern, middleware.CORS(handler))
}

// HandleWithAuth 注册需要认证的处理器
func (r *Router) HandleWithAuth(pattern string, handler http.HandlerFunc) {
	r.mux.HandleFunc(pattern, middleware.CORS(middleware.Auth(handler)))
}

// RegisterRoutes 注册所有应用路由
func RegisterRoutes(mux *http.ServeMux, db *sql.DB) {
	router := &Router{mux: mux}

	// 初始化数据仓库
	userRepo := repository.NewUserRepository(db)
	blogRepo := repository.NewBlogRepository(db)
	categoryRepo := repository.NewCategoryRepository(db)
	visitRepo := repository.NewVisitRepository(db)
	guestRepo := repository.NewGuestRepository(db)
	ownerRepo := repository.NewOwnerVisitRepository(db)
	commentRepo := repository.NewCommentRepository(db)

	// 初始化处理器
	authHandler := handler.NewAuthHandler(userRepo, ownerRepo)
	userHandler := handler.NewUserHandler(userRepo)
	blogHandler := handler.NewBlogHandler(blogRepo)
	categoryHandler := handler.NewCategoryHandler(categoryRepo)
	visitHandler := handler.NewVisitHandler(visitRepo, guestRepo, ownerRepo)
	commentHandler := handler.NewCommentHandler(commentRepo)
	githubAuthHandler := handler.NewGitHubAuthHandler(userRepo, ownerRepo)
	uploadHandler := handler.NewUploadHandler()

	// 注册 /path 和 /api/path 两种路径模式的路由
	registerDualRoutes(router, authHandler, userHandler, blogHandler, categoryHandler, visitHandler, commentHandler, githubAuthHandler, uploadHandler)
}

// registerDualRoutes 注册 /path 和 /api/path 两种路径模式的路由
func registerDualRoutes(
	router *Router,
	authHandler *handler.AuthHandler,
	userHandler *handler.UserHandler,
	blogHandler *handler.BlogHandler,
	categoryHandler *handler.CategoryHandler,
	visitHandler *handler.VisitHandler,
	commentHandler *handler.CommentHandler,
	githubAuthHandler *handler.GitHubAuthHandler,
	uploadHandler *handler.UploadHandler,
) {
	paths := []string{"", "/api"}

	for _, prefix := range paths {
		// 认证路由
		router.Handle(prefix+"/login", func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodPost {
				authHandler.Login(w, r)
				return
			}
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		})

		// 用户路由
		router.Handle(prefix+"/users", func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodGet:
				userHandler.GetUsers(w, r)
			case http.MethodPost:
				middleware.Auth(userHandler.AddUser)(w, r)
			default:
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			}
		})

		// 博客路由
		router.Handle(prefix+"/blogs", func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodGet {
				blogHandler.GetBlogs(w, r)
				return
			}
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		})

		router.Handle(prefix+"/blogs/", func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodGet {
				blogHandler.GetBlogDetail(w, r)
				return
			}
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		})

		// 访问记录路由
		router.Handle(prefix+"/visits", func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodGet:
				visitHandler.GetVisitLogs(w, r)
			case http.MethodPost:
				visitHandler.LogVisit(w, r)
			default:
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			}
		})

		// 访客路由
		router.Handle(prefix+"/guest", func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodPost {
				visitHandler.LogGuestRecord(w, r)
				return
			}
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		})

		// 博主路由
		router.Handle(prefix+"/owner/visits", func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodGet {
				visitHandler.GetOwnerVisitStats(w, r)
				return
			}
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		})

		router.Handle(prefix+"/owner/today-visits", func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodGet {
				visitHandler.GetOwnerTodayVisits(w, r)
				return
			}
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		})

		// 分类路由
		router.Handle(prefix+"/categories", func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodGet:
				categoryHandler.GetCategories(w, r)
			case http.MethodPost:
				categoryHandler.CreateCategory(w, r)
			default:
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			}
		})

		router.Handle(prefix+"/categories/", func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodGet:
				categoryHandler.GetCategoryByID(w, r)
			case http.MethodPut:
				categoryHandler.UpdateCategory(w, r)
			case http.MethodDelete:
				categoryHandler.DeleteCategory(w, r)
			default:
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			}
		})

		// 评论路由
		router.HandleWithAuth(prefix+"/comments", func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodGet:
				commentHandler.GetComments(w, r)
			case http.MethodPost:
				commentHandler.CreateComment(w, r)
			default:
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			}
		})

		router.HandleWithAuth(prefix+"/comments/", func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodDelete {
				commentHandler.DeleteComment(w, r)
				return
			}
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		})

		// GitHub OAuth 路由
		router.Handle(prefix+"/auth/github", func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodGet {
				githubAuthHandler.GetGitHubLoginURL(w, r)
				return
			}
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		})

		router.Handle(prefix+"/auth/github/callback", func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodGet {
				githubAuthHandler.GitHubCallback(w, r)
				return
			}
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		})

		router.Handle(prefix+"/auth/github/login", func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodPost {
				githubAuthHandler.GitHubCallbackWithCode(w, r)
				return
			}
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		})

		// GitHub 仓库路由
		router.Handle(prefix+"/github/repos", func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodGet {
				githubAuthHandler.GetOwnerRepos(w, r)
				return
			}
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		})

		// 图片上传代理路由
		router.Handle(prefix+"/upload", func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodPost {
				uploadHandler.ProxyUpload(w, r)
				return
			}
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		})
	}
}
