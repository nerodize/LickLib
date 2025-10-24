package db

import (
	"database/sql"
	"os"
)

func RunSQLFile(sqlDB *sql.DB, path string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	_, err = sqlDB.Exec(string(b))
	return err
}
