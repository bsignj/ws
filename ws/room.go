package ws

import "log"

type Room struct {
	name       string
	clients    map[*Client]bool
	register   chan *Client
	unregister chan *Client
	send       chan *Event
}

func newRoom(name string) *Room {
	room := &Room{
		name:       name,
		clients:    make(map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		send:       make(chan *Event),
	}

	// Routine starten
	go room.run()

	return room
}

// Startet die Routine zur Verwaltung des Raums
func (room *Room) run() {
	for {
		select {
		// Client hinzufügen
		case client := <-room.register:
			room.clients[client] = true
			client.rooms[room.name] = room
			log.Printf("[MSG-Room] Client %s betritt den Raum '%s'", client.ID, room.name)

		// Client entfernen
		case client := <-room.unregister:
			if _, ok := room.clients[client]; ok {
				delete(room.clients, client)
				delete(client.rooms, room.name)
				log.Printf("[MSG-Room] Client %s verlässt den Raum '%s'", client.ID, room.name)
			}

		// Nachricht an alle Clients im Raum schicken
		case event := <-room.send:
			for client := range room.clients {
				select {
				case client.send <- event:
				default:
					log.Printf("[WARN] Event-Verlust bei Client %s", client.ID)
				}
			}
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
