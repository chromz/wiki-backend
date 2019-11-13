package ticker

import (
	"database/sql"
	"github.com/PuerkitoBio/goquery"
	"github.com/chromz/wiki-backend/pkg/log"
	"github.com/chromz/wiki-backend/pkg/persistence"
	"github.com/gocolly/colly"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
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
var userAgent string

type file struct {
	classID  int64
	courseID int64
	gradeID  int64
	fileName string
}

// NewTicker constructor of the synchronizer ticker
func NewTicker(basePathFlag, userAgentFlag, directory string,
	pollingRate int) *Synchronizer {
	db = persistence.GetDb()
	destDir = directory
	basePath = basePathFlag
	userAgent = userAgentFlag
	return &Synchronizer{
		ticker: time.NewTicker(time.Millisecond * time.Duration(pollingRate)),
	}
}

func isRelative(testURL string) bool {
	u, err := url.Parse(testURL)
	return err == nil && u.Scheme == "" && u.Host == ""
}

func onAsset(processedLinks map[string]string, assetCount *int,
	midDir, fileName, postfix, attribute string, urlStruct *url.URL,
	client *http.Client) colly.HTMLCallback {
	resFolder := filepath.Base(fileName) + "_resources/"
	fileName += "_resources/"
	err := os.MkdirAll(fileName, 0700)
	if err != nil {
		logger.Error("Unable to create resources dir", err)
		return nil
	}
	return func(e *colly.HTMLElement) {
		link := e.Attr(attribute)
		var downloadLink string
		if len(link) > 2 && link[0] == '/' && link[1] == '/' {
			link = "https:" + link
		}
		if isRelative(link) {
			downloadLink = urlStruct.Scheme + "://" +
				urlStruct.Host + link
		} else {
			downloadLink = link
		}
		if _, ok := processedLinks[link]; ok {
			return
		}
		req, err := http.NewRequest("GET", downloadLink, nil)
		if err != nil {
			logger.Error("Unable to create request", err)
			return
		}
		req.Header.Set("User-Agent", userAgent)
		response, err := client.Do(req)
		if err != nil {
			logger.Error("Unable to download file", err)
			return
		}
		if err != nil {
			logger.Error("Unable to download file", err)
			return
		}

		assetFileName := fileName
		baseName := strconv.Itoa(*assetCount) + postfix
		dbName := basePath + midDir + resFolder + baseName
		processedLinks[link] = dbName
		e.DOM.SetAttr(attribute, dbName)
		assetFileName += baseName
		*assetCount++
		file, err := os.OpenFile(assetFileName,
			os.O_WRONLY|os.O_CREATE, 0700)
		if err != nil {
			logger.Error("Unable to open file", err)
			return
		}
		_, err = io.Copy(file, response.Body)
		if err != nil {
			logger.Error("Unable to copy file", err)
			return
		}
		logger.Info("Downloaded asset: " + assetFileName)
		file.Close()
		response.Body.Close()
	}
}

func afterScrap(fileName string) colly.HTMLCallback {
	return func(e *colly.HTMLElement) {
		htmlContent, _ := goquery.OuterHtml(e.DOM)
		file, err := os.OpenFile(fileName,
			os.O_WRONLY|os.O_CREATE, 0700)
		if err != nil {
			logger.Error("Unable to open file", err)
			return
		}
		file.Write([]byte(htmlContent))
		logger.Info("Processed web page written: " + fileName)
		file.Close()
	}
}

func parseMarkdown(procFile file, markdownText,
	dir, midDir string) (map[string]string, error) {
	processedLinks := make(map[string]string)
	linkMatches := linkRegex.FindAllStringSubmatch(markdownText, -1)
	client := &http.Client{}
	collector := colly.NewCollector()
	for _, matchArray := range linkMatches {
		if len(matchArray) < 2 {
			continue
		}
		urlSplit := strings.Split(matchArray[1], " ")
		mdURL := urlSplit[0]

		if _, ok := processedLinks[mdURL]; ok {
			continue
		}

		urlStruct, err := url.Parse(mdURL)
		if err != nil {
			return nil, err
		}

		req, err := http.NewRequest("GET", mdURL, nil)
		if err != nil {
			logger.Error("Unable to create request", err)
			return nil, err
		}
		req.Header.Set("User-Agent", userAgent)

		baseName := filepath.Base(mdURL)
		extension := filepath.Ext(baseName)
		prefix := strconv.FormatInt(procFile.classID, 10) + "_"
		var fileName string
		var styleCount int
		fileName = dir + prefix + baseName
		if extension == "" {
			// Assume it is html
			htmlLinks := make(map[string]string)
			collector.OnHTML(`link[rel="stylesheet"]`,
				onAsset(
					htmlLinks,
					&styleCount,
					midDir,
					fileName,
					"_style.css",
					"href",
					urlStruct,
					client,
				))
			collector.OnHTML(`img`,
				onAsset(
					htmlLinks,
					&styleCount,
					midDir,
					fileName,
					"_image",
					"src",
					urlStruct,
					client,
				))
			fileName += ".html"
			processedLinks[mdURL] = basePath + midDir +
				prefix + baseName + ".html"
			collector.OnHTML("html", afterScrap(fileName))
			collector.Visit(mdURL)
		} else {
			logger.Info(fileName)
			response, err := client.Do(req)
			if err != nil {
				logger.Error("Unable to download file", err)
				continue
			}
			file, err := os.OpenFile(fileName,
				os.O_WRONLY|os.O_CREATE, 0700)
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
			processedLinks[mdURL] = dbName
			file.Close()
			response.Body.Close()
		}

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
