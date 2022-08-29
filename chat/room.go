package chat

import (
	"fmt"
	"golang.org/x/net/context"
	"log"
)

const welcomeMessage = "%s joined the room"

type Room struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Private bool   `json:"private"`

	//Registered clients
	clients map[*Client]bool

	// Register requests from the clients
	register chan *Client

	// Unregister requests from the clients
	unregister chan *Client

	// Inbound messages from the clients.
	broadcast chan *Message
}

// NewRoom creates a new Room type
func NewRoom(id string, name string, private bool) *Room {
	return &Room{
		ID:         id,
		Name:       name,
		Private:    private,
		clients:    make(map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan *Message),
	}
}

// RunRoom runs our room, accepting various requests
func (room *Room) RunRoom() {
	go room.subscribeToRoomMessages()

	for {
		select {
		case client := <-room.register:
			room.registerClientInRoom(client)
		case client := <-room.unregister:
			room.unregisterClientInRoom(client)
		case message := <-room.broadcast:
			room.publishRoomMessage(message.encode())
		}
	}
}

// Register client pointer in the room
func (room *Room) registerClientInRoom(client *Client) {
	if !room.Private {
		room.notifyClientJoined(client)
	}
	room.clients[client] = true
}

// Deleting client pointer from the clients map
func (room *Room) unregisterClientInRoom(client *Client) {
	if _, ok := room.clients[client]; ok {
		delete(room.clients, client)
	}
}

// Send read messages to registered clients in the room
func (room *Room) broadcastToClientsInRoom(message []byte) {
	for client := range room.clients {
		client.send <- message
	}
}

func (room *Room) publishRoomMessage(message []byte) {
	ctx := context.TODO()

	fmt.Println("publishRoomMessage")

	err := redisClient.Publish(ctx, room.Name, message).Err()

	if err != nil {
		log.Println(err)
	}
}

func (room *Room) subscribeToRoomMessages() {
	ctx := context.TODO()

	pubsub := redisClient.Subscribe(ctx, room.Name)

	ch := pubsub.Channel()

	for msg := range ch {
		room.broadcastToClientsInRoom([]byte(msg.Payload))
	}
}

func (room *Room) notifyClientJoined(client *Client) {
	message := &Message{
		Action:  SendMessageAction,
		Target:  room,
		Message: fmt.Sprintf(welcomeMessage, client.GetUsername()),
	}

	room.publishRoomMessage(message.encode())
}

// GetId get the room ID
func (room *Room) GetId() string {
	return room.ID
}

// GetName get the room name
func (room *Room) GetName() string {
	return room.Name
}

func (room *Room) GetPrivate() bool {
	return room.Private
}
