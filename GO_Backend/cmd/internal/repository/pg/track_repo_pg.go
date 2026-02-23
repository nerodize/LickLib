package pg

import (
	models "LickLib/cmd/internal/entity"
	"LickLib/cmd/internal/repository"

	"github.com/google/uuid"

	"gorm.io/gorm"
)

type TrackRepoGorm struct {
	db *gorm.DB
}

var _ repository.TrackRepository = &TrackRepoGorm{}

func NewTrackRepoGorm(db *gorm.DB) *TrackRepoGorm {
	return &TrackRepoGorm{db: db}
}
func (r *TrackRepoGorm) FindByID(id uuid.UUID) (*models.Track, error) {
	var track models.Track
	if err := r.db.First(&track, id).Error; err != nil {
		return nil, err // could be gorm.ErrRecordNotFound or other DB error
	}
	return &track, nil
}

func (r *TrackRepoGorm) FindByUsername(username string) ([]models.Track, error) {
	var tracks []models.Track
	err := r.db.
		Joins("User").                            // Notwendig zum Filtern nach "username"
		Preload("User").                          // Notwendig, damit das User-Feld im Struct befüllt wird
		Where("\"User\".username = ?", username). // Name der Relation/Tabelle beachten
		Order("tracks.created_at desc").
		Find(&tracks).Error
	return tracks, err
}

func (r *TrackRepoGorm) FindByUserID(userID uuid.UUID) ([]models.Track, error) {
	var tracks []models.Track
	if err := r.db.
		Joins("User").
		Where("user_id = ?", userID).
		Order("tracks.created_at desc").
		Find(&tracks).Error; err != nil {
		return nil, err
	}
	return tracks, nil
}

var _ repository.TrackRepository = &TrackRepoGorm{}

func (r *TrackRepoGorm) CreateTrack(track *models.Track) error {
	return r.db.Create(track).Error
}

func (r *TrackRepoGorm) DeleteTrack(id uuid.UUID) error {
	return r.db.Delete(&models.Track{}, id).Error
	// vllt fehlt hier ByID oder eine Art zur Autorisierung
}

// erlaubt alle x-beliebige value types
func (r *TrackRepoGorm) UpdateTrack(id uuid.UUID, updates map[string]interface{}) error {
	// Updates führt nur die Änderungen aus, die in der Map stehen
	return r.db.Model(&models.Track{}).Where("id = ?", id).Updates(updates).Error
}
