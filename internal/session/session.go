package session

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"github.com/chromz/wiki-backend/pkg/argon"
	"github.com/chromz/wiki-backend/pkg/errormessages"
	"github.com/chromz/wiki-backend/pkg/persistence"
	"github.com/dgrijalva/jwt-go"
	"github.com/julienschmidt/httprouter"
	"net/http"
	"os"
	"strings"
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

type tokenResponse struct {
	Token string `json:"token"`
}

type key string

// ClaimsKey is the context key to get claims
const ClaimsKey key = "claims"
const tokenTimeConstant = 60

var jwtSecret = []byte(os.Getenv("JWT_SECRET"))

const cookieName = "token"

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
		return
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

	expirationTime := time.Now().Add(tokenTimeConstant * time.Minute)
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
		return
	}
	resp := &tokenResponse{
		Token: tokenString,
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

// Refresh is an endpoint dedicated to refresh jwt tokens
// MUST be used with AuthMiddleware
func Refresh(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	claims := r.Context().Value(ClaimsKey).(*Claims)

	if time.Unix(claims.ExpiresAt, 0).Sub(time.Now()) > 30*time.Second {
		errormessages.WriteErrorMessage(w,
			"Token can only be refreshed when "+
				"it is withing 30 seconds of expire",
			http.StatusBadRequest)
		return
	}
	expirationTime := time.Now().Add(tokenTimeConstant * time.Minute)
	claims.StandardClaims = jwt.StandardClaims{
		ExpiresAt: expirationTime.Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		errormessages.WriteErrorMessage(w, "Unable to generate jwt",
			http.StatusInternalServerError)
		return
	}
	resp := &tokenResponse{
		Token: tokenString,
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

func keyFunc(token *jwt.Token) (interface{}, error) {
	if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
		return nil, errors.New("Invalid singing method")
	}
	return []byte(jwtSecret), nil
}

// AuthMiddleware middleware that checks if the JWT token is valid
func AuthMiddleware(next httprouter.Handle) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request,
		p httprouter.Params) {
		authorization := r.Header.Get("Authorization")
		parsedHeader := strings.Split(authorization, " ")
		if parsedHeader[0] != "Bearer" {
			errormessages.WriteErrorMessage(w, "Invalid token type",
				http.StatusBadRequest)
			return
		}

		token, err := jwt.ParseWithClaims(parsedHeader[1], &Claims{},
			keyFunc)
		if err != nil {
			errormessages.WriteErrorMessage(w,
				"Incorrect or expired token",
				http.StatusUnauthorized)
			return
		}
		// Check if token is valid
		if claims, ok := token.Claims.(*Claims); ok && token.Valid {
			ctx := context.WithValue(r.Context(), ClaimsKey,
				claims)
			next(w, r.WithContext(ctx), p)
		} else {
			errormessages.WriteErrorMessage(w, "Invalid token",
				http.StatusUnauthorized)
		}

	}
}
