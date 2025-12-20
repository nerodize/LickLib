package pg

import (
	models "LickLib/cmd/internal/entity"
	"LickLib/cmd/internal/repository"

	"gorm.io/gorm"
)

type TrackRepoGorm struct {
	db *gorm.DB
}

var _ repository.TrackRepository = &TrackRepoGorm{}

func NewTrackRepoGorm(db *gorm.DB) *TrackRepoGorm {
	return &TrackRepoGorm{db: db}
}
func (r *TrackRepoGorm) FindByID(id uint) (*models.Track, error) {
	var track models.Track
	if err := r.db.First(&track, id).Error; err != nil {
		return nil, err // could be gorm.ErrRecordNotFound or other DB error
	}
	return &track, nil
}

func (r *TrackRepoGorm) FindByUsername(username string) ([]models.Track, error) {
	var tracks []models.Track
	if err := r.db.
		Joins("User").
		Where("username = ?", username).
		Order("tracks.created_at desc").
		Find(&tracks).Error; err != nil {
		return nil, err
	}
	return tracks, nil

}
