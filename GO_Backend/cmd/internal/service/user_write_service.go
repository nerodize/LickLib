package service

import (
	models "LickLib/cmd/internal/entity"
	"LickLib/cmd/internal/repository"
	"LickLib/cmd/storage"

	"context"
	"errors"
	"log"

	//besser als "math/rand" vor allem für PWs
	"crypto/rand"
	"math/big"

	"github.com/google/uuid"
)

type UserMetadata struct {
	Username string
	Email    *string
}

type UserWriteService struct {
	userRepo  repository.UserRepository
	trackRepo repository.TrackRepository
	storage   storage.MinioClient
}

// Konstruktor
func NewUserWriteService(ur repository.UserRepository, tr repository.TrackRepository, s storage.MinioClient) *UserWriteService {
	return &UserWriteService{
		userRepo:  ur,
		trackRepo: tr,
		storage:   s,
	}
}

type UpdateUserRequest struct {
	Username *string `json:"username"`
	Email    *string `json:"email"`
}

func (s *UserWriteService) CreateUser(ctx context.Context, data UserMetadata) error {
	// Falls du die Email loggen willst, mach es sicher:
	emailLog := "n/a"
	if data.Email != nil {
		emailLog = *data.Email
	}
	log.Printf("Erstelle User %s mit Email %s", data.Username, emailLog)

	userEntity := &models.User{
		ID:           CreateUserID(),
		Username:     data.Username,
		Email:        data.Email, // Hier wird einfach die Adresse kopiert (sicher!)
		PasswordHash: GeneratePassword(8),
	}

	return s.userRepo.CreateUser(userEntity)
}

// wtf
func GenerateSecurePassword() string {
	// 1. Zufällige Länge zwischen 8 und 16 festlegen
	// crypto/rand braucht ein bisschen mehr Handling als math/rand
	lengthRange := big.NewInt(9) // 0 bis 8
	randomExtra, err := rand.Int(rand.Reader, lengthRange)
	if err != nil {
		return ""
	}
	length := int(randomExtra.Int64()) + 8

	// 2. Erlaubte Zeichen
	charset := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*"

	// 3. Passwort-Slice vorbereiten
	pw := make([]byte, length)
	charLen := big.NewInt(int64(len(charset)))

	for i := 0; i < length; i++ {
		// Sicherer Zufallsindex für das Charset
		idx, err := rand.Int(rand.Reader, charLen)
		if err != nil {
			return ""
		}
		pw[i] = charset[idx.Int64()]
	}

	return string(pw)
}

func GeneratePassword(pwLength int) string {
	charset := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*"
	// sonst scheinbar Konkatenation nötig
	id := make([]byte, pwLength)

	for i := 0; i < pwLength; i++ {
		randomIndice := getRandomIndice(len(charset))
		id[i] = charset[randomIndice]
	}
	return string(id)
}

func getRandomIndice(max int) int {
	bigMax := big.NewInt(int64(max))

	n, err := rand.Int(rand.Reader, bigMax)
	if err != nil {
		return 0
	}

	return int(n.Int64())
}

// helper => kann ausgelagert werden und eigentlich private
func CreateUserID() uuid.UUID {
	// Generiert eine Version 4 UUID (zufallsbasiert)
	newID := uuid.New()
	return newID
}

// muss kaskadierendes Delete sein => checken ob Tracks von dem user und dann Abfahrt
func (s *UserWriteService) DeleteUser(ctx context.Context, userID uuid.UUID) error {
	// muss byID sein...
	tracks, err := s.trackRepo.FindByUserID(userID)

	for _, track := range tracks {
		err := s.storage.Delete(ctx, track.StorageKey)
		if err != nil {
			log.Printf("Warning: couldn't delete S3 file %s: %v ", track.StorageKey, err)
		}
	}

	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return err
	}

	// manuelle Dereferenzierung, Go compiler übernimmt das aber by default also nicht nötig
	if (*user).ID != userID {
		return errors.New("Authorization error")
	}

	return s.userRepo.DeleteUser(userID)
}

func (s *UserWriteService) UpdateUser(ctx context.Context, userID uuid.UUID, req UpdateUserRequest) error {
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return err
	}
	// könnte man wieder mit expliziter Dereferenzierung schreiben
	if user.ID != userID {
		return errors.New("not authorized to perform this action")
	}

	updates := make(map[string]interface{})
	if req.Username != nil {
		updates["username"] = *req.Username
	}
	if req.Email != nil {
		updates["email"] = *req.Email
	}

	return s.userRepo.UpdateUser(userID, updates)
}
