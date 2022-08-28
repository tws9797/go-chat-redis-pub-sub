package models

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
	"time"
)

// User to be implemented by the client
type User interface {
	GetID() string
	GetUsername() string
}

type DBUser struct {
	ID        primitive.ObjectID `json:"id" bson:"_id"`
	Username  string             `json:"username" bson:"username"`
	Password  string             `json:"password" bson:"password"`
	CreatedAt time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt time.Time          `json:"updated_at" bson:"updated_at"`
}

type SignInInput struct {
	Username string `json:"username" bson:"username" binding:"required"`
	Password string `json:"password" bson:"password" binding:"required"`
}

type SignUpInput struct {
	Username  string    `json:"username" bson:"username" binding:"required"`
	Password  string    `json:"password" bson:"password" binding:"required"`
	CreatedAt time.Time `json:"created_at" bson:"created_at"`
	UpdatedAt time.Time `json:"updated_at" bson:"updated_at"`
}

type CreateUserRequest struct {
	Username  string    `json:"username" bson:"username" binding:"required"`
	CreatedAt time.Time `json:"created_at" bson:"created_at"`
	UpdatedAt time.Time `json:"updated_at" bson:"updated_at"`
}

func (user *DBUser) GetID() string {
	return user.ID.String()
}

func (user *DBUser) GetUsername() string {
	return user.Username
}
