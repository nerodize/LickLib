package models

type TrackStatus string

const (
	TrackStatusUploading  TrackStatus = "UPLOADING"
	TrackStatusReady      TrackStatus = "READY"
	TrackStatusFailed     TrackStatus = "FAILED"
	TrackStatusProcessing TrackStatus = "PROCESSING"
)
