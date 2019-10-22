package main

import (
	"flag"
	"github.com/chromz/wiki-backend/internal/routes"
	"github.com/chromz/wiki-backend/pkg/log"
	"github.com/chromz/wiki-backend/pkg/persistence"
	_ "github.com/mattn/go-sqlite3"
	"net/http"
)

func main() {
	logger := log.GetLogger()
	defer logger.Sync()
	port := flag.String("p", "3000", "wiki -p [PORT]")
	dbPath := flag.String("d", "./ecommunity.db",
		"wiki -d [PATH TO DATABASE]")
	flag.Parse()
	logger.InitMessage("backend", "port:"+*port)
	persistence.SetDbPath(*dbPath)
	logger.FatalError("Could not listen and serve",
		http.ListenAndServe(":"+*port, routes.RouteHandler()))
}
