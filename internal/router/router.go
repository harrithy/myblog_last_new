package router

import (
	"database/sql"
	"myblog_last_new/internal/handler"
	"myblog_last_new/internal/middleware"
	"myblog_last_new/internal/repository"
	"myblog_last_new/internal/response"
	"net/http"
)

type routeHandlers map[string]http.HandlerFunc

// Router wraps http.ServeMux and applies common middleware.
type Router struct {
	mux *http.ServeMux
}

// New creates a new Router.
func New() *Router {
	return &Router{mux: http.NewServeMux()}
}

// ServeHTTP implements http.Handler.
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.mux.ServeHTTP(w, req)
}

// Handle registers a plain handler with CORS.
func (r *Router) Handle(pattern string, handler http.HandlerFunc) {
	r.mux.HandleFunc(pattern, middleware.CORS(handler))
}

// HandleWithAuth registers a handler protected by JWT auth.
func (r *Router) HandleWithAuth(pattern string, handler http.HandlerFunc) {
	r.mux.HandleFunc(pattern, middleware.CORS(middleware.Auth(handler)))
}

// HandleWithOwnerAuth registers a handler protected by owner-only auth.
func (r *Router) HandleWithOwnerAuth(pattern string, handler http.HandlerFunc) {
	r.mux.HandleFunc(pattern, middleware.CORS(middleware.OwnerOnly(handler)))
}

// HandleMethods registers method-specific handlers with CORS.
func (r *Router) HandleMethods(pattern string, handlers routeHandlers) {
	r.Handle(pattern, methodHandler(handlers))
}

// HandleMethodsWithAuth registers method-specific handlers with JWT auth.
func (r *Router) HandleMethodsWithAuth(pattern string, handlers routeHandlers) {
	r.HandleWithAuth(pattern, methodHandler(handlers))
}

// HandleMethodsWithOwnerAuth registers method-specific handlers with owner auth.
func (r *Router) HandleMethodsWithOwnerAuth(pattern string, handlers routeHandlers) {
	r.HandleWithOwnerAuth(pattern, methodHandler(handlers))
}

func methodHandler(handlers routeHandlers) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if handler, ok := handlers[r.Method]; ok {
			handler(w, r)
			return
		}

		response.MethodNotAllowed(w, "Method not allowed")
	}
}

// RegisterRoutes registers all application routes.
func RegisterRoutes(mux *http.ServeMux, db *sql.DB) {
	router := &Router{mux: mux}

	userRepo := repository.NewUserRepository(db)
	blogRepo := repository.NewBlogRepository(db)
	categoryRepo := repository.NewCategoryRepository(db)
	visitRepo := repository.NewVisitRepository(db)
	guestRepo := repository.NewGuestRepository(db)
	ownerRepo := repository.NewOwnerVisitRepository(db)
	commentRepo := repository.NewCommentRepository(db)

	authHandler := handler.NewAuthHandler(userRepo, ownerRepo)
	userHandler := handler.NewUserHandler(userRepo)
	blogHandler := handler.NewBlogHandler(blogRepo)
	categoryHandler := handler.NewCategoryHandler(categoryRepo)
	visitHandler := handler.NewVisitHandler(visitRepo, guestRepo, ownerRepo)
	commentHandler := handler.NewCommentHandler(commentRepo)
	githubAuthHandler := handler.NewGitHubAuthHandler(userRepo, ownerRepo)
	uploadHandler := handler.NewUploadHandler()
	aiHandler := handler.NewAIHandler()

	registerDualRoutes(router, authHandler, userHandler, blogHandler, categoryHandler, visitHandler, commentHandler, githubAuthHandler, uploadHandler, aiHandler)
}

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
	aiHandler *handler.AIHandler,
) {
	for _, prefix := range []string{"", "/api"} {
		router.HandleMethods(prefix+"/register", routeHandlers{
			http.MethodPost: authHandler.Register,
		})

		router.HandleMethods(prefix+"/login", routeHandlers{
			http.MethodPost: authHandler.Login,
		})

		router.HandleMethods(prefix+"/auth/verify", routeHandlers{
			http.MethodGet: authHandler.VerifyToken,
		})

		router.HandleMethods(prefix+"/users", routeHandlers{
			http.MethodGet:  userHandler.GetUsers,
			http.MethodPost: middleware.OwnerOnly(userHandler.AddUser),
		})

		router.HandleMethodsWithAuth(prefix+"/users/", routeHandlers{
			http.MethodGet:    userHandler.GetUserByID,
			http.MethodPut:    userHandler.UpdateUser,
			http.MethodDelete: userHandler.DeleteUser,
		})

		router.HandleMethods(prefix+"/blogs", routeHandlers{
			http.MethodGet: blogHandler.GetBlogs,
		})

		router.HandleMethods(prefix+"/blogs/", routeHandlers{
			http.MethodGet: blogHandler.GetBlogDetail,
		})

		router.HandleMethods(prefix+"/visits", routeHandlers{
			http.MethodGet:  visitHandler.GetVisitLogs,
			http.MethodPost: visitHandler.LogVisit,
		})

		router.HandleMethods(prefix+"/guest", routeHandlers{
			http.MethodPost: visitHandler.LogGuestRecord,
		})

		router.HandleMethods(prefix+"/owner/visits", routeHandlers{
			http.MethodGet: visitHandler.GetOwnerVisitStats,
		})

		router.HandleMethods(prefix+"/owner/today-visits", routeHandlers{
			http.MethodGet: visitHandler.GetOwnerTodayVisits,
		})

		router.HandleMethods(prefix+"/categories", routeHandlers{
			http.MethodGet:  categoryHandler.GetCategories,
			http.MethodPost: middleware.OwnerOnly(categoryHandler.CreateCategory),
		})

		router.HandleMethods(prefix+"/categories/hot-tags", routeHandlers{
			http.MethodGet: categoryHandler.GetHotTags,
		})

		router.HandleMethods(prefix+"/categories/", routeHandlers{
			http.MethodGet:    categoryHandler.GetCategoryByID,
			http.MethodPut:    middleware.OwnerOnly(categoryHandler.UpdateCategory),
			http.MethodDelete: middleware.OwnerOnly(categoryHandler.DeleteCategory),
		})

		router.HandleMethods(prefix+"/comments", routeHandlers{
			http.MethodGet:  commentHandler.GetComments,
			http.MethodPost: commentHandler.CreateComment,
		})

		router.HandleMethods(prefix+"/comments/", routeHandlers{
			http.MethodDelete: middleware.Auth(commentHandler.DeleteComment),
		})

		router.HandleMethods(prefix+"/auth/github", routeHandlers{
			http.MethodGet: githubAuthHandler.GetGitHubLoginURL,
		})

		router.HandleMethods(prefix+"/auth/github/callback", routeHandlers{
			http.MethodGet: githubAuthHandler.GitHubCallback,
		})

		router.HandleMethods(prefix+"/auth/github/login", routeHandlers{
			http.MethodPost: githubAuthHandler.GitHubCallbackWithCode,
		})

		router.HandleMethods(prefix+"/github/repos", routeHandlers{
			http.MethodGet: githubAuthHandler.GetOwnerRepos,
		})

		router.HandleMethodsWithOwnerAuth(prefix+"/upload", routeHandlers{
			http.MethodPost: uploadHandler.ProxyUpload,
		})

		router.HandleMethods(prefix+"/ai/chat", routeHandlers{
			http.MethodPost: aiHandler.Chat,
		})
	}
}
