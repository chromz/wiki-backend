package main

import (
	"flag"
	"github.com/chromz/wiki-backend/internal/routes"
	"github.com/chromz/wiki-backend/internal/textclass"
	"github.com/chromz/wiki-backend/pkg/log"
	"github.com/chromz/wiki-backend/pkg/persistence"
	_ "github.com/mattn/go-sqlite3"
	"net/http"
)

func main() {
	logger := log.GetLogger()
	defer logger.Sync()
	port := flag.String("p", "3000", "wiki -p [PORT]")
	dbPath := flag.String("D", "./ecommunity.db",
		"wiki -d [PATH TO DATABASE]")
	directory := flag.String("dir", "sync/", "wiki -dir [DIR PATH]")
	baseURI := flag.String("U", "http://localhost:3000/static/", "wiki -U [URI]")
	flag.Parse()
	logger.InitMessage("backend", "port:"+*port)
	persistence.SetDbPath(*dbPath)
	if (*directory)[len(*directory)-1] != '/' {
		*directory += "/"
	}
	textclass.NewSyncDir(*directory)
	textclass.NewBaseURI(*baseURI)
	logger.FatalError("Could not listen and serve",
		http.ListenAndServe(":"+*port, routes.RouteHandler()))
}
