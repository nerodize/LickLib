package service

import (
	"context"
	"io"
)

// StorageClient definiert alle Storage-Operationen
type StorageClient interface {
	Upload(ctx context.Context, objectName string, reader io.Reader, size int64) error
	Delete(ctx context.Context, objectName string) error
	GetPresignedURL(ctx context.Context, objectName string) (string, error)
}
