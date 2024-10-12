package ws

import (
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
	clients    map[*Client]bool
	rooms      map[string]*Room
	events     map[string]EventHandler
	register   chan *Client
	unregister chan *Client
	send       chan *Event
}

func NewHub() *Hub {
	hub := &Hub{
		clients:    make(map[*Client]bool),
		rooms:      make(map[string]*Room),
		events:     make(map[string]EventHandler),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		send:       make(chan *Event),
	}

	// Routine starten
	go hub.run()

	return hub
}

func (hub *Hub) run() {
	for {
		select {
		// Client Register
		case client := <-hub.register:
			// Client hinzufügen
			hub.clients[client] = true

			// Rate-Limiter hinzufügen
			log.Printf("[MSG-HUB] Client registriert: %s", client.ID)

		// Client Unregister
		case client := <-hub.unregister:
			if _, ok := hub.clients[client]; ok {
				// Client entfernen
				for _, room := range client.rooms {
					room.unregister <- client
				}
				delete(hub.clients, client)
				close(client.send)

				// Rate-Limiter entfernen
				log.Printf("[MSG-HUB] Client abgemeldet: %s", client.ID)
			}

		// Nachricht an alle Clients im Hub schicken
		case event := <-hub.send:
			for client := range hub.clients {
				select {
				case client.send <- event:
				default:
					// Client entfernen
					for _, room := range client.rooms {
						room.unregister <- client
					}
					delete(hub.clients, client)
					close(client.send)

					// Rate-Limiter entfernen
					log.Printf("[WARN] Event-Verlust bei Client %s", client.ID)
				}
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
	hub.register <- client

	// Go-Routinen für den Client zum Lesen und Schreiben von Nachrichten
	go client.readMessage()
	go client.writeMessage()

	return client, nil
}

// On registriert einen Event-Handler für ein bestimmtes Ereignis
func (hub *Hub) On(event string, eventHandler EventHandler) *Hub {
	hub.events[event] = eventHandler
	return hub
}

// CreateRooms erstellt mehrere Räume
func (hub *Hub) CreateRooms(roomNames []string) {
	for _, name := range roomNames {
		room := newRoom(name)
		hub.rooms[name] = room
	}
}

// Broadcast sendet ein Ereignis an alle Clients im Hub
func (hub *Hub) Broadcast(event *Event) {
	hub.send <- event
}

// BroadcastToRoom sendet ein Ereignis an alle Clients in einem bestimmten Raum
func (hub *Hub) BroadcastToRoom(roomName string, event *Event) {
	if room, ok := hub.rooms[roomName]; ok {
		room.broadcast(event)
	}
}

func (hub *Hub) SubscribeToRoom(roomName string, client *Client) {
	if room, ok := hub.rooms[roomName]; ok {
		room.subscribe(client)
	}
}

func (hub *Hub) UnsubscribeFromRoom(roomName string, client *Client) {
	if room, ok := hub.rooms[roomName]; ok {
		room.unsubscribe(client)
	}
}
