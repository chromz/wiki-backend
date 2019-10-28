package textclass

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
	if t.CourseID <= 0 {
		return errors.New("Invalid course id")
	}
	if t.Title == "" {
		return errors.New("Title is missing")
	}
	return nil
}

// TextClassDDL query to create the text class table
const TextClassDDL = `
CREATE TABLE "text_class" (
	"id"	INTEGER,
	"course_id"	INTEGER NOT NULL,
	"file_name"	TEXT NOT NULL,
	"proc_file_name"	TEXT DEFAULT NULL,
	"title"	TEXT NOT NULL,
	FOREIGN KEY("course_id") REFERENCES "course"("id") ON DELETE CASCADE,
	PRIMARY KEY("id")
);
`

// Create creates a new textclass in db, prepares for execution
func Create(w http.ResponseWriter, r *http.Request, p httprouter.Params) {

}
