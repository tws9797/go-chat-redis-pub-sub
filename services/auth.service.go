package services

import (
	"go-chat-redis-pub-sub/models"
)

type AuthService interface {
	SignUpUser(*models.SignUpInput) (*models.DBUser, error)
}
