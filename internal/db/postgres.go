package db

import (
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func initPostgres(dsn string, cfg *gorm.Config) (*gorm.DB, error) {
	return gorm.Open(postgres.Open(dsn), cfg)
}
