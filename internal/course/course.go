package course

import (
	"encoding/json"
	"errors"
	"github.com/chromz/wiki-backend/internal/session"
	"github.com/chromz/wiki-backend/internal/textclass"
	"github.com/chromz/wiki-backend/pkg/errormessages"
	"github.com/chromz/wiki-backend/pkg/pagination"
	"github.com/chromz/wiki-backend/pkg/persistence"
	"github.com/julienschmidt/httprouter"
	"github.com/mattn/go-sqlite3"
	"net/http"
	"os"
	"strconv"
)

// CourseDDL is the query to create the clasroom table
const CourseDDL = `
CREATE TABLE IF NOT EXISTS "classroom" (
	"id"	INTEGER PRIMARY KEY AUTOINCREMENT UNIQUE,
	"grade_id"	INTEGER NOT NULL,
	"name"	TEXT NOT NULL,
	"description"	TEXT,
	FOREIGN KEY("grade_id") REFERENCES "grade"("id") ON DELETE CASCADE
);
`

// Course struct that represents a course in a grade
type Course struct {
	ID          int64  `json:"id"`
	GradeID     int64  `json:"gradeId"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// Validate validates the integrity of Course
func (c *Course) Validate() error {
	if c.Name == "" {
		return errors.New("Invalid name")
	}
	if c.GradeID <= 0 {
		return errors.New("Invalid grade id")
	}
	return nil

}

// Create is an endpoint to create a course
func Create(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	claims := r.Context().Value(session.ClaimsKey).(*session.Claims)
	if claims.Role != "TEACHER" {
		errormessages.WriteErrorInterface(w, "Not enough privileges",
			http.StatusUnauthorized)
		return
	}
	course := &Course{}
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(course)
	if err != nil {
		errormessages.WriteErrorMessage(w, "Invalid body type",
			http.StatusBadRequest)
		return
	}

	gradeID, err := strconv.ParseInt(p.ByName("id"), 0, 64)
	if err != nil {
		errormessages.WriteErrorMessage(w, "Invalid grade id",
			http.StatusBadRequest)
		return

	}
	course.GradeID = gradeID
	if err = course.Validate(); err != nil {
		errormessages.WriteErrorMessage(w, "Invalid course data",
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

	insertQuery := `
		INSERT INTO course(grade_id, name, description)
		VALUES(?, ?, ?)
	`
	res, err := tx.Exec(insertQuery, course.GradeID, course.Name,
		course.Description)
	if err != nil {
		if sqliteErr, ok := err.(sqlite3.Error); ok {
			if sqliteErr.ExtendedCode == sqlite3.ErrConstraintForeignKey {
				errormessages.WriteErrorMessage(w,
					"Invalid grade id",
					http.StatusBadRequest)
				tx.Rollback()
				return
			}
		}
		errormessages.WriteErrorMessage(w, "Unable to add course",
			http.StatusInternalServerError)
		tx.Rollback()
		return
	}
	course.ID, _ = res.LastInsertId()
	dirName := textclass.SyncDir() +
		strconv.FormatInt(gradeID, 10) + "/" +
		strconv.FormatInt(course.ID, 10) + "/"
	if err = os.Mkdir(dirName, 0700); err != nil {
		errormessages.WriteErrorMessage(w, "Unable  to create course",
			http.StatusInternalServerError)
		tx.Rollback()
		return
	}
	imgDirName := textclass.SyncDir() + "assets/" +
		strconv.FormatInt(gradeID, 10) + "/" +
		strconv.FormatInt(course.ID, 10) + "/"
	if err = os.Mkdir(imgDirName, 0700); err != nil {
		errormessages.WriteErrorMessage(w, "Unable  to create course",
			http.StatusInternalServerError)
		tx.Rollback()
		return
	}
	err = tx.Commit()
	if err != nil {
		errString := "Unable to add course"
		errormessages.WriteErrorMessage(w, errString,
			http.StatusInternalServerError)
		tx.Rollback()
		return
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(course)
}

// Read returns available courses, paginated
func Read(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	params := r.URL.Query()

	gradeID, err := strconv.ParseInt(p.ByName("id"), 0, 64)
	if err != nil {
		errormessages.WriteErrorMessage(w, "Invalid grade id",
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
		SELECT id, grade_id, name, description
		FROM course
		WHERE id > ?
		AND grade_id = ?
		LIMIT ?
	`

	rows, err := db.Query(findQuery, page.NextToken, gradeID, page.Size)
	if err != nil {
		errormessages.WriteErrorMessage(w, "Unable to find courses",
			http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	var courses []Course
	for rows.Next() {
		course := Course{}
		err = rows.Scan(&course.ID, &course.GradeID,
			&course.Name, &course.Description)
		if err != nil {
			errormessages.WriteErrorMessage(w,
				"Unable to find courses",
				http.StatusInternalServerError)
			return
		}
		courses = append(courses, course)
	}
	page.Data = courses
	coursesCount := len(courses)
	if coursesCount > 0 {
		page.NextToken = courses[coursesCount-1].ID
	} else {
		page.NextToken = -1
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(page)
}

// Update updates a course resource
func Update(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	claims := r.Context().Value(session.ClaimsKey).(*session.Claims)
	if claims.Role != "TEACHER" {
		errormessages.WriteErrorInterface(w, "Not enough privileges",
			http.StatusUnauthorized)
		return
	}
	course := &Course{}
	decoder := json.NewDecoder(r.Body)
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

	err = decoder.Decode(course)
	course.ID = courseID
	course.GradeID = gradeID
	if err != nil {
		errormessages.WriteErrorMessage(w, "Invalid body type",
			http.StatusBadRequest)
		return
	}

	if err = course.Validate(); err != nil {
		errormessages.WriteErrorMessage(w, "Invalid course",
			http.StatusBadRequest)
		return
	}

	if course.ID <= 0 {
		errormessages.WriteErrorMessage(w, "ID is required",
			http.StatusBadRequest)
		return
	}

	db := persistence.GetDb()
	updateQuery := `
		UPDATE course
		SET name = ?, description = ?
		WHERE id = ?
	`
	res, err := db.Exec(updateQuery, course.Name, course.Description,
		course.ID)
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

// Delete endpoint to delete a specifig course
func Delete(w http.ResponseWriter, r *http.Request, p httprouter.Params) {

	claims := r.Context().Value(session.ClaimsKey).(*session.Claims)
	if claims.Role != "TEACHER" {
		errormessages.WriteErrorInterface(w, "Not enough privileges",
			http.StatusUnauthorized)
		return
	}
	gradeID, err := strconv.ParseInt(p.ByName("id"), 0, 64)
	if err != nil {
		errormessages.WriteErrorMessage(w, "Invalid grade id",
			http.StatusBadRequest)
		return

	}
	courseID, err := strconv.ParseInt(p.ByName("courseid"), 0, 64)
	if err != nil {
		errormessages.WriteErrorMessage(w, "Invalid id",
			http.StatusBadRequest)
		return

	}

	if courseID <= 0 {
		errormessages.WriteErrorMessage(w, "ID is required",
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
	deleteQuery := `
		DELETE FROM course
		WHERE id = ?
	`
	res, err := tx.Exec(deleteQuery, courseID)
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

	dirName := textclass.SyncDir() +
		strconv.FormatInt(gradeID, 10) + "/" +
		strconv.FormatInt(courseID, 10) + "/"
	if err = os.RemoveAll(dirName); err != nil {
		errormessages.WriteErrorMessage(w, "Unable  to create course",
			http.StatusInternalServerError)
		tx.Rollback()
		return
	}
	imgDirName := textclass.SyncDir() + "assets/" +
		strconv.FormatInt(gradeID, 10) + "/" +
		strconv.FormatInt(courseID, 10) + "/"
	if err = os.RemoveAll(imgDirName); err != nil {
		errormessages.WriteErrorMessage(w, "Unable  to create course",
			http.StatusInternalServerError)
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
