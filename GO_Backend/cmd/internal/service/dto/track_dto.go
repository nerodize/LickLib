package dto

type TrackDTO struct {
	UserID      uint
	Title       string
	Description string
	FileExt     string
	SizeBytes   int64
	FileURL     string
}
