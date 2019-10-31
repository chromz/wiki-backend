package textclass

import (
	"database/sql"
	"encoding/json"
	"errors"
	"github.com/chromz/wiki-backend/internal/session"
	"github.com/chromz/wiki-backend/pkg/errormessages"
	"github.com/chromz/wiki-backend/pkg/pagination"
	"github.com/chromz/wiki-backend/pkg/persistence"
	"github.com/julienschmidt/httprouter"
	"github.com/mattn/go-sqlite3"
	"io"
	"net/http"
	"os"
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
func SyncDir() string {
	return syncDir
}

// NewSyncDir sets up a directory to store files and synchronize
func NewSyncDir(dir string) {
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
	claims := r.Context().Value(session.ClaimsKey).(*session.Claims)
	if claims.Role != "TEACHER" {
		errormessages.WriteErrorInterface(w, "Not enough privileges",
			http.StatusUnauthorized)
		return
	}
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

// ReadFile is an endpoint to get the markdown file
func ReadFile(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	classID, err := strconv.ParseInt(p.ByName("classid"), 0, 64)
	if err != nil {
		errormessages.WriteErrorMessage(w, "Invalid id",
			http.StatusBadRequest)
		return
	}
	db := persistence.GetDb()
	findQuery := `
		SELECT file_name, proc_file_name
		FROM text_class
		WHERE id = ?
	`
	row := db.QueryRow(findQuery, classID)
	var fileName, procFileName string
	err = row.Scan(&fileName, &procFileName)
	if err == sql.ErrNoRows {
		errormessages.WriteErrorInterface(w,
			"Class does not exists",
			http.StatusNotFound)
		return
	}
	finalFileName := procFileName
	if finalFileName == "" {
		finalFileName = fileName
	}

	if finalFileName == "" {
		errormessages.WriteErrorInterface(w,
			"There is no file for the class",
			http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "text/markdown")
	http.ServeFile(w, r, finalFileName)
}

// WriteFile is an endpoint to upload and process markdown text
func WriteFile(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	claims := r.Context().Value(session.ClaimsKey).(*session.Claims)
	if claims.Role != "TEACHER" {
		errormessages.WriteErrorInterface(w, "Not enough privileges",
			http.StatusUnauthorized)
		return
	}
	gradeID, err := strconv.ParseInt(p.ByName("id"), 0, 64)
	if err != nil {
		errormessages.WriteErrorMessage(w, "Invalid id",
			http.StatusBadRequest)
		return
	}
	courseID, err := strconv.ParseInt(p.ByName("courseid"), 0, 64)
	if err != nil {
		errormessages.WriteErrorMessage(w, "Invalid id",
			http.StatusBadRequest)
		return
	}
	classID, err := strconv.ParseInt(p.ByName("classid"), 0, 64)
	if err != nil {
		errormessages.WriteErrorMessage(w, "Invalid id",
			http.StatusBadRequest)
		return
	}
	r.ParseMultipartForm(10 << 20)

	file, multipartHeader, err := r.FormFile("file")
	if err != nil {
		errormessages.WriteErrorMessage(w, "Unable to get file",
			http.StatusBadRequest)
		return
	}
	defer file.Close()

	mimeType := multipartHeader.Header.Get("Content-Type")
	if mimeType != "text/markdown" {
		errormessages.WriteErrorMessage(w, "Invalid file",
			http.StatusBadRequest)
		return
	}

	gradeIDDir := strconv.FormatInt(gradeID, 10) + "/"
	courseIDDir := strconv.FormatInt(courseID, 10) + "/"
	classIDDir := strconv.FormatInt(classID, 10) + "/"
	directory := syncDir + gradeIDDir + courseIDDir + classIDDir
	imgDirectory := syncDir + "images/" + gradeIDDir +
		courseIDDir + classIDDir
	fileName := directory + multipartHeader.Filename

	db := persistence.GetDb()
	tx, err := db.Begin()
	if err != nil {
		errormessages.WriteErrorMessage(w, "Unable to reach database",
			http.StatusInternalServerError)
		tx.Rollback()
		return
	}
	updateQuery := `
		UPDATE text_class
		SET file_name = ?
		WHERE id = ?
	`
	res, err := tx.Exec(updateQuery, fileName, classID)
	if err != nil {
		errormessages.WriteErrorMessage(w, "Invalid course id",
			http.StatusInternalServerError)
		tx.Rollback()
		return
	}
	rowsAffected, err := res.RowsAffected()
	if rowsAffected != 1 {
		errormessages.WriteErrorInterface(w, "Id not found",
			http.StatusNotFound)
		tx.Rollback()
		return

	}

	if _, err = os.Stat(directory); os.IsNotExist(err) {
		err = os.Mkdir(directory, 0700)
		if err != nil {
			if os.IsNotExist(err) {
				errormessages.WriteErrorMessage(w,
					"Invalid grade/course id",
					http.StatusInternalServerError)
				tx.Rollback()
				return

			}
			errormessages.WriteErrorMessage(w,
				"Unable to create directory",
				http.StatusInternalServerError)
			tx.Rollback()
			return
		}
		err = os.Mkdir(imgDirectory, 0700)
		if err != nil {
			errormessages.WriteErrorMessage(w,
				"Unable to create images directory",
				http.StatusInternalServerError)
			tx.Rollback()
			return
		}
	}

	osFile, err := os.OpenFile(fileName, os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		errormessages.WriteErrorMessage(w, "Could not open os file",
			http.StatusBadRequest)
		tx.Rollback()
		return
	}
	defer osFile.Close()
	io.Copy(osFile, file)
	err = tx.Commit()
	if err != nil {
		errString := "Unable to update text class"
		errormessages.WriteErrorMessage(w, errString,
			http.StatusInternalServerError)
		tx.Rollback()
		return
	}
	w.WriteHeader(http.StatusNoContent)
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
	claims := r.Context().Value(session.ClaimsKey).(*session.Claims)
	if claims.Role != "TEACHER" {
		errormessages.WriteErrorInterface(w, "Not enough privileges",
			http.StatusUnauthorized)
		return
	}
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
	claims := r.Context().Value(session.ClaimsKey).(*session.Claims)
	if claims.Role != "TEACHER" {
		errormessages.WriteErrorInterface(w, "Not enough privileges",
			http.StatusUnauthorized)
		return
	}
	gradeID, err := strconv.ParseInt(p.ByName("id"), 0, 64)
	if err != nil {
		errormessages.WriteErrorMessage(w, "Invalid id",
			http.StatusBadRequest)
		return
	}
	courseID, err := strconv.ParseInt(p.ByName("courseid"), 0, 64)
	if err != nil {
		errormessages.WriteErrorMessage(w, "Invalid id",
			http.StatusBadRequest)
		return
	}
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

	db := persistence.GetDb()
	deleteQuery := `
		DELETE FROM text_class
		WHERE id = ?
	`
	tx, err := db.Begin()
	if err != nil {
		errormessages.WriteErrorMessage(w, "Unable to reach database",
			http.StatusInternalServerError)
		return
	}
	res, err := tx.Exec(deleteQuery, classID)
	if err != nil {
		errormessages.WriteErrorMessage(w, "Unable to delete",
			http.StatusInternalServerError)
		tx.Rollback()
		return
	}
	affectedRows, err := res.RowsAffected()
	if affectedRows == 0 {
		errormessages.WriteErrorMessage(w, "Invalid id",
			http.StatusBadRequest)
		tx.Rollback()
		return
	}

	gradeIDDir := strconv.FormatInt(gradeID, 10) + "/"
	courseIDDir := strconv.FormatInt(courseID, 10) + "/"
	classIDDir := strconv.FormatInt(classID, 10) + "/"
	directory := syncDir + gradeIDDir + courseIDDir + classIDDir
	imgDirectory := syncDir + "images/" + gradeIDDir +
		courseIDDir + classIDDir

	if err = os.RemoveAll(directory); err != nil {
		errormessages.WriteErrorInterface(w, "Unable to remove class",
			http.StatusUnauthorized)
		tx.Rollback()
		return
	}

	if err = os.RemoveAll(imgDirectory); err != nil {
		errormessages.WriteErrorInterface(w, "Unable to remove class",
			http.StatusUnauthorized)
		tx.Rollback()
		return
	}

	err = tx.Commit()
	if err != nil {
		errString := "Unable to remove class"
		errormessages.WriteErrorMessage(w, errString,
			http.StatusInternalServerError)
		tx.Rollback()
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
