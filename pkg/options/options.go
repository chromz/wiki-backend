package options

import (
	"net/http"
)

// SetOptions sets the allowed methods, for inflight requests
func SetOptions(allowed string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", allowed)
			w.Header().Set("Access-Control-Allow-Headers",
				"Authorization, Content-Type")
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}
