package storage

import (
	"LickLib/cmd/internal/config" // Import deiner Config-Struktur
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/url"
	"time"

	"github.com/google/uuid"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// MinioClient kapselt den echten Client
type MinioClient struct {
	Client     *minio.Client
	BucketName string
}

// NewMinioClient ist dein "Konstruktor" (analog zur @Bean Methode)
func NewMinioClient(cfg config.BucketConfig) *MinioClient {
	// Initialisierung des MinIO Clients
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""), // TODO: token noch nötig?
		Secure: false,                                                     // In Docker meist false (kein HTTPS lokal)
	})
	if err != nil {
		log.Fatalf("Fehler beim Erstellen des MinIO Clients: %v", err)
	}

	log.Printf("MinIO Client erfolgreich für Endpoint %s erstellt", cfg.Endpoint)

	return &MinioClient{
		Client:     client,
		BucketName: cfg.Name,
	}
}

func (m *MinioClient) Upload(ctx context.Context, objectName string, reader io.Reader, size int64) error {
	const minioMaxSize = 5 * 1024 * 1024 * 1024 // 5GB (MinIO default)
	if size > minioMaxSize {
		return errors.New("file exceeds MinIO size limit")
	}

	_, err := m.Client.PutObject(ctx, m.BucketName, objectName, reader, size, minio.PutObjectOptions{
		ContentType: "application/octet-stream",
	})
	return err
}

func (m *MinioClient) Delete(ctx context.Context, objectName string) error {
	return m.Client.RemoveObject(ctx, m.BucketName, objectName, minio.RemoveObjectOptions{})
}

func (m *MinioClient) GetPresignedURL(ctx context.Context, objectName string) (string, error) {
	// Wie lange soll der Link gültig sein?
	expiry := time.Second * 60 * 15 // 15 Minuten => nice code

	reqParams := make(url.Values)
	reqParams.Set("response-content-type", "audio/mpeg") //wichtig sonst download und kein playback

	presignedURL, err := m.Client.PresignedGetObject(ctx, m.BucketName, objectName, expiry, reqParams)
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned url: %w", err)
	}

	return presignedURL.String(), nil
}

func (m *MinioClient) GenerateTrackKey(userID uuid.UUID, trackID uuid.UUID, ext string) string {
	return fmt.Sprintf("users/%s/tracks/%s%s", userID, trackID, ext)
}

func (m *MinioClient) ValidateAudioFile(file io.Reader, size int64) error {
	if size > (100 << 20) {
		return errors.New("Track size must be smaller than 100MB")
	}

	return nil
}
