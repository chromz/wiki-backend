package textclass

import (
	"encoding/json"
	"errors"
	"github.com/chromz/wiki-backend/internal/session"
	"github.com/chromz/wiki-backend/pkg/errormessages"
	"github.com/chromz/wiki-backend/pkg/pagination"
	"github.com/chromz/wiki-backend/pkg/persistence"
	"github.com/julienschmidt/httprouter"
	"github.com/mattn/go-sqlite3"
	"net/http"
	"strconv"
)

var syncDir string

// TextClass represents a markdown textual class
type TextClass struct {
	ID           int64  `json:"id"`
	CourseID     int64  `json:"courseId"`
	Title        string `json:"title"`
	FileName     string `json:"fileName"`
	ProcFileName string `json:"procFileName"`
}

// SyncDir sets the dir to synchronize
// it should be improved
func SyncDir(dir string) {
	syncDir = dir
}

// Validate checks if textclass is valid
func (t *TextClass) Validate() error {
	if t.Title == "" {
		return errors.New("Title is missing")
	}
	return nil
}

// TextClassDDL query to create the text class table
const TextClassDDL = `
CREATE TABLE IF NOT EXISTS "text_class" (
	"id"	INTEGER,
	"course_id"	INTEGER NOT NULL,
	"file_name"	TEXT DEFAULT '',
	"proc_file_name"	TEXT DEFAULT '',
	"title"	TEXT NOT NULL,
	FOREIGN KEY("course_id") REFERENCES "course"("id") ON DELETE CASCADE,
	PRIMARY KEY("id")
);
`

// Create creates a new textclass in db, prepares for execution
func Create(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	textClass := &TextClass{}
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(textClass)

	if err != nil {
		errormessages.WriteErrorMessage(w, "Invalid body type",
			http.StatusBadRequest)
		return
	}

	courseID, err := strconv.ParseInt(p.ByName("courseid"), 0, 64)
	if err != nil {
		errormessages.WriteErrorMessage(w, "Invalid id",
			http.StatusBadRequest)
		return

	}
	textClass.CourseID = courseID
	if err = textClass.Validate(); err != nil {
		errormessages.WriteErrorMessage(w, "Invalid text class object",
			http.StatusBadRequest)
		return
	}

	claims := r.Context().Value(session.ClaimsKey).(*session.Claims)
	if claims.Role != "TEACHER" {
		errormessages.WriteErrorInterface(w, "Not enough privileges", +http.StatusUnauthorized)
		return
	}
	db := persistence.GetDb()
	tx, err := db.Begin()
	if err != nil {
		errormessages.WriteErrorMessage(w, "Unable to reach database",
			http.StatusInternalServerError)
		tx.Rollback()
		return
	}

	insertQuery := `
		INSERT INTO text_class(course_id, title)
		VALUES(?, ?)
	`
	res, err := tx.Exec(insertQuery, textClass.CourseID, textClass.Title)
	if err != nil {
		if sqliteErr, ok := err.(sqlite3.Error); ok {
			if sqliteErr.ExtendedCode == sqlite3.ErrConstraintForeignKey {
				errormessages.WriteErrorMessage(w,
					"Invalid course id",
					http.StatusBadRequest)
				tx.Rollback()
				return
			}
		}
		errormessages.WriteErrorMessage(w, "Invalid course id",
			http.StatusInternalServerError)
		tx.Rollback()
		return
	}

	textClass.ID, err = res.LastInsertId()
	err = tx.Commit()
	if err != nil {
		errString := "Unable to add text class"
		errormessages.WriteErrorMessage(w, errString,
			http.StatusInternalServerError)
		tx.Rollback()
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(textClass)
}

// Read returns available text classess, paginated
func Read(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	params := r.URL.Query()

	courseID, err := strconv.ParseInt(p.ByName("courseid"), 0, 64)
	if err != nil {
		errormessages.WriteErrorMessage(w, "Invalid course id",
			http.StatusBadRequest)
		return

	}

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

	if err = page.Validate(); err != nil {
		errormessages.WriteErrorMessage(w, "Invalid pagination",
			http.StatusBadRequest)
		return
	}

	db := persistence.GetDb()
	findQuery := `
SELECT id, course_id, file_name, proc_file_name, title
		FROM text_class
		WHERE id > ?
		AND course_id = ?
		LIMIT ?
	`
	rows, err := db.Query(findQuery, page.NextToken, courseID, page.Size)
	if err != nil {
		errormessages.WriteErrorMessage(w, "Unable to find text classes",
			http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	var classes []TextClass
	for rows.Next() {
		class := TextClass{}
		err = rows.Scan(&class.ID, &class.CourseID, &class.FileName,
			&class.ProcFileName, &class.Title)
		if err != nil {
			errormessages.WriteErrorMessage(w,
				"Unable to find classes"+err.Error(),
				http.StatusInternalServerError)
			return
		}
		classes = append(classes, class)
	}
	page.Data = classes
	classesCount := len(classes)
	if classesCount > 0 {
		page.NextToken = classes[classesCount-1].ID
	} else {
		page.NextToken = -1
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(page)

}

// Update updates a text class resource
func Update(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	textClass := &TextClass{}
	decoder := json.NewDecoder(r.Body)
	classID, err := strconv.ParseInt(p.ByName("classid"), 0, 64)
	if err != nil {
		errormessages.WriteErrorMessage(w, "Invalid id",
			http.StatusBadRequest)
		return
	}

	err = decoder.Decode(textClass)
	textClass.ID = classID
	if err != nil {
		errormessages.WriteErrorMessage(w, "Invalid body type",
			http.StatusBadRequest)
		return
	}

	if err = textClass.Validate(); err != nil {
		errormessages.WriteErrorMessage(w, "Invalid class",
			http.StatusBadRequest)
		return
	}

	if textClass.ID <= 0 {
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
UPDATE text_class
		SET title = ?
		WHERE id = ?
	`
	res, err := db.Exec(updateQuery, textClass.Title, textClass.ID)
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected != 1 {
		errormessages.WriteErrorInterface(w, "Id not found",
			http.StatusNotFound)
		return

	}
	if err != nil {
		errormessages.WriteErrorInterface(w, "Unable to update course",
			http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Delete is an endpoint to delete a specific text class
func Delete(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	classID, err := strconv.ParseInt(p.ByName("classid"), 0, 64)
	if err != nil {
		errormessages.WriteErrorMessage(w, "Invalid id",
			http.StatusBadRequest)
		return
	}

	if classID <= 0 {
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
DELETE FROM text_class
		WHERE id = ?
	`
	res, err := db.Exec(deleteQuery, classID)
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
