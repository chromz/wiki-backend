package users

import (
	"database/sql"
	"encoding/json"
	"errors"
	"github.com/chromz/wiki-backend/pkg/argon"
	"github.com/chromz/wiki-backend/pkg/errormessages"
	"github.com/chromz/wiki-backend/pkg/log"
	"github.com/chromz/wiki-backend/pkg/persistence"
	"github.com/google/uuid"
	"github.com/julienschmidt/httprouter"
	"net/http"
)

var logger = log.GetLogger()

// UsersDDL is the create query of the users table
const UsersDDL = `
CREATE TABLE IF NOT EXISTS "user" (
	"id"	TEXT NOT NULL UNIQUE,
	"username"	TEXT NOT NULL UNIQUE,
	"first_name"	TEXT NOT NULL,
	"last_name"	TEXT NOT NULL,
	"password"	TEXT NOT NULL,
	PRIMARY KEY("id")
);
`

// User is a struct that represents a user in the system
type User struct {
	ID        string `json:"id"`
	Username  string `json:"username"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Password  string `json:"password"`
}

// SignUpUser creates an entry on the users table
func SignUpUser(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	user := &User{}
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(user)
	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		errormessages.WriteErrorMessage(w, "Invalid body type",
			http.StatusBadRequest)
		return
	}
	user.ID = uuid.New().String()
	if validations, err := user.validate(); err != nil {
		errormessages.WriteErrorInterface(w, validations,
			http.StatusBadRequest)
		return
	}
	db := persistence.GetDb()
	tx, err := db.Begin()
	if err != nil {
		errormessages.WriteErrorMessage(w, "Unable to reach database",
			http.StatusInternalServerError)
		return
	}
	// Hash the password

	findQuery := `
		SELECT username FROM user WHERE username = ?
	`
	row := tx.QueryRow(findQuery, user.Username)
	var username string
	err = row.Scan(&username)
	if err != sql.ErrNoRows {
		errormessages.WriteErrorInterface(w, "User already exists",
			http.StatusConflict)
		tx.Rollback()
		return
	}
	hashedPassword, err := argon.GenerateFromPassword([]byte(user.Password),
		3, 12*1024, 1)
	if err != nil {
		errormessages.WriteErrorMessage(w, "Unable to hash password",
			http.StatusInternalServerError)
		tx.Rollback()
		return
	}
	insertQuery := `
		INSERT INTO user(id, username, first_name, last_name, password)
		VALUES(?, ?, ?, ?, ?)
		`
	tx.Exec(insertQuery, user.ID, user.Username, user.FirstName,
		user.LastName, hashedPassword)
	insertUserRole := `
		INSERT INTO user_role(user_id, role_id)
		VALUES (?, 2)
	`
	tx.Exec(insertUserRole, user.ID)
	err = tx.Commit()
	if err != nil {
		errString := "Unable to add user"
		errormessages.WriteErrorMessage(w, errString,
			http.StatusInternalServerError)
		logger.Error(errString, err)
		tx.Rollback()
		return
	}
	w.WriteHeader(http.StatusCreated)
	logger.Info("User created " + user.ID)
}

func (user *User) validate() (map[string][]string, error) {
	errs := make(map[string][]string)
	if user.ID == "" {
		errs["id"] = append(errs["ID"], "id is a required element")
	}

	if user.Username == "" {
		errs["username"] = append(errs["username"],
			"username is a required element")
	}

	if user.FirstName == "" {
		errs["firstName"] = append(errs["firstName"],
			"first name is a required element")
	}

	if user.LastName == "" {
		errs["lastName"] = append(errs["lastName"],
			"last name is a required element")
	}

	if user.Password == "" {
		errs["password"] = append(errs["password"],
			"password is a required element")
	}
	if len(errs) > 0 {
		return errs, errors.New("Validation failed")
	}
	return errs, nil
}
