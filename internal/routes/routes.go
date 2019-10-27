package routes

import (
	"github.com/chromz/wiki-backend/internal/session"
	"github.com/chromz/wiki-backend/internal/users"
	"github.com/julienschmidt/httprouter"
	"net/http"
)

func cors(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Access-Control-Request-Method") != "" {
		// Set CORS headers
		header := w.Header()
		header.Set("Access-Control-Allow-Methods", header.Get("Allow"))
		header.Set("Access-Control-Allow-Origin", "*")
		header.Set("Access-Control-Allow-Headers",
			"Authorization, Content-Type")
	}

	// Adjust status code to 204
	w.WriteHeader(http.StatusNoContent)
}

func originMiddleware(next httprouter.Handle) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request,
		p httprouter.Params) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		next(w, r, p)
	}
}

// RouteHandler returns the handler of all routes on the api
func RouteHandler() http.Handler {
	router := httprouter.New()
	router.GlobalOPTIONS = http.HandlerFunc(cors)
	router.POST("/users", originMiddleware(users.SignUpUser))
	router.POST("/auth", originMiddleware(session.Authenticate))
	router.POST("/auth/token",
		originMiddleware(session.AuthMiddleware(session.Refresh)),
	)
	return router
}
