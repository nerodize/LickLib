package storage

import (
	"LickLib/cmd/internal/config" // Import deiner Config-Struktur
	"context"
	"io"
	"log"

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
	_, err := m.Client.PutObject(ctx, m.BucketName, objectName, reader, size, minio.PutObjectOptions{
		ContentType: "application/octet-stream",
	})
	return err
}

func (m *MinioClient) Delete(ctx context.Context, objectName string) error {
	return m.Client.RemoveObject(ctx, m.BucketName, objectName, minio.RemoveObjectOptions{})
}
