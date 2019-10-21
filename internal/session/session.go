package session

import (
	"net/http"
)

func logMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

	})
}

func RegisterUser(w http.ResponseWriter, r *http.Request) {

}
