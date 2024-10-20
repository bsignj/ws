package ws

import (
	"github.com/alphadose/haxmap"
	"log"
	"time"
)

type Room struct {
	name       string
	clients    *haxmap.Map[string, *Client]
	register   chan *Client
	unregister chan *Client
	send       chan *Event
	workers    int
	ticker     *time.Ticker
}

func newRoom(name string) *Room {
	room := &Room{
		name:       name,
		clients:    haxmap.New[string, *Client](100_000),
		register:   make(chan *Client, 100_000),
		unregister: make(chan *Client, 100_000),
		send:       make(chan *Event, 1_000_000),
		workers:    100,
		ticker:     time.NewTicker(100 * time.Millisecond), // Throttle broadcasts every 100ms
	}

	// Routine starten

	for i := 0; i < room.workers; i++ {
		go room.run(i)
	}

	return room
}

// Startet die Routine zur Verwaltung des Raums
func (room *Room) run(workerID int) {

	for {
		select {
		// Client hinzufügen
		case client := <-room.register:
			room.clients.Set(client.ID, client)
			client.rooms.Set(room.name, room)
			log.Printf("WorkerID-%d | [MSG-Room] Client %s betritt den Raum '%s'", workerID, client.ID, room.name)

		// Client entfernen
		case client := <-room.unregister:
			if _, ok := room.clients.Get(client.ID); ok {
				room.clients.Del(client.ID)
				client.rooms.Del(room.name)
				log.Printf("WorkerID-%d | [MSG-Room] Client %s verlässt den Raum '%s'", workerID, client.ID, room.name)
			}

		// Nachricht an alle Clients im Raum schicken
		case event := <-room.send:
			room.clients.ForEach(func(_ string, client *Client) bool {
				select {
				case client.send <- event:
				default:
					log.Printf("WorkerID-%d | [WARN] Event-Verlust bei Client %s", workerID, client.ID)
				}
				return true
			})
		}
	}
}

// Schickt die Nachricht an alle Clients im Raum
func (room *Room) broadcast(event *Event) {
	room.send <- event
}

func (room *Room) subscribe(client *Client) {
	room.register <- client
}

func (room *Room) unsubscribe(client *Client) {
	room.unregister <- client
}
