package session

import (
	"database/sql"
	"encoding/json"
	"github.com/chromz/wiki-backend/pkg/argon"
	"github.com/chromz/wiki-backend/pkg/errormessages"
	"github.com/chromz/wiki-backend/pkg/log"
	"github.com/chromz/wiki-backend/pkg/persistence"
	"github.com/dgrijalva/jwt-go"
	"github.com/julienschmidt/httprouter"
	"net/http"
	"os"
	"time"
)

// Credentials is a struct that represents the login info
type Credentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// Claims is a struct that represents the data inside a JWT
type Claims struct {
	UserID string `json:"userId"`
	Role   string `json:"role"`
	jwt.StandardClaims
}

var jwtSecret = []byte(os.Getenv("JWT_SECRET"))
var logger = log.GetLogger()

// RolesDDL DDL for roles table
const RolesDDL = `
CREATE TABLE IF NOT EXISTS "role" (
	"id"	INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT UNIQUE,
	"name"	TEXT NOT NULL UNIQUE,
	"description"	TEXT
);
`

// UserRolesDDL DDL for users -> role intermediate table
const UserRolesDDL = `
CREATE TABLE IF NOT EXISTS "user_role" (
	"user_id"	TEXT UNIQUE,
	"role_id"	INTEGER,
	FOREIGN KEY("role_id") REFERENCES "role"("id"),
	FOREIGN KEY("user_id") REFERENCES "user"("id"),
	PRIMARY KEY("user_id")
);
`

// Authenticate is a HandlerFunc that logins the user
func Authenticate(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	credentials := &Credentials{}
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(credentials)

	if err != nil {
		errormessages.WriteErrorMessage(w, "Invalid body type",
			http.StatusBadRequest)
	}

	db := persistence.GetDb()
	if err != nil {
		errormessages.WriteErrorMessage(w, "Unable to reach database",
			http.StatusInternalServerError)
		return
	}
	findUserQuery := `
		SELECT id, password
		FROM user WHERE username = ?
	`
	var userID, hash string
	row := db.QueryRow(findUserQuery, credentials.Username)
	err = row.Scan(&userID, &hash)
	if err == sql.ErrNoRows {
		errormessages.WriteErrorMessage(w, "User does not exist",
			http.StatusNotFound)
		return
	}
	if err != nil {
		errormessages.WriteErrorMessage(w, "Unable to fetch user",
			http.StatusInternalServerError)
		return
	}

	err = argon.CompareHashAndPassword([]byte(hash),
		[]byte(credentials.Password))
	if err != nil {
		errormessages.WriteErrorMessage(w, "Username or password incorrect",
			http.StatusUnauthorized)
		return
	}

	rolesQuery := `
		SELECT role.name
		FROM user_role
		JOIN role ON user_role.role_id = role.id
		WHERE user_role.user_id = ?
	`
	var roleName string
	row = db.QueryRow(rolesQuery, userID)
	err = row.Scan(&roleName)
	if err == sql.ErrNoRows {
		errormessages.WriteErrorMessage(w, "User does not have a role",
			http.StatusConflict)
		return
	}

	expirationTime := time.Now().Add(time.Hour)
	claims := &Claims{
		UserID: userID,
		Role:   roleName,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: expirationTime.Unix(),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		errormessages.WriteErrorMessage(w, "Unable to generate jwt",
			http.StatusInternalServerError)
		logger.Error("Unable to generate jwt", err)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    tokenString,
		Expires:  expirationTime,
		HttpOnly: true,
	})
	w.WriteHeader(http.StatusNoContent)
}
