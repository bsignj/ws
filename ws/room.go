package ws

import (
	"github.com/alphadose/haxmap"
	"go.uber.org/zap"
	"webs/config"
	log "webs/pkg/logger"
)

type Room struct {
	name       string
	clients    *haxmap.Map[string, *Client]
	register   chan *Client
	unregister chan *Client
	send       chan []byte
	workers    int
}

func newRoom(name string, cfg *config.Room) *Room {
	room := &Room{
		name:       name,
		clients:    haxmap.New[string, *Client](uintptr(cfg.BufferedClientSize)),
		register:   make(chan *Client, cfg.BufferedRegisterSize),
		unregister: make(chan *Client, cfg.BufferedUnregisterSize),
		send:       make(chan []byte, cfg.BufferedMessageSize),
		workers:    100,
	}

	// Routine starten

	for i := 0; i < room.workers; i++ {
		go room.consumeBroadcastedMessage(i)
	}
	go room.run()

	return room
}

// Startet die Routine zur Verwaltung des Raums
func (room *Room) run() {

	for {
		select {
		// Client hinzufügen
		case client := <-room.register:
			room.clients.Set(client.ID, client)
			client.rooms.Set(room.name, room)
			log.ZLogger.Debug("[MSG-Room] Client %s betritt den Raum '%s'", zap.String("ClientID", client.ID), zap.String("RoomName", room.name))

		// Client entfernen
		case client := <-room.unregister:
			if _, ok := room.clients.Get(client.ID); ok {
				room.clients.Del(client.ID)
				client.rooms.Del(room.name)
				log.ZLogger.Debug("[MSG-Room] Client %s verlässt den Raum '%s'", zap.String("ClientID", client.ID), zap.String("RoomName", room.name))
			}
		}
	}
}

func (room *Room) consumeBroadcastedMessage(workerID int) {
	for {
		select {
		// Nachricht an alle Clients im Raum schicken
		case event := <-room.send:
			room.clients.ForEach(func(_ string, client *Client) bool {
				select {
				case client.send <- event:
				default:
					log.ZLogger.Warn("[MSG-Room] Event-Verlust bei Client %s", zap.Int("WorkerID", workerID), zap.String("ClientID", client.ID))
				}
				return true
			})
		}
	}
}

// Schickt die Nachricht an alle Clients im Raum
func (room *Room) broadcast(event []byte) {
	room.send <- event
}

func (room *Room) subscribe(client *Client) {
	room.register <- client
}

func (room *Room) unsubscribe(client *Client) {
	room.unregister <- client
}
