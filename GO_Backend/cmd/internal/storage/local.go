package storage

import (
	"io"
	"os"
	"path/filepath"
)

type LocalStorage struct {
	BasePath string
	BaseURL  string
}

func NewLocalStorage(basePath, baseURL string) *LocalStorage {
	// Filemode => Linuxberechtigungen => webserver darf lesen
	_ = os.MkdirAll(basePath, 0o755)
	return &LocalStorage{BasePath: basePath, BaseURL: baseURL}
}

func (s *LocalStorage) Save(r io.Reader, destPath string) (int64, error) {
	fullPath := filepath.Join(s.BasePath, filepath.FromSlash(destPath))
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		return 0, err
	}

	f, err := os.Create(fullPath)
	if err != nil {
		return 0, err
	}
	// close erst beim exit dieser Funktion aufrufen
	defer f.Close()
	n, err := io.Copy(f, r)
	return n, err
}

func (s *LocalStorage) URL(destPath string) string {
	return s.BaseURL + "/" + filepath.ToSlash(destPath)
}

func (s *LocalStorage) PresignPut(destPath string, ttlSeconds int64) (string, error) {
	return "", nil
}
