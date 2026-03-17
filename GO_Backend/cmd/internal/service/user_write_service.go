package service

import (
	models "LickLib/cmd/internal/entity"
	"LickLib/cmd/internal/repository"
	"LickLib/cmd/storage"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"context"
	"errors"

	//"log"

	"LickLib/cmd/internal/config"

	"github.com/google/uuid"
)

// Die "dtos" unter Umständen noch auslagern
type UserMetadata struct {
	//ID       uuid.UUID `json:"user_id"`
	Username string  `json:"username"`
	Email    *string `json:"email"`
	Password string  `json:"password"` // Hier gehört es hin!
}

type UserWriteService struct {
	userRepo  repository.UserRepository
	trackRepo repository.TrackRepository
	storage   storage.MinioClient
	kcCfg     *config.KeycloakConfig
}

// Konstruktor
func NewUserWriteService(ur repository.UserRepository, tr repository.TrackRepository, s storage.MinioClient, cfg *config.KeycloakConfig) *UserWriteService {
	return &UserWriteService{
		userRepo:  ur,
		trackRepo: tr,
		storage:   s,
		kcCfg:     cfg,
	}
}

// sollte das hier stehen? eher nicht
var ErrUserAlreadyExists = errors.New("user already exists")

type UpdateUserRequest struct {
	Username *string `json:"username"`
	Email    *string `json:"email"`
}

func (s *UserWriteService) getAdminToken(cfg *config.KeycloakConfig) (string, error) {
	data := url.Values{}
	data.Set("grant_type", "client_credentials")
	data.Set("client_id", cfg.ClientID)
	data.Set("client_secret", cfg.ClientSecret)

	resp, err := http.PostForm(cfg.TokenUrl(), data)
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	var res map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&res)
	return res["access_token"].(string), nil

}

// ohne middleware, anders geht es faktisch nicht...
func (s *UserWriteService) CreateUser(ctx context.Context, data UserMetadata) (uuid.UUID, error) {
	if err := s.ValidateMetadata(data); err != nil {
		return uuid.Nil, err
	}

	exists, err := s.userRepo.ExistsByUsernameOrEmail(data.Username, *data.Email)
	if exists {
		return uuid.Nil, errors.New("user already exists")
	}

	// 2. Keycloak Admin-Token holen
	token, err := s.getAdminToken(s.kcCfg)
	if err != nil {
		return uuid.Nil, err
	}

	// Passwort direkt an die Keycloak-API übergeben
	kcUserID, err := s.createKeycloakUser(token, data)
	if err != nil {
		return uuid.Nil, err
	}

	// 4. Den User in Postgres spiegeln (Der "Local"-Eintrag für deine Tracks etc.)
	userEntity := &models.User{
		ID:       kcUserID, // Hier die ID nutzen die KC generiert hat!
		Username: data.Username,
		Email:    data.Email,
	}

	if err := s.userRepo.CreateUser(userEntity); err != nil {
		// ROLLBACK-Logik: Falls DB-Fehler, User in Keycloak wieder löschen!
		s.deleteKeycloakUser(token, kcUserID)
		return uuid.Nil, fmt.Errorf("local sync failed, rolled back identity: %v", err)
	}

	return kcUserID, err
}

func (s *UserWriteService) createKeycloakUser(adminToken string, data UserMetadata) (uuid.UUID, error) {
	// 1. Payload vorbereiten
	kcUser := map[string]interface{}{
		"username": data.Username,
		"email":    data.Email,
		"enabled":  true,
		"credentials": []map[string]interface{}{
			{
				"type":      "password",
				"value":     data.Password,
				"temporary": false,
			},
		},
	}

	jsonData, _ := json.Marshal(kcUser)

	// 2. Request absetzen (nutzt jetzt s.kcCfg)
	req, _ := http.NewRequest("POST", s.kcCfg.AdminUsersUrl(), bytes.NewBuffer(jsonData))
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return uuid.Nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		// falls Keycloak sagt "User already exists" (409)
		return uuid.Nil, fmt.Errorf("keycloak returned status %d", resp.StatusCode)
	}

	// UUID aus Header fischen
	location := resp.Header.Get("Location")
	segments := strings.Split(location, "/")
	kcUUIDStr := segments[len(segments)-1]

	return uuid.Parse(kcUUIDStr)
}
func (s *UserWriteService) deleteKeycloakUser(adminToken string, userID uuid.UUID) error {
	// Die URL für einen spezifischen User ist /admin/realms/{realm}/users/{id}
	url := fmt.Sprintf("%s/%s", s.kcCfg.AdminUsersUrl(), userID.String())

	req, _ := http.NewRequest("DELETE", url, nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// 204 No Content bedeutet Erfolg
	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusNotFound {
		return fmt.Errorf("keycloak delete failed: status %d", resp.StatusCode)
	}

	return nil
}

// Hilfsfunktion für das Keycloak-Update
func (s *UserWriteService) updateKeycloakUser(token string, userID uuid.UUID, payload map[string]interface{}) error {
	url := fmt.Sprintf("%s/%s", s.kcCfg.AdminUsersUrl(), userID.String())
	jsonData, _ := json.Marshal(payload)

	req, _ := http.NewRequest("PUT", url, bytes.NewBuffer(jsonData)) // PUT für Updates
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := (&http.Client{}).Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("keycloak update status: %d", resp.StatusCode)
	}
	return nil
}

// muss kaskadierendes Delete sein => checken ob Tracks von dem user und dann Abfahrt
func (s *UserWriteService) DeleteUser(ctx context.Context, userID uuid.UUID) error {
	// 1. Dateien aufräumen (S3/Local Storage)
	tracks, _ := s.trackRepo.FindByUserID(userID)
	for _, track := range tracks {
		_ = s.storage.Delete(ctx, track.StorageKey)
	}
	token, err := s.getAdminToken(s.kcCfg)
	if err != nil {
		return fmt.Errorf("auth for keycloak deletion failed: %v", err)
	}

	if err := s.deleteKeycloakUser(token, userID); err != nil {
		return fmt.Errorf("keycloak deletion failed: %v", err)
	}

	return s.userRepo.DeleteUser(userID)
}

func (s *UserWriteService) UpdateUser(ctx context.Context, userID uuid.UUID, req UpdateUserRequest) error {
	updates := make(map[string]interface{})
	if req.Username != nil {
		updates["username"] = *req.Username
	}
	if req.Email != nil {
		updates["email"] = *req.Email
	}

	if len(updates) == 0 {
		return nil
	}

	token, err := s.getAdminToken(s.kcCfg)
	if err != nil {
		return err
	}

	if err := s.updateKeycloakUser(token, userID, updates); err != nil {
		return fmt.Errorf("keycloak update failed: %v", err)
	}

	return s.userRepo.UpdateUser(userID, updates)
}

func (s *UserWriteService) ValidateMetadata(data UserMetadata) error {
	if len(strings.TrimSpace(data.Username)) < 3 { // removes trailing and leading whitespaces
		return errors.New("Username is too short. Must be at least 3 letters.")
	}

	if data.Email == nil || !strings.Contains(*data.Email, "@") {
		return errors.New("invalid email adress")
	}

	if len(data.Password) < 8 {
		return errors.New("Passwords must be at least 8 letters long")
	}

	return nil

}
