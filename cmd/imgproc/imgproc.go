package main

import (
	"flag"
	"github.com/chromz/wiki-backend/internal/ticker"
	"github.com/chromz/wiki-backend/pkg/log"
	"github.com/chromz/wiki-backend/pkg/persistence"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	logger := log.GetLogger()
	defer logger.Sync()
	dbPath := flag.String("D", "./ecommunity.db",
		"imgproc -d [PATH TO DATABASE]")
	directory := flag.String("dir", "sync/", "imgproc -dir [DIR PATH]")
	pollingRate := flag.Int("p", 5000, "imgproc -p [POLLING RATE]")
	basePath := flag.String("b", "http://localhost:3000/static/",
		"imgproc -b [BASE PATH]")
	flag.Parse()
	if (*directory)[len(*directory)-1] != '/' {
		*directory += "/"
	}
	logger.InitMessage("imageproc", "with directory "+*directory)
	persistence.SetDbPath(*dbPath)
	ticker := ticker.NewTicker(*basePath, *directory, *pollingRate)
	ticker.Run()
}
