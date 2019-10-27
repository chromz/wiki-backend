package grade

import (
	"encoding/json"
	"errors"
	"github.com/chromz/wiki-backend/internal/session"
	"github.com/chromz/wiki-backend/pkg/errormessages"
	"github.com/chromz/wiki-backend/pkg/pagination"
	"github.com/chromz/wiki-backend/pkg/persistence"
	"github.com/julienschmidt/httprouter"
	"net/http"
	"strconv"
)

// GradeDDL is the query to create the grades table
const GradeDDL = `
CREATE TABLE IF NOT EXISTS "grade" (
	"id"	INTEGER PRIMARY KEY AUTOINCREMENT UNIQUE,
	"name"	TEXT NOT NULL,
	"description"	TEXT
);
`

// Grade is a struct that represents a school grade
type Grade struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// Validate validates grade values
func (g *Grade) Validate() error {
	if g.Name == "" {
		return errors.New("Invalid name")
	}
	return nil
}

// Create creates a grade resource
func Create(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	grade := &Grade{}
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(grade)
	if err != nil {
		errormessages.WriteErrorMessage(w, "Invalid body type",
			http.StatusBadRequest)
		return
	}

	if err = grade.Validate(); err != nil {
		errormessages.WriteErrorMessage(w, "Invalid grade",
			http.StatusBadRequest)
		return
	}

	claims := r.Context().Value(session.ClaimsKey).(*session.Claims)
	if claims.Role != "TEACHER" {
		errormessages.WriteErrorInterface(w, "Not enough privileges",
			http.StatusUnauthorized)
		return
	}
	db := persistence.GetDb()
	tx, err := db.Begin()
	if err != nil {
		errormessages.WriteErrorMessage(w, "Unable to reach database",
			http.StatusInternalServerError)
		return
	}

	insertQuery := `
		INSERT INTO grade(name, description)
		VALUES(?, ?)
	`
	res, err := tx.Exec(insertQuery, grade.Name, grade.Description)
	if err != nil {
		errormessages.WriteErrorMessage(w, "Unable to add grade",
			http.StatusInternalServerError)
		tx.Rollback()
		return
	}
	grade.ID, _ = res.LastInsertId()
	err = tx.Commit()
	if err != nil {
		errString := "Unable to add grade"
		errormessages.WriteErrorMessage(w, errString,
			http.StatusInternalServerError)
		tx.Rollback()
		return
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(grade)
}

// Read returns available grades, paginated
func Read(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	params := r.URL.Query()

	size, err := strconv.Atoi(params.Get("size"))
	if err != nil {
		errormessages.WriteErrorMessage(w, "Invalid size",
			http.StatusBadRequest)
		return
	}
	nextToken, err := strconv.ParseInt(params.Get("nextToken"), 0, 64)
	if err != nil {
		errormessages.WriteErrorMessage(w, "Invalid next token",
			http.StatusBadRequest)
		return
	}
	page := &pagination.Page{
		Size:      size,
		NextToken: nextToken,
	}
	db := persistence.GetDb()
	findQuery := `
		SELECT id, name, description
		FROM grade
		WHERE id > ?
		LIMIT ?
	`
	if err = page.Validate(); err != nil {
		errormessages.WriteErrorMessage(w, "Invalid pagination",
			http.StatusBadRequest)
		return
	}

	rows, err := db.Query(findQuery, page.NextToken, page.Size)
	if err != nil {
		errormessages.WriteErrorMessage(w, "Unable to find classes",
			http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	var grades []Grade
	for rows.Next() {
		grade := Grade{}
		err = rows.Scan(&grade.ID, &grade.Name, &grade.Description)
		if err != nil {
			errormessages.WriteErrorMessage(w,
				"Unable to find classes",
				http.StatusInternalServerError)
			return
		}
		grades = append(grades, grade)
	}
	page.Data = grades
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(page)
}

// Update updates a grade resource
func Update(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	grade := &Grade{}
	decoder := json.NewDecoder(r.Body)
	gradeID, err := strconv.ParseInt(p.ByName("id"), 0, 64)
	if err != nil {
		errormessages.WriteErrorMessage(w, "Invalid id",
			http.StatusBadRequest)
		return

	}
	err = decoder.Decode(grade)
	grade.ID = gradeID
	if err != nil {
		errormessages.WriteErrorMessage(w, "Invalid body type",
			http.StatusBadRequest)
		return
	}

	if err = grade.Validate(); err != nil {
		errormessages.WriteErrorMessage(w, "Invalid grade",
			http.StatusBadRequest)
		return
	}

	if grade.ID <= 0 {
		errormessages.WriteErrorMessage(w, "ID is required",
			http.StatusBadRequest)
		return
	}

	claims := r.Context().Value(session.ClaimsKey).(*session.Claims)
	if claims.Role != "TEACHER" {
		errormessages.WriteErrorInterface(w, "Not enough privileges",
			http.StatusUnauthorized)
		return
	}
	db := persistence.GetDb()
	updateQuery := `
		UPDATE grade
		SET name = ?, description = ?
		WHERE id = ?
	`
	res, err := db.Exec(updateQuery, grade.Name, grade.Description, grade.ID)
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected != 1 {
		errormessages.WriteErrorInterface(w, "Id not found",
			http.StatusNotFound)
		return

	}
	if err != nil {
		errormessages.WriteErrorInterface(w, "Unable to update grade",
			http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Delete endpoint to delete a specifig grade
func Delete(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	gradeID, err := strconv.ParseInt(p.ByName("id"), 0, 64)

	if err != nil {
		errormessages.WriteErrorMessage(w, "Invalid id",
			http.StatusBadRequest)
		return

	}

	if gradeID <= 0 {
		errormessages.WriteErrorMessage(w, "ID is required",
			http.StatusBadRequest)
		return
	}

	claims := r.Context().Value(session.ClaimsKey).(*session.Claims)
	if claims.Role != "TEACHER" {
		errormessages.WriteErrorInterface(w, "Not enough privileges",
			http.StatusUnauthorized)
		return
	}
	db := persistence.GetDb()
	deleteQuery := `
		DELETE FROM grade
		WHERE id = ?
	`
	res, err := db.Exec(deleteQuery, gradeID)
	if err != nil {
		errormessages.WriteErrorMessage(w, "Unable to delete",
			http.StatusInternalServerError)
		return
	}
	affectedRows, err := res.RowsAffected()

	if affectedRows == 0 {
		errormessages.WriteErrorMessage(w, "Invalid id",
			http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusNoContent)
	// remember to delete the files!!!!
}
