package service

import (
	"context"
	"io"

	"github.com/google/uuid"
)

// StorageClient definiert alle Storage-Operationen
type StorageClient interface {
	Upload(ctx context.Context, objectName string, reader io.Reader, size int64) error
	Delete(ctx context.Context, objectName string) error
	GenerateTrackKey(userID, trackID uuid.UUID, ext string) string
	GetPresignedURL(ctx context.Context, objectName string) (string, error)
}
