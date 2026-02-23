package service

import (
	models "LickLib/cmd/internal/entity"
	"LickLib/cmd/internal/repository"
	"errors"

	"github.com/google/uuid"
)

type UserReadService struct {
	repo repository.UserRepository
}

func NewUserService(r repository.UserRepository) *UserReadService {
	return &UserReadService{repo: r}
}

func (s *UserReadService) GetUserByID(id uuid.UUID) (*models.User, error) {
	var user models.User
	if err := s.repo.PreloadUserByID(id, &user); err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *UserReadService) GetUserByUsername(username string) (*models.User, error) {
	user, err := s.repo.FindByUsername(username)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, errors.New("username not found, no matching users")
	}
	return user, nil
}
