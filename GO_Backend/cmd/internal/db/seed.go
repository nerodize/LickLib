package db

import (
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	models "LickLib/cmd/internal/entity"
)

func ptr[T any](v T) *T { return &v }

// Idempotent Seed: legt User + Tracks an, ohne Duplikate
func Seed(gdb *gorm.DB) error {
	return gdb.Transaction(func(tx *gorm.DB) error {
		// Beispiel-User
		users := []models.User{
			{Username: "max", Email: ptr("max@example.com"), PasswordHash: "dummy-hash"},
			{Username: "lisa", Email: ptr("lisa@example.com"), PasswordHash: "dummy-hash"},
			{Username: "tom", Email: ptr("tom@example.com"), PasswordHash: "dummy-hash"},
			{Username: "jane", Email: ptr("jane@example.com"), PasswordHash: "dummy-hash"},
			{Username: "sam", Email: ptr("sam@example.com"), PasswordHash: "dummy-hash"},
			{Username: "sara", Email: ptr("sara@example.com"), PasswordHash: "dummy-hash"},
			{Username: "alex", Email: ptr("alex@example.com"), PasswordHash: "dummy-hash"},
			{Username: "mia", Email: ptr("mia@example.com"), PasswordHash: "dummy-hash"},
			{Username: "chris", Email: ptr("chris@example.com"), PasswordHash: "dummy-hash"},
			{Username: "nico", Email: ptr("nico@example.com"), PasswordHash: "dummy-hash"},
		}

		for _, u := range users {
			if err := tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "username"}},
				DoNothing: true,
			}).Create(&u).Error; err != nil {
				return err
			}
		}

		// Schwierigkeitsgrade
		easy := models.DifficultyEasy
		medium := models.DifficultyMedium
		hard := models.DifficultyHard

		// Beispiel-Tracks pro User
		tracks := []models.Track{
			{Username: "max", Title: "Pentatonic Lick #1", Description: "A minor box 1", Difficulty: &easy, FileExt: "wav", SizeBytes: 123456},
			{Username: "max", Title: "Blues Run", Description: "Fast E minor lick", Difficulty: &medium, FileExt: "wav", SizeBytes: 98765},
			{Username: "lisa", Title: "Jazz Line", Description: "ii-V-I progression", Difficulty: &hard, FileExt: "mp3", SizeBytes: 654321},
			{Username: "tom", Title: "Sweep Practice", Description: "Arpeggio sweep exercise", Difficulty: &medium, FileExt: "wav", SizeBytes: 222222},
			{Username: "jane", Title: "Alternate Picking #3", Description: "Speed drill", Difficulty: &hard, FileExt: "wav", SizeBytes: 300000},
			{Username: "sam", Title: "Funk Groove", Description: "E7#9 rhythm pattern", Difficulty: &easy, FileExt: "mp3", SizeBytes: 150000},
			{Username: "sara", Title: "Legato Flow", Description: "Three-note-per-string run", Difficulty: &medium, FileExt: "wav", SizeBytes: 250000},
			{Username: "alex", Title: "Classic Rock Riff", Description: "E major intro", Difficulty: &easy, FileExt: "mp3", SizeBytes: 180000},
			{Username: "mia", Title: "Metal Gallop", Description: "Right-hand workout", Difficulty: &hard, FileExt: "wav", SizeBytes: 350000},
			{Username: "chris", Title: "Chord Melody", Description: "Jazz chord solo", Difficulty: &medium, FileExt: "mp3", SizeBytes: 270000},
			{Username: "nico", Title: "Blues Turnaround", Description: "Classic E turnaround", Difficulty: &easy, FileExt: "wav", SizeBytes: 90000},
		}

		for _, t := range tracks {
			if err := tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "username"}, {Name: "title"}},
				DoNothing: true,
			}).Create(&t).Error; err != nil {
				return err
			}
		}

		return nil
	})
}
