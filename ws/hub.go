package ws

import (
	"github.com/alphadose/haxmap"
	"go.uber.org/zap"
	"net/http"
	"time"
	"webs/config"
	log "webs/pkg/logger"

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
	config     *config.Config
}

func NewHub(cfg *config.Hub) *Hub {
	hub := &Hub{
		clients:    haxmap.New[string, *Client](uintptr(cfg.BufferedClientSize)),
		rooms:      haxmap.New[string, *Room](uintptr(cfg.BufferedRoomSize)),
		events:     haxmap.New[string, EventHandler](uintptr(cfg.BufferedEventSize)),
		register:   make(chan *Client, cfg.BufferedRegisterSize),
		unregister: make(chan *Client, cfg.BufferedUnregisterSize),
		send:       make(chan *Event, cfg.BufferedMessageSize),
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
				client.rooms.ForEach(func(key string, room *Room) bool {

					// Disconnect from Room
					if _, ok := room.clients.Get(client.ID); ok {
						room.clients.Del(client.ID)
						client.rooms.Del(room.name)
						log.ZLogger.Debug("[MSG-Room] Client %s verlässt den Raum '%s'", zap.String("ClientID", client.ID), zap.String("RoomName", room.name))
					}

					// Disconnect from Hub
					close(client.send)
					client.conn.Close()
					hub.clients.Del(client.ID)
					// Rate-Limiter entfernen
					log.ZLogger.Debug("[MSG-HUB] Client abgemeldet: %s", zap.String("ClientID", client.ID))
					return true
				})
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
		log.ZLogger.Error("[ERR-HUB] Akzeptieren der Verbindung:", zap.Error(err))
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
func (hub *Hub) CreateRooms(roomNames []string, cfg *config.Room) {
	for _, name := range roomNames {
		room := newRoom(name, cfg)
		hub.rooms.Set(name, room)
	}
}

// Broadcast sendet ein Ereignis an alle Clients im Hub
func (hub *Hub) Broadcast(event *Event) {
	hub.send <- event
}

// BroadcastToRoom sendet ein Ereignis an alle Clients in einem bestimmten Raum
func (hub *Hub) BroadcastToRoom(roomName string, event []byte) {
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
