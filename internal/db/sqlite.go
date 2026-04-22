package db

import (
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func initSQLite(dsn string, cfg *gorm.Config) (*gorm.DB, error) {
	return gorm.Open(sqlite.Open(dsn), cfg)
}
