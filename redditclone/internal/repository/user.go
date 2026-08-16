package repository

import (
	"context"
	"errors"
	"math/rand"
	"strconv"
	"sync"

	"github.com/Mimist-Illusionard/obslab/internal/models"
)

var (
	ErrNoUser  = errors.New("user not found")
	ErrHasUser = errors.New("already exists")
	ErrBadPass = errors.New("invalid password")
)

type UserRepository interface {
	Login(login, pass string) (*models.User, error)
	Register(login, pass string) (*models.User, error)
}

type UserMemoryRepository struct {
	users []models.User
	mu    sync.Mutex
}

func (ur *UserMemoryRepository) Register(login, pass string) (*models.User, error) {
	_, span := tracer.Start(context.Background(), "UserMemoryRepository.Register")
	defer span.End()

	ur.mu.Lock()
	defer ur.mu.Unlock()

	for _, user := range ur.users {
		if user.Name == login {
			return nil, ErrHasUser
		}
	}

	u := models.User{
		ID:   strconv.FormatInt(rand.Int63(), 10),
		Name: login,
	}

	err := u.ChangePassword(pass)
	if err != nil {
		return nil, err
	}

	ur.users = append(ur.users, u)

	return &u, nil
}

func (ur *UserMemoryRepository) Login(login, pass string) (*models.User, error) {
	_, span := tracer.Start(context.Background(), "UserMemoryRepository.Login")
	defer span.End()

	ur.mu.Lock()
	defer ur.mu.Unlock()

	index := -1
	for i, user := range ur.users {
		if user.Name == login {
			index = i
			break
		}
	}

	if index == -1 {
		return nil, ErrNoUser
	}

	ok := ur.users[index].CheckPassword(pass)
	if !ok {
		return nil, ErrBadPass
	}

	return &ur.users[index], nil
}
