package db

import (
	models "LickLib/cmd/internal/entity"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func ptr[T any](v T) *T { return &v }

func Seed(gdb *gorm.DB) error {
	return gdb.Transaction(func(tx *gorm.DB) error {
		// 1. User seeden
		users := []models.User{
			// ob das so toll ist mit der Mail
			{ID: uuid.New(), Username: "max", Email: ptr("max@example.com")},
			{ID: uuid.New(), Username: "lisa", Email: ptr("lisa@example.com")},
			{ID: uuid.New(), Username: "tom", Email: ptr("tom@example.com")},
			{ID: uuid.New(), Username: "jane", Email: ptr("jane@example.com")},
			{ID: uuid.New(), Username: "sam", Email: ptr("sam@example.com")},
			{ID: uuid.New(), Username: "sara", Email: ptr("sara@example.com")},
			{ID: uuid.New(), Username: "alex", Email: ptr("alex@example.com")},
			{ID: uuid.New(), Username: "mia", Email: ptr("mia@example.com")},
			{ID: uuid.New(), Username: "chris", Email: ptr("chris@example.com")},
			{ID: uuid.New(), Username: "nico", Email: ptr("nico@example.com")},
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

		userIDByName := make(map[string]uuid.UUID, len(dbUsers))
		for _, u := range dbUsers {
			userIDByName[u.Username] = u.ID // u.ID ist jetzt ein String
		}

		// 3. Schwierigkeitsgrade
		easy := models.DifficultyEasy
		medium := models.DifficultyMedium
		hard := models.DifficultyHard

		// 4. Tracks mit UserID statt Username
		tracks := []models.Track{
			{
				ID:          uuid.New(),
				UserID:      userIDByName["max"],
				Title:       "Pentatonic Lick #1",
				Description: "A minor box 1",
				Difficulty:  &easy,
				FileExt:     "wav",
				SizeBytes:   123456,
				StorageKey:  "seed-max-pentatonic-1.wav", // Dummy Key
			},
			{
				ID:          uuid.New(),
				UserID:      userIDByName["max"],
				Title:       "Blues Run",
				Description: "Fast E minor lick",
				Difficulty:  &medium,
				FileExt:     "wav",
				SizeBytes:   98765,
				StorageKey:  "seed-max-blues-run.wav",
			},
			{
				ID:          uuid.New(),
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
				ID:          uuid.New(),
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
