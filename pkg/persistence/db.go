package persistence

import (
	"database/sql"
	"github.com/chromz/wiki-backend/pkg/log"
	"sync"
)

var (
	db     *sql.DB
	dbPath string
	once   sync.Once
	logger = log.GetLogger()
)

// SetDbPath sets the sqlite file path
func SetDbPath(path string) {
	dbPath = path
}

// GetDb get a single instance of the sqlite database
func GetDb() *sql.DB {
	once.Do(func() {
		var err error
		db, err = sql.Open("sqlite3", dbPath+"?_foreign_keys=on")
		if err != nil {
			logger.Error("Unable to open sqlite database", err)
		}
	})
	return db
}
