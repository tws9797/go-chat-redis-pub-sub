package services

import "go-chat-redis-pub-sub/models"

type RoomService interface {
	AddRoom(room *models.RoomInput) (*models.RoomDBResponse, error)
	FindRoomByName(name string) (*models.RoomDBResponse, error)
}
