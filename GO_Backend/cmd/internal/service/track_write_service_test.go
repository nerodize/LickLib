package service

import (
	models "LickLib/cmd/internal/entity"
	"bytes"
	"context"
	"errors"
	"io"
	"testing"

	"github.com/google/uuid"
)

// ===== MOCK REPOSITORY =====

type MockTrackRepo struct {
	CreateTrackFunc               func(*models.Track) error
	UpdateTrackFunc               func(uuid.UUID, map[string]interface{}) error
	DeleteFailedTracksByTitleFunc func(uuid.UUID, string) error
	FindByIDFunc                  func(uuid.UUID) (*models.Track, error)
}

func (m *MockTrackRepo) CreateTrack(track *models.Track) error {
	if m.CreateTrackFunc != nil {
		return m.CreateTrackFunc(track)
	}
	return nil
}

func (m *MockTrackRepo) UpdateTrack(id uuid.UUID, updates map[string]interface{}) error {
	if m.UpdateTrackFunc != nil {
		return m.UpdateTrackFunc(id, updates)
	}
	return nil
}

func (m *MockTrackRepo) DeleteFailedTracksByTitle(userID uuid.UUID, title string) error {
	if m.DeleteFailedTracksByTitleFunc != nil {
		return m.DeleteFailedTracksByTitleFunc(userID, title)
	}
	return nil
}

func (m *MockTrackRepo) FindByID(id uuid.UUID) (*models.Track, error) {
	if m.FindByIDFunc != nil {
		return m.FindByIDFunc(id)
	}
	return nil, errors.New("not implemented")
}

// ✅ DIESE METHODEN FEHLEN BEI DIR!
func (m *MockTrackRepo) FindByUsername(username string) ([]models.Track, error) {
	return nil, nil
}

func (m *MockTrackRepo) FindByUserID(userID uuid.UUID) ([]models.Track, error) {
	return nil, nil
}

func (m *MockTrackRepo) DeleteTrack(id uuid.UUID) error {
	return nil
}

// ===== MOCK STORAGE =====

type MockMinioClient struct {
	UploadFunc           func(context.Context, string, io.Reader, int64) error
	DeleteFunc           func(context.Context, string) error
	GenerateTrackKeyFunc func(uuid.UUID, uuid.UUID, string) string
	GetPresignedURLFunc  func(context.Context, string) (string, error)
	//ValidateAudioFileFunc func(io.Reader, int64) error // ← NEU

}

func (m *MockMinioClient) Upload(ctx context.Context, key string, r io.Reader, size int64) error {
	if m.UploadFunc != nil {
		return m.UploadFunc(ctx, key, r, size)
	}
	return nil
}

func (m *MockMinioClient) Delete(ctx context.Context, key string) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, key)
	}
	return nil
}

func (m *MockMinioClient) GenerateTrackKey(userID, trackID uuid.UUID, ext string) string {
	if m.GenerateTrackKeyFunc != nil {
		return m.GenerateTrackKeyFunc(userID, trackID, ext)
	}
	return "mock-key"
}

func (m *MockMinioClient) GetPresignedURL(ctx context.Context, key string) (string, error) {
	if m.GetPresignedURLFunc != nil {
		return m.GetPresignedURLFunc(ctx, key)
	}
	return "http://mock-url", nil
}

// ===== TESTS =====

func TestValidateMetadata(t *testing.T) {
	service := &TrackWriteService{}

	tests := []struct {
		name        string
		metadata    TrackMetadata
		expectError bool
	}{
		{
			name: "valid metadata",
			metadata: TrackMetadata{
				Title:       "Valid Title",
				Description: "A good description",
				UserID:      uuid.New(),
				Difficulty:  "EASY",
				FileExt:     ".mp3",
			},
			expectError: false,
		},
		{
			name: "title too short",
			metadata: TrackMetadata{
				Title:       "ab",
				Description: "Good description",
				UserID:      uuid.New(),
				FileExt:     ".mp3",
			},
			expectError: true,
		},
		{
			name: "description too short",
			metadata: TrackMetadata{
				Title:       "Good Title",
				Description: "short",
				UserID:      uuid.New(),
				FileExt:     ".mp3",
			},
			expectError: true,
		},
		{
			name: "invalid difficulty",
			metadata: TrackMetadata{
				Title:       "Good Title",
				Description: "Good description",
				UserID:      uuid.New(),
				Difficulty:  "INVALID",
				FileExt:     ".mp3",
			},
			expectError: true,
		},
		{
			name: "invalid file extension",
			metadata: TrackMetadata{
				Title:       "Good Title",
				Description: "Good description",
				UserID:      uuid.New(),
				FileExt:     ".exe",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.validateMetadata(tt.metadata)

			if tt.expectError && err == nil {
				t.Errorf("expected error, got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("expected no error, got %v", err)
			}
		})
	}
}

func TestValidateAudioFile(t *testing.T) {
	service := &TrackWriteService{}

	tests := []struct {
		name        string
		fileData    []byte
		size        int64
		expectError bool
	}{
		{
			name:        "valid MP3 with ID3 tag",
			fileData:    append([]byte("ID3"), make([]byte, 100)...),
			size:        1024,
			expectError: false,
		},
		{
			name:        "valid WAV file",
			fileData:    append([]byte("RIFF"), append(make([]byte, 4), []byte("WAVE")...)...),
			size:        2048,
			expectError: false,
		},
		{
			name:        "valid FLAC file",
			fileData:    append([]byte("fLaC"), make([]byte, 100)...),
			size:        3072,
			expectError: false,
		},
		{
			name:        "invalid file (plain text)",
			fileData:    []byte("This is plain text, not audio"),
			size:        512,
			expectError: true,
		},
		{
			name:        "file too large",
			fileData:    append([]byte("ID3"), make([]byte, 100)...),
			size:        200 * 1024 * 1024, // 200MB
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file := bytes.NewReader(tt.fileData)
			err := service.validateAudioFile(file, tt.size)

			if tt.expectError && err == nil {
				t.Errorf("expected error, got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("expected no error, got %v", err)
			}
		})
	}
}

func TestUploadTrack_Success(t *testing.T) {
	mockRepo := &MockTrackRepo{
		CreateTrackFunc: func(track *models.Track) error {
			if track.Status != models.TrackStatusUploading {
				t.Errorf("expected status UPLOADING, got %v", track.Status)
			}
			return nil
		},
		UpdateTrackFunc: func(id uuid.UUID, updates map[string]interface{}) error {
			if updates["status"] != models.TrackStatusReady {
				t.Errorf("expected status READY, got %v", updates["status"])
			}
			return nil
		},
	}

	mockStorage := &MockMinioClient{
		UploadFunc: func(ctx context.Context, key string, r io.Reader, size int64) error {
			return nil
		},
		GenerateTrackKeyFunc: func(userID, trackID uuid.UUID, ext string) string {
			return "test-key.mp3"
		},
	}

	service := NewTrackWriteService(mockStorage, mockRepo)

	audioData := append([]byte("ID3"), make([]byte, 100)...)
	file := bytes.NewReader(audioData)

	metadata := TrackMetadata{
		Title:       "Test Track",
		Description: "Test Description",
		UserID:      uuid.New(),
		FileExt:     ".mp3",
	}

	err := service.UploadTrack(context.Background(), file, int64(len(audioData)), metadata)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestUploadTrack_FailedUpload_SetsStatusToFailed(t *testing.T) {
	mockRepo := &MockTrackRepo{
		CreateTrackFunc: func(track *models.Track) error {
			return nil
		},
		UpdateTrackFunc: func(id uuid.UUID, updates map[string]interface{}) error {
			// Verify Rollback
			if updates["status"] != models.TrackStatusFailed {
				t.Errorf("expected status FAILED, got %v", updates["status"])
			}
			return nil
		},
	}

	mockStorage := &MockMinioClient{
		UploadFunc: func(ctx context.Context, key string, r io.Reader, size int64) error {
			return errors.New("minio upload failed")
		},
	}

	service := NewTrackWriteService(mockStorage, mockRepo)

	audioData := append([]byte("ID3"), make([]byte, 100)...)
	file := bytes.NewReader(audioData)

	metadata := TrackMetadata{
		Title:       "Test Track",
		Description: "Test Description",
		UserID:      uuid.New(),
		FileExt:     ".mp3",
	}

	err := service.UploadTrack(context.Background(), file, int64(len(audioData)), metadata)

	if err == nil {
		t.Error("expected error, got nil")
	}
}
