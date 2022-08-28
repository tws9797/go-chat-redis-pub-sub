package chat

import (
	"encoding/json"
	"github.com/go-redis/redis/v8"
	"go-chat-redis-pub-sub/models"
	"go-chat-redis-pub-sub/services"
	"golang.org/x/net/context"
	"log"
)

var redisClient *redis.Client

const PubSubGeneralChannel = "general"

// Hub maintains the set of active clients and broadcasts messages to the client
type Hub struct {
	// Keep tracks of registered clients
	clients map[*Client]bool

	// Register requests from the clients
	register chan *Client

	// Unregister requests from the clients
	unregister chan *Client

	// Inbound messages from the clients.
	broadcast chan []byte

	// Keep tracks of the rooms in the hub
	rooms map[*Room]bool

	users []models.User

	roomService services.RoomService

	userService services.UserService

	authService services.AuthService
}

// NewHub creates a new Hub type
func NewHub(userService services.UserService, roomService services.RoomService, rdsClient *redis.Client) *Hub {

	redisClient = rdsClient

	hub := &Hub{
		clients:     make(map[*Client]bool),
		broadcast:   make(chan []byte),
		register:    make(chan *Client),
		unregister:  make(chan *Client),
		rooms:       make(map[*Room]bool),
		userService: userService,
		roomService: roomService,
	}

	dbUsers, err := userService.GetAllUsers()

	if err != nil {
		panic(err)
	}

	hub.users = make([]models.User, len(dbUsers))

	for i, v := range dbUsers {
		hub.users[i] = v
	}

	return hub
}

// Run Hub server using broadcast, register and unregister channels to listen for different inbound messages
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.registerClient(client)
		case client := <-h.unregister:
			h.unregisterClient(client)
		case message := <-h.broadcast:
			h.broadcastToClients(message)
		}
	}
}

// Register client pointer
func (h *Hub) registerClient(client *Client) {

	if user := h.findUserByID(client.GetID()); user == nil {
		h.authService.SignUpUser(
			&models.SignUpInput{
				Username: user.GetUsername(),
				Password: "TEST",
			})
	}

	h.publishClientJoined(client)
	h.listOnlineClients(client)
	h.clients[client] = true
}

// Deleting client pointer from the clients map
// Close the client's send channel to signal the client that no more messages will be sent to the client
func (h *Hub) unregisterClient(client *Client) {
	if _, ok := h.clients[client]; ok {
		delete(h.clients, client)
		h.publishClientLeft(client)
	}
}

// Send read messages to registered clients
func (h *Hub) broadcastToClients(message []byte) {
	//for client := range h.clients {
	//	select {
	//	case client.send <- message:
	//	default:
	//		close(client.send)
	//		delete(h.clients, client)
	//	}
	//}
	for client := range h.clients {
		client.send <- message
	}
}

// Search the room in the hub by the name entered
func (h *Hub) findRoomByName(name string) *Room {
	var foundRoom *Room
	for room := range h.rooms {
		if room.GetName() == name {
			foundRoom = room
			break
		}
	}

	if foundRoom == nil {
		foundRoom = h.runRoomFromRepository(name)
	}

	return foundRoom
}

func (h *Hub) runRoomFromRepository(name string) *Room {
	var room *Room
	dbRoom, err := h.roomService.FindRoomByName(name)
	if err != nil {
		log.Printf("Room not found: %v\n", err)
	}

	if dbRoom != nil {
		room = NewRoom(dbRoom.Name, dbRoom.Private)
		room.ID = dbRoom.ID

		go room.RunRoom()
		h.rooms[room] = true
	}

	return room
}

func (h *Hub) findRoomByID(ID string) *Room {
	var foundRoom *Room
	for room := range h.rooms {
		if room.GetId() == ID {
			foundRoom = room
			break
		}
	}

	return foundRoom
}

// Create Room
func (h *Hub) createRoom(name string, private bool) *Room {
	room := NewRoom(name, private)

	roomInput := &models.RoomInput{
		Name:    room.Name,
		Private: room.Private,
	}

	_, err := h.roomService.AddRoom(roomInput)
	if err != nil {
		return nil
	}

	go room.RunRoom()
	h.rooms[room] = true

	return room
}

func (h *Hub) notifyClientJoined(client *Client) {
	message := &Message{
		Action: UserJoinedAction,
		Sender: client,
	}

	h.broadcastToClients(message.encode())
}

func (h *Hub) notifyClientLeft(client *Client) {
	message := &Message{
		Action: UserLeftAction,
		Sender: client,
	}

	h.broadcastToClients(message.encode())
}

func (h *Hub) listOnlineClients(client *Client) {

	var uniqueUsers = make(map[string]bool)
	for _, user := range h.users {
		if ok := uniqueUsers[user.GetID()]; !ok {
			message := &Message{
				Action: UserJoinedAction,
				Sender: user,
			}
			uniqueUsers[user.GetID()] = true
			client.send <- message.encode()
		}
	}
}

func (h *Hub) findClientByID(ID string) *Client {
	var foundClient *Client
	for client := range h.clients {
		if client.GetID() == ID {
			foundClient = client
			break
		}
	}

	return foundClient
}

func (h *Hub) publishClientJoined(client *Client) {
	ctx := context.TODO()

	message := &Message{
		Action: UserJoinedAction,
		Sender: client,
	}

	if err := redisClient.Publish(ctx, PubSubGeneralChannel, message).Err(); err != nil {
		log.Println(err)
	}
}

func (h *Hub) publishClientLeft(client *Client) {
	ctx := context.TODO()

	message := &Message{
		Action: UserLeftAction,
		Sender: client,
	}

	if err := redisClient.Publish(ctx, PubSubGeneralChannel, message.encode()).Err(); err != nil {
		log.Println(err)
	}
}

func (h *Hub) listenPubSubChannel() {
	ctx := context.TODO()

	pubsub := redisClient.Subscribe(ctx, PubSubGeneralChannel)
	ch := pubsub.Channel()

	var message Message
	for msg := range ch {
		var message Message
		if err := json.Unmarshal([]byte(msg.Payload), &message); err != nil {
			log.Printf("Error on unmarshal JSON message %s", err)
			return
		}
	}

	switch message.Action {
	case UserJoinedAction:
		h.handleUserJoined(message)
	case UserLeftAction:
		h.handleUserLeft(message)
	case JoinRoomPrivateAction:
		h.handleUserJoinPrivate(message)
	}
}

func (h *Hub) handleUserJoinPrivate(message Message) {
	// Find client for given user, if found add the user to the room.
	targetClients := h.findClientsByID(message.Message)
	for _, targetClient := range targetClients {
		targetClient.joinRoom(message.Target.GetName(), message.Sender)
	}
}

func (h *Hub) findClientsByID(ID string) []*Client {
	var foundClients []*Client
	for client := range h.clients {
		if client.GetID() == ID {
			foundClients = append(foundClients, client)
		}
	}

	return foundClients
}

func (h *Hub) handleUserJoined(message Message) {
	h.users = append(h.users, message.Sender)
	h.broadcastToClients(message.encode())
}

func (h *Hub) handleUserLeft(message Message) {
	for i, user := range h.users {
		if user.GetID() == message.Sender.GetID() {
			h.users[i] = h.users[len(h.users)-1]
			h.users = h.users[:len(h.users)-1]
			break
		}
	}

	h.broadcastToClients(message.encode())
}

// Add the findUserByID method used by client.go
func (h *Hub) findUserByID(ID string) models.User {
	var foundUser models.User
	for _, client := range h.users {
		if client.GetID() == ID {
			foundUser = client
			break
		}
	}

	return foundUser
}
