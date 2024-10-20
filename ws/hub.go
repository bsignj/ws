package ws

import (
	"github.com/alphadose/haxmap"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// Rate Limiter
	maxMessages   = 60
	resetInterval = maxMessages * time.Second
)

var websocketUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// Commented for testing purposes
	//CheckOrigin: func(r *http.Request) bool {
	//	origin := r.Header.Get("Origin")
	//	switch origin {
	//	case "http://localhost:8383":
	//		return true
	//	default:
	//		return false
	//	}
	//},
}

type Hub struct {
	clients    *haxmap.Map[string, *Client]
	rooms      *haxmap.Map[string, *Room]
	events     *haxmap.Map[string, EventHandler]
	register   chan *Client
	unregister chan *Client
	send       chan *Event
}

func NewHub() *Hub {
	hub := &Hub{
		clients:    haxmap.New[string, *Client](100_000),
		rooms:      haxmap.New[string, *Room](100),
		events:     haxmap.New[string, EventHandler](500_000),
		register:   make(chan *Client, 100_000),
		unregister: make(chan *Client, 100_000),
		send:       make(chan *Event, 100_000),
	}

	// Routine starten
	go hub.run()

	return hub
}

func (hub *Hub) run() {
	for {
		select {
		// Client Unregister
		case client := <-hub.unregister:
			if _, ok := hub.clients.Get(client.ID); ok {
				// Client entfernen
				client.rooms.ForEach(func(key string, value *Room) bool {
					value.unregister <- client
					return true
				})
				close(client.send)
				client.conn.Close()
				hub.clients.Del(client.ID)

				// Rate-Limiter entfernen
				log.Printf("[MSG-HUB] Client abgemeldet: %s", client.ID)
			}
		}
	}
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

// OnConnect verwaltet die WebSocket-Verbindung und initialisiert den Client
func (hub *Hub) OnConnect(w http.ResponseWriter, r *http.Request) (*Client, error) {
	// Upgrade die reguläre Verbindung zu einer Websocket Verbindung
	conn, err := websocketUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("[ERR-HUB] Akzeptieren der Verbindung:", err)
		return nil, err
	}

	// Client IP-RateLimiter hinzufügen
	// ip := helpers.GetClientIPFromRequest(r)
	// if _, exists := hub.ipRateLimiter.GetLimiter(ip); !exists {
	// 	hub.ipRateLimiter.AddLimiter(ip, 2, 10*time.Second) // Beispiel: max. 5 Nachrichten pro 10 Sekunden
	// }

	// Client erstellen und zum Hub hinzufügen
	client := newClient(hub, conn)
	// Client Register
	hub.clients.Set(client.ID, client)

	// Go-Routinen für den Client zum Lesen und Schreiben von Nachrichten
	go client.readMessage()
	go client.writeMessage()

	return client, nil
}

// On registriert einen Event-Handler für ein bestimmtes Ereignis
func (hub *Hub) On(event string, eventHandler EventHandler) *Hub {
	hub.events.Set(event, eventHandler)
	return hub
}

// CreateRooms erstellt mehrere Räume
func (hub *Hub) CreateRooms(roomNames []string) {
	for _, name := range roomNames {
		room := newRoom(name)
		hub.rooms.Set(name, room)
	}
}

// Broadcast sendet ein Ereignis an alle Clients im Hub
func (hub *Hub) Broadcast(event *Event) {
	hub.send <- event
}

// BroadcastToRoom sendet ein Ereignis an alle Clients in einem bestimmten Raum
func (hub *Hub) BroadcastToRoom(roomName string, event *Event) {
	if room, ok := hub.rooms.Get(roomName); ok {
		room.broadcast(event)
	}
}

func (hub *Hub) SubscribeToRoom(roomName string, client *Client) {
	if room, ok := hub.rooms.Get(roomName); ok {
		room.subscribe(client)
	}
}

func (hub *Hub) UnsubscribeFromRoom(roomName string, client *Client) {
	if room, ok := hub.rooms.Get(roomName); ok {
		room.unsubscribe(client)
	}
}
