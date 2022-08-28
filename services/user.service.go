package services

import (
	"go-chat-redis-pub-sub/models"
)

type UserService interface {
	RemoveUser(string) error
	GetAllUsers() ([]*models.DBUser, error)
	FindUserById(string) (*models.DBUser, error)
	FindUserByUsername(string) (*models.DBUser, error)
}
