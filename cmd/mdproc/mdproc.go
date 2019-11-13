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
		"mdproc -d [PATH TO DATABASE]")
	directory := flag.String("dir", "sync/", "mdproc -dir [DIR PATH]")
	pollingRate := flag.Int("p", 5000, "mdproc -p [POLLING RATE]")
	basePath := flag.String("b", "http://localhost:3000/static/",
		"mdproc -b [BASE PATH]")
	userAgent := flag.String("A", "", "mdproc -A [USER AGENT]")
	flag.Parse()
	if (*directory)[len(*directory)-1] != '/' {
		*directory += "/"
	}
	logger.InitMessage("mdproc", "with directory "+*directory)
	persistence.SetDbPath(*dbPath)
	ticker := ticker.NewTicker(*basePath, *userAgent,
		*directory, *pollingRate)
	ticker.Run()
}
