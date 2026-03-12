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

	//besser als "math/rand" vor allem für PWs
	"crypto/rand"
	"math/big"

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

/*
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
*/

// ohne middleware, anders geht es faktisch nicht...
func (s *UserWriteService) CreateUser(ctx context.Context, data UserMetadata) (uuid.UUID, error) {
	// 1. Lokaler Vorab-Check (Schnell)
	exists, err := s.userRepo.ExistsByUsernameOrEmail(data.Username, *data.Email)
	if exists {
		return uuid.Nil, errors.New("user already exists")
	}

	// 2. Keycloak Admin-Token holen
	token, err := s.getAdminToken(s.kcCfg)
	if err != nil {
		return uuid.Nil, err
	}

	// 3. Den User in Keycloak anlegen (Der "Master"-Eintrag)
	// Wir übergeben das Passwort direkt an die Keycloak-API
	kcUserID, err := s.createKeycloakUser(token, data, data.Password)
	if err != nil {
		return uuid.Nil, err
	}

	// 4. Den User in Postgres spiegeln (Der "Local"-Eintrag für deine Tracks etc.)
	userEntity := &models.User{
		ID:       kcUserID, // Hier nutzen wir die ID, die Keycloak gerade generiert hat!
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

func (s *UserWriteService) createKeycloakUser(adminToken string, data UserMetadata, password string) (uuid.UUID, error) {
	// 1. Payload vorbereiten
	kcUser := map[string]interface{}{
		"username": data.Username,
		"email":    data.Email,
		"enabled":  true,
		"credentials": []map[string]interface{}{
			{
				"type":      "password",
				"value":     password,
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
		// Kleiner Tipp: Hier den Body lesen hilft extrem beim Debuggen,
		// falls Keycloak sagt "User already exists" (409)
		return uuid.Nil, fmt.Errorf("keycloak returned status %d", resp.StatusCode)
	}

	// 3. UUID aus Header fischen
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

// wtf; mit keycloak sollte es sowieso redundant sein.
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
	// 1. Dateien aufräumen (S3/Local Storage)
	tracks, _ := s.trackRepo.FindByUserID(userID)
	for _, track := range tracks {
		_ = s.storage.Delete(ctx, track.StorageKey)
		// Tipp: Hier nur loggen, nicht abbrechen.
		// Wenn S3 mal zickt, wollen wir den User trotzdem löschen können.
	}

	// 2. Keycloak Admin-Token holen
	token, err := s.getAdminToken(s.kcCfg)
	if err != nil {
		return fmt.Errorf("auth for keycloak deletion failed: %v", err)
	}

	// 3. User in Keycloak löschen (Der wichtigste Schritt!)
	if err := s.deleteKeycloakUser(token, userID); err != nil {
		return fmt.Errorf("keycloak deletion failed: %v", err)
	}

	// 4. User in Postgres löschen
	// Durch "ON DELETE CASCADE" in der DB würden hier
	// die Track-Einträge automatisch mitgelöscht werden.
	return s.userRepo.DeleteUser(userID)
}

func (s *UserWriteService) UpdateUser(ctx context.Context, userID uuid.UUID, req UpdateUserRequest) error {

	/*
		// könnte man wieder mit expliziter Dereferenzierung schreiben
		if user.ID != userID {
			return errors.New("not authorized to perform this action")
		}
	*/

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
