package services

import (
	"errors"
	"go-chat-redis-pub-sub/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/net/context"
	"strings"
	"time"
)

type RoomServiceImpl struct {
	collection *mongo.Collection
}

func NewRoomService(collection *mongo.Collection) RoomService {
	return &RoomServiceImpl{collection}
}

func (rs *RoomServiceImpl) AddRoom(room *models.RoomInput) (*models.RoomDBResponse, error) {
	ctx := context.TODO()

	room.CreatedAt = time.Now()
	room.UpdatedAt = room.CreatedAt

	res, err := rs.collection.InsertOne(ctx, &room)

	if err != nil {
		if er, ok := err.(mongo.WriteException); ok && er.WriteErrors[0].Code == 11000 {
			return nil, errors.New("roomname already exist")
		}
		return nil, err
	}

	var newRoom *models.RoomDBResponse
	query := bson.M{"_id": res.InsertedID}

	err = rs.collection.FindOne(ctx, query).Decode(&newRoom)
	if err != nil {
		return nil, err
	}

	return newRoom, nil
}

func (rs *RoomServiceImpl) FindRoomByName(roomName string) (*models.RoomDBResponse, error) {

	ctx := context.TODO()
	var room *models.RoomDBResponse

	query := bson.M{"name": strings.ToLower(roomName)}
	err := rs.collection.FindOne(ctx, query).Decode(&room)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			return &models.RoomDBResponse{}, err
		}

		return nil, err
	}

	return room, nil
}
