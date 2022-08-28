package services

import (
	"errors"
	"go-chat-redis-pub-sub/models"
	"go-chat-redis-pub-sub/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/net/context"
	"time"
)

type AuthServiceImpl struct {
	collection *mongo.Collection
}

func NewAuthService(collection *mongo.Collection) AuthService {
	return &AuthServiceImpl{collection}
}

func (uc *AuthServiceImpl) SignUpUser(user *models.SignUpInput) (*models.DBUser, error) {

	ctx := context.TODO()

	user.CreatedAt = time.Now()
	user.UpdatedAt = user.CreatedAt

	hashedPassword, _ := utils.HashPassword(user.Password)
	user.Password = hashedPassword

	res, err := uc.collection.InsertOne(ctx, &user)

	if err != nil {
		if er, ok := err.(mongo.WriteException); ok && er.WriteErrors[0].Code == 11000 {
			return nil, errors.New("username already exist")
		}
		return nil, err
	}

	// Create a unique index for the username field
	index := mongo.IndexModel{Keys: bson.M{"username": 1}, Options: options.Index().SetUnique(true)}

	if _, err := uc.collection.Indexes().CreateOne(ctx, index); err != nil {
		return nil, errors.New("could not create index for email")
	}

	var newUser *models.DBUser
	query := bson.M{"_id": res.InsertedID}

	err = uc.collection.FindOne(ctx, query).Decode(&newUser)
	if err != nil {
		return nil, err
	}

	return newUser, nil
}
