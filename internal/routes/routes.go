package routes

import (
	"github.com/chromz/wiki-backend/internal/course"
	"github.com/chromz/wiki-backend/internal/grade"
	"github.com/chromz/wiki-backend/internal/session"
	"github.com/chromz/wiki-backend/internal/textclass"
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
		w.Header().Set("Content-Type", "application/json")
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
	router.POST("/grade",
		originMiddleware(session.AuthMiddleware(grade.Create)),
	)
	router.GET("/grade",
		originMiddleware(session.AuthMiddleware(grade.Read)),
	)
	router.PUT("/grade/:id",
		originMiddleware(session.AuthMiddleware(grade.Update)),
	)
	router.DELETE("/grade/:id",
		originMiddleware(session.AuthMiddleware(grade.Delete)),
	)
	router.POST("/grade/:id/course",
		originMiddleware(session.AuthMiddleware(course.Create)),
	)
	router.GET("/grade/:id/course",
		originMiddleware(session.AuthMiddleware(course.Read)),
	)
	router.PUT("/grade/:id/course/:courseid",
		originMiddleware(session.AuthMiddleware(course.Update)),
	)
	router.DELETE("/grade/:id/course/:courseid",
		originMiddleware(session.AuthMiddleware(course.Delete)),
	)
	router.POST("/grade/:id/course/:courseid/textclass",
		originMiddleware(session.AuthMiddleware(textclass.Create)),
	)
	router.GET("/grade/:id/course/:courseid/textclass",
		originMiddleware(session.AuthMiddleware(textclass.Read)),
	)
	router.POST("/grade/:id/course/:courseid/textclass/:classid/file",
		originMiddleware(session.AuthMiddleware(textclass.WriteFile)),
	)
	router.PUT("/grade/:id/course/:courseid/textclass/:classid",
		originMiddleware(session.AuthMiddleware(textclass.Update)),
	)
	router.DELETE("/grade/:id/course/:courseid/textclass/:classid",
		originMiddleware(session.AuthMiddleware(textclass.Delete)),
	)
	return router
}
