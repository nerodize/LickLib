package storage

import "io"

type Storage interface {
	Save(reader io.Reader, destPath string) (int64, error)
	URL(destPath string) string
	PresignPut(destPath string, ttlSeconds int64) (string, error)
}
