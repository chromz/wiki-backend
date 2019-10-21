package routes

import (
	"github.com/julienschmidt/httprouter"
	"net/http"
)

// RouteHandler returns the handler of all routes on the api
func RouteHandler() http.Handler {
	router := httprouter.New()

	return router
}
