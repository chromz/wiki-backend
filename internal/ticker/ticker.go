package ticker

import (
	"database/sql"
	"github.com/chromz/wiki-backend/pkg/log"
	"github.com/chromz/wiki-backend/pkg/persistence"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Synchronizer is a struct that ticks every polling rate
type Synchronizer struct {
	ticker *time.Ticker
}

var destDir string

var logger = log.GetLogger()
var db *sql.DB
var linkRegex = regexp.MustCompile(`\[.*?\]\((.*?)\)`)
var basePath string

type file struct {
	classID  int64
	courseID int64
	gradeID  int64
	fileName string
}

// NewTicker constructor of the synchronizer ticker
func NewTicker(basePathFlag, directory string, pollingRate int) *Synchronizer {
	db = persistence.GetDb()
	destDir = directory
	basePath = basePathFlag
	return &Synchronizer{
		ticker: time.NewTicker(time.Millisecond * time.Duration(pollingRate)),
	}
}

func parseMarkdown(procFile file, markdownText, dir, midDir string) (map[string]string, error) {
	processedLinks := make(map[string]string)
	linkMatches := linkRegex.FindAllStringSubmatch(markdownText, -1)
	for _, matchArray := range linkMatches {
		if len(matchArray) < 2 {
			continue
		}
		urlSplit := strings.Split(matchArray[1], " ")
		mdURL := urlSplit[0]
		if _, ok := processedLinks[mdURL]; ok {
			continue
		}

		response, err := http.Get(mdURL)
		if err != nil {
			logger.Error("Unable to download file", err)
			continue
		}
		baseName := filepath.Base(mdURL)
		extension := filepath.Ext(baseName)
		prefix := strconv.FormatInt(procFile.classID, 10) + "_"
		var fileName string
		fileName = dir + prefix + baseName
		if extension == "" {
			// Assume it is html
			fileName += ".html"
		}
		logger.Info(fileName)
		file, err := os.OpenFile(fileName, os.O_WRONLY|os.O_CREATE, 0700)
		if err != nil {
			logger.Error("Unable to open file", err)
			return nil, err
		}
		_, err = io.Copy(file, response.Body)
		if err != nil {
			logger.Error("Unable to copy file", err)
			return nil, err
		}
		dbName := basePath + midDir + prefix + baseName
		if extension == "" {
			dbName += ".html"
		}
		processedLinks[mdURL] = dbName

		file.Close()
		response.Body.Close()
		logger.Info("Resource: " + fileName + " created")
	}
	return processedLinks, nil
}

func processMarkdown(procFile file, markdownText string) {
	classIDDir := strconv.FormatInt(procFile.classID, 10) + "/"
	courseIDDir := strconv.FormatInt(procFile.courseID, 10) + "/"
	gradeIDDir := strconv.FormatInt(procFile.gradeID, 10) + "/"
	midDir := gradeIDDir + courseIDDir + classIDDir
	dir := destDir + "assets/" + midDir
	processedImages, err := parseMarkdown(procFile, markdownText, dir,
		midDir)
	var replaces []string
	for k, v := range processedImages {
		replaces = append(replaces, k)
		replaces = append(replaces, v)
	}
	replacer := strings.NewReplacer(replaces...)
	processedMarkdown := replacer.Replace(markdownText)

	baseName := filepath.Base(procFile.fileName)
	processedFileName := destDir + midDir + "processed_" + baseName
	err = ioutil.WriteFile(processedFileName,
		[]byte(processedMarkdown), 0700)
	if err != nil {
		logger.Error("Unable to write processed file", err)
		return
	}
	logger.Info("Saved processed file to " + processedFileName)
	updateQuery := `
UPDATE text_class
		SET proc_file_name = ?
		WHERE id = ?
	`
	res, err := db.Exec(updateQuery, processedFileName, procFile.classID)
	if err != nil {
		logger.Error("Unable to update text class", err)
		return
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected != 1 {
		logger.Info("Unable to update text class")
		return

	}
}

func process() {
	logger.Info("Pulling data from database")
	selectQuery := `
SELECT text_class.id as class_id, course_id, grade_id, file_name
		FROM text_class
		JOIN course ON course.id = text_class.course_id
		JOIN grade ON course.grade_id = grade.id
		WHERE text_class.proc_file_name = '';
	`
	rows, err := db.Query(selectQuery)
	if err != nil {
		logger.Info("Unable to connect to database")
		return
	}
	var rowsToProc []file
	for rows.Next() {
		procFile := file{}
		err = rows.Scan(&procFile.classID, &procFile.courseID,
			&procFile.gradeID, &procFile.fileName)
		if err != nil {
			logger.Error("Unable to row scan", err)
			return
		}
		if procFile.fileName == "" {
			continue
		}
		rowsToProc = append(rowsToProc, procFile)
	}
	rows.Close()
	for _, procFile := range rowsToProc {
		logger.Info("Processing file: " + procFile.fileName)
		data, err := ioutil.ReadFile(procFile.fileName)
		if err != nil {
			logger.Error("Error reading file", err)
			return
		}
		processMarkdown(procFile, string(data))
	}
}

// Run starts the ticker
func (synchronizer *Synchronizer) Run() {
	for {
		select {
		case <-synchronizer.ticker.C:
			process()
		}
	}
}