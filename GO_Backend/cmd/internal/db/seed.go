package db

import (
	models "LickLib/cmd/internal/entity"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func ptr[T any](v T) *T { return &v }

func Seed(gdb *gorm.DB) error {
	return gdb.Transaction(func(tx *gorm.DB) error {
		// 1. User seeden
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

		// 2. User aus DB holen und Username → ID mappen
		var dbUsers []models.User
		if err := tx.Find(&dbUsers).Error; err != nil {
			return err
		}

		userIDByName := make(map[string]int, len(dbUsers))
		for _, u := range dbUsers {
			userIDByName[u.Username] = u.ID
		}

		// 3. Schwierigkeitsgrade
		easy := models.DifficultyEasy
		medium := models.DifficultyMedium
		hard := models.DifficultyHard

		// 4. Tracks mit UserID statt Username
		tracks := []models.Track{
			{
				UserID:      userIDByName["max"],
				Title:       "Pentatonic Lick #1",
				Description: "A minor box 1",
				Difficulty:  &easy,
				FileExt:     "wav",
				SizeBytes:   123456,
				StorageKey:  "seed-max-pentatonic-1.wav", // Dummy Key
			},
			{
				UserID:      userIDByName["max"],
				Title:       "Blues Run",
				Description: "Fast E minor lick",
				Difficulty:  &medium,
				FileExt:     "wav",
				SizeBytes:   98765,
				StorageKey:  "seed-max-blues-run.wav",
			},
			{
				UserID:      userIDByName["lisa"],
				Title:       "Jazz Line",
				Description: "ii-V-I progression",
				Difficulty:  &hard,
				FileExt:     "mp3",
				SizeBytes:   654321,
				StorageKey:  "seed-lisa-jazz-line.mp3",
			},
			// ... fülle die restlichen Tracks analog mit StorageKey auf
			{
				UserID:      userIDByName["nico"],
				Title:       "Blues Turnaround",
				Description: "Classic E turnaround",
				Difficulty:  &easy,
				FileExt:     "wav",
				SizeBytes:   90000,
				StorageKey:  "seed-nico-blues.wav",
			},
		}

		for _, t := range tracks {
			// WICHTIG: Wenn du OnConflict nutzt, musst du storage_key eventuell
			// ausschließen oder mit updaten, falls sich der Titel nicht geändert hat.
			if err := tx.Clauses(clause.OnConflict{
				Columns: []clause.Column{
					{Name: "user_id"},
					{Name: "title"},
				},
				DoNothing: true,
			}).Create(&t).Error; err != nil {
				return err
			}
		}

		return nil
	})
}
