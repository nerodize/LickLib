package pg

import (
	models "LickLib/cmd/internal/entity"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	dsn := "host=127.0.0.1 user=test password=test dbname=test port=5434 sslmode=disable"

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	// Clean Slate
	db.Exec("DROP SCHEMA public CASCADE")
	db.Exec("CREATE SCHEMA public")
	db.Exec("GRANT ALL ON SCHEMA public TO test")
	db.Exec("GRANT ALL ON SCHEMA public TO public")

	// Migrations
	_, filename, _, _ := runtime.Caller(0)
	projectRoot := filepath.Join(filepath.Dir(filename), "../../../..")
	migrationPath := filepath.Join(projectRoot, "migrations")

	migrationFS := os.DirFS(migrationPath)

	sourceDriver, err := iofs.New(migrationFS, ".")
	if err != nil {
		t.Fatalf("Failed to create source driver: %v", err)
	}

	databaseURL := "postgres://test:test@127.0.0.1:5434/test?sslmode=disable"

	m, err := migrate.NewWithSourceInstance("iofs", sourceDriver, databaseURL)
	if err != nil {
		t.Fatalf("Failed to create migrate instance: %v", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	return db
}

func cleanupTestDB(t *testing.T, db *gorm.DB) {
	db.Exec("TRUNCATE TABLE tracks CASCADE")
	db.Exec("TRUNCATE TABLE users CASCADE")
}

func createTestUser(t *testing.T, db *gorm.DB) uuid.UUID {
	userID := uuid.New()
	err := db.Exec("INSERT INTO users (id, username) VALUES (?, ?)", userID, "testuser").Error
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}
	return userID
}

// ===== TESTS =====

func TestCreateTrack(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	repo := NewTrackRepoGorm(db)
	userID := createTestUser(t, db)

	track := &models.Track{
		ID:          uuid.New(),
		UserID:      userID,
		Title:       "Test Track",
		Description: "Test Description",
		Difficulty:  models.PtrDifficulty(models.DifficultyEasy),
		Status:      models.TrackStatusUploading,
		FileExt:     ".mp3",
		SizeBytes:   1024,
	}

	err := repo.CreateTrack(track)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	var found models.Track
	result := db.First(&found, "id = ?", track.ID)
	if result.Error != nil {
		t.Errorf("track not found in database: %v", result.Error)
	}

	if found.Title != "Test Track" {
		t.Errorf("expected title 'Test Track', got '%s'", found.Title)
	}
}

func TestUpdateTrack_StatusTransition(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	repo := NewTrackRepoGorm(db)
	userID := createTestUser(t, db)

	track := &models.Track{
		ID:          uuid.New(),
		UserID:      userID,
		Title:       "Test Track",
		Description: "Test",
		Status:      models.TrackStatusUploading,
		FileExt:     ".mp3",
		SizeBytes:   1024,
	}
	repo.CreateTrack(track)

	err := repo.UpdateTrack(track.ID, map[string]interface{}{
		"status":      models.TrackStatusReady,
		"storage_key": "users/123/tracks/456.mp3",
	})

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	var updated models.Track
	db.First(&updated, "id = ?", track.ID)

	if updated.Status != models.TrackStatusReady {
		t.Errorf("expected status READY, got %v", updated.Status)
	}
	if updated.StorageKey != "users/123/tracks/456.mp3" {
		t.Errorf("expected storage_key set, got '%s'", updated.StorageKey)
	}
}

func TestDeleteFailedTracksByTitle(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	repo := NewTrackRepoGorm(db)
	userID := createTestUser(t, db) // ✅

	track1 := &models.Track{
		ID:          uuid.New(),
		UserID:      userID,
		Title:       "My Track",
		Description: "Test",
		Status:      models.TrackStatusFailed,
		FileExt:     ".mp3",
		SizeBytes:   1024,
	}
	track2 := &models.Track{
		ID:          uuid.New(),
		UserID:      userID,
		Title:       "My Track",
		Description: "Test",
		Status:      models.TrackStatusFailed,
		FileExt:     ".mp3",
		SizeBytes:   1024,
	}
	track3 := &models.Track{
		ID:          uuid.New(),
		UserID:      userID,
		Title:       "My Track",
		Description: "Test",
		Status:      models.TrackStatusReady,
		FileExt:     ".mp3",
		SizeBytes:   1024,
	}

	repo.CreateTrack(track1)
	repo.CreateTrack(track2)
	repo.CreateTrack(track3)

	err := repo.DeleteFailedTracksByTitle(userID, "My Track")
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	var remaining []models.Track
	db.Where("user_id = ? AND title = ?", userID, "My Track").Find(&remaining)

	if len(remaining) != 1 {
		t.Errorf("expected 1 track remaining, got %d", len(remaining))
	}

	if len(remaining) > 0 && remaining[0].Status != models.TrackStatusReady {
		t.Errorf("expected READY track to remain, got %v", remaining[0].Status)
	}
}

func TestUniqueConstraint_OnlyForReadyTracks(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	repo := NewTrackRepoGorm(db)
	userID := createTestUser(t, db)

	track1 := &models.Track{
		ID:          uuid.New(),
		UserID:      userID,
		Title:       "Duplicate Title",
		Description: "Test",
		Status:      models.TrackStatusReady,
		FileExt:     ".mp3",
		SizeBytes:   1024,
	}
	err := repo.CreateTrack(track1)
	if err != nil {
		t.Fatalf("failed to create first track: %v", err)
	}

	track2 := &models.Track{
		ID:          uuid.New(),
		UserID:      userID,
		Title:       "Duplicate Title",
		Description: "Test",
		Status:      models.TrackStatusReady,
		FileExt:     ".mp3",
		SizeBytes:   1024,
	}
	err = repo.CreateTrack(track2)
	if err == nil {
		t.Error("expected UNIQUE constraint error, got nil")
	}

	track3 := &models.Track{
		ID:          uuid.New(),
		UserID:      userID,
		Title:       "Duplicate Title",
		Description: "Test",
		Status:      models.TrackStatusFailed,
		FileExt:     ".mp3",
		SizeBytes:   1024,
	}
	err = repo.CreateTrack(track3)
	if err != nil {
		t.Errorf("expected no error for FAILED track, got %v", err)
	}
}
