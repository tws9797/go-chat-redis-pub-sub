package chat

import (
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"go-chat-redis-pub-sub/models"
	"golang.org/x/net/context"
	"log"
	"time"
)

const (
	// Max wait time when writing message to peer
	writeWait = 10 * time.Second

	// Max time till next pong from peer
	pongWait = 60 * time.Second

	// Send ping interval, must be less than pong wait time
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 10000
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
}

var (
	newline = []byte{'\n'}
	//space   = []byte{' '}
)

// Client is a middleman between the websocket connection and a single instance of the Hub type
// To hold the connection
type Client struct {
	ID string `json:"id"`

	Username string `json:"name"`

	hub *Hub

	// The websocket connection
	conn *websocket.Conn

	// Buffered channel of outbound message (message routed from a client to the end user)
	send chan []byte

	rooms map[*Room]bool
}

// newClient define Client struct
func newClient(conn *websocket.Conn, hub *Hub, name string, ID string) *Client {

	return &Client{
		ID:       ID,
		Username: name,
		conn:     conn,
		hub:      hub,
		send:     make(chan []byte, 256),
		rooms:    make(map[*Room]bool),
	}
}

// ServeWS create Websocket connection, handles websocket requests from client/peer requests
func ServeWS(hub *Hub, c *gin.Context) {

	user := c.MustGet("currentUser").(*models.DBUser)

	// Upgrade the HTTP server connection to the websocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Println(err)
		return
	}

	// Create a new client instance for every websocket connection
	client := newClient(conn, hub, user.GetUsername(), user.GetID())

	// Allow collection of memory referenced by the caller by doing all work in new goroutines
	go client.writePump()
	go client.readPump()

	// Register the client in the hub Why move behind pump?
	hub.register <- client
}

// writePump handles sending the messages from the hub to the websocket connection
// A goroutine running writePump is started for each connection. The
// application ensures that there is at most one writer to a connection by
// executing all writes from this goroutine.
func (c *Client) writePump() {
	fmt.Println("writePump...")
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			fmt.Println("writing....")
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				//wsSocket close the channel
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued chat messages to the current websocket message.
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write(newline)
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			fmt.Println("ticker...")
			// Setting a new write deadline
			// Write to a connection returns an error when the write operation does not complete by the last set deadline
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				fmt.Println("empty")
				return
			}
		}
	}
}

// readPump handles reading the messages from the websocket connection to the hub
// The application runs readPump in a per-connection goroutine. The application
// ensures that there is at most one reader on a connection by executing all
// reads from this goroutine.
func (c *Client) readPump() {
	fmt.Println("readPump...")
	defer func() {
		c.disconnect()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		fmt.Println("reading....")
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}
		//message = bytes.TrimSpace(bytes.Replace(message, newline, space, -1))
		//c.hub.broadcast <- message

		c.handleNewMessage(message)
	}
}

func (c *Client) handleNewMessage(jsonMessage []byte) {
	fmt.Println("handleNewMessage...")
	var message Message
	if err := json.Unmarshal(jsonMessage, &message); err != nil {
		log.Printf("Error on unmarshal JSON message %s", err)
	}

	message.Sender = c
	fmt.Println(message.Action)
	switch message.Action {
	case SendMessageAction:
		roomID := message.Target.GetId()

		if room := c.hub.findRoomByID(roomID); room != nil {
			room.broadcast <- &message
		}
	case JoinRoomAction:
		c.handleJoinRoomMessage(message)
	case LeaveRoomAction:
		c.handleLeaveRoomMessage(message)
	case JoinRoomPrivateAction:
		c.handleJoinRoomPrivateMessage(message)
	}
}

func (c *Client) handleJoinRoomMessage(message Message) {

	roomName := message.Message

	c.joinRoom(roomName, nil)
}

func (c *Client) handleLeaveRoomMessage(message Message) {
	room := c.hub.findRoomByID(message.Message)
	if room == nil {
		return
	}
	if _, ok := c.rooms[room]; ok {
		delete(c.rooms, room)
	}

	room.unregister <- c
}

func (c *Client) handleJoinRoomPrivateMessage(message Message) {
	target := c.hub.findClientByID(message.Message)
	if target == nil {
		return
	}

	roomName := message.Message + c.GetID()

	joinedRoom := c.joinRoom(roomName, target)

	if joinedRoom != nil {
		c.inviteTargetUser(target, joinedRoom)
	}
}

func (c *Client) joinRoom(roomName string, sender models.User) *Room {
	room := c.hub.findRoomByName(roomName)

	if room == nil {
		room = c.hub.createRoom(roomName, sender != nil)
	}

	// Don't allow to join private rooms through public room message
	if sender == nil && room.Private {
		return nil
	}

	if !c.isInRoom(room) {
		c.rooms[room] = true
		room.register <- c
		c.notifyRoomJoined(room, sender)
	}

	return room
}

func (c *Client) inviteTargetUser(target models.User, room *Room) {
	ctx := context.TODO()

	inviteMessage := &Message{
		Action:  JoinRoomPrivateAction,
		Message: target.GetID(),
		Target:  room,
		Sender:  c,
	}

	if err := redisClient.Publish(ctx, PubSubGeneralChannel, inviteMessage).Err(); err != nil {
		log.Println(err)
	}
}
func (c *Client) isInRoom(room *Room) bool {
	if _, ok := c.rooms[room]; ok {
		return true
	}
	return false
}

func (c *Client) notifyRoomJoined(room *Room, sender models.User) {
	message := Message{
		Action: RoomJoinedAction,
		Target: room,
		Sender: sender,
	}

	c.send <- message.encode()
}

func (c *Client) disconnect() {
	c.hub.unregister <- c
	for room := range c.rooms {
		room.unregister <- c
	}
	c.conn.Close()
}

func (c *Client) GetUsername() string {
	return c.Username
}

func (c *Client) GetID() string {
	return c.ID
}
