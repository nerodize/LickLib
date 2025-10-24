package db

import (
	"os"

	"gorm.io/gorm"
)

// kinda pointless
// TODO: might be able to remove this safely
func AutoMigrate(gdb *gorm.DB, models ...any) error {
	// In DEV ok; in PROD meist Migrationstool verwenden
	if os.Getenv("APP_ENV") == "dev" {
		return gdb.AutoMigrate(models...)
	}
	return nil
}
