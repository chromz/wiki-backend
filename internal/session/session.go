package session

import (
	"encoding/json"
	"github.com/chromz/wiki-backend/pkg/errormessages"
	"github.com/julienschmidt/httprouter"
	"net/http"
)

// Credentials is a struct that represents the login info
type Credentials struct {
	username string
	password string
}

// RolesDDL DDL for roles table
const RolesDDL = `
CREATE TABLE IF NOT EXISTS "role" (
	"id"	INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT UNIQUE,
	"name"	TEXT NOT NULL UNIQUE
);
`

// UserRolesDDL DDL for users -> role intermediate table
const UserRolesDDL = `
CREATE TABLE IF NOT EXISTS "user_role" (
	"user_id"	TEXT UNIQUE,
	"role_id"	INTEGER,
	PRIMARY KEY("user_id")
);
`

func logMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

	})
}

// Authenticate is a HandlerFunc that logins the user
func Authenticate(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	credentials := &Credentials{}
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(credentials)

	if err != nil {
		errormessages.WriteErrorMessage(w, "Invalid body type",
			http.StatusBadRequest)
	}

}
