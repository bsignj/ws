package ws

import (
	"encoding/json"
	"log"
	"strings"
)

type EventHandler func(*Event)

type Event struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

// NewEventFromRaw erstellt ein Event aus Bytes
func NewEventFromRaw(message []byte) (*Event, error) {
	event := new(Event)
	return event, json.Unmarshal(message, event)
}

// Raw konvertiert das Event in Bytes
func (event *Event) Raw() ([]byte, error) {
	return json.Marshal(event)
}

// UnmarshalPayload unmarshalled die Payload in das gegebene Ziel
func (event *Event) UnmarshalPayload(target interface{}) error {
	return json.Unmarshal(event.Payload, target)
}

// CreateAndSendEvent sendet ein Event an den angegebenen Raum
func CreateAndSendEvent(hub *Hub, eventType string, payload interface{}) {
	data, err := json.Marshal(payload)
	if err != nil {
		log.Printf("[ERR-Event] Marshal %s -> %v", eventType, err)
		return
	}

	outgoingEvent := &Event{
		Type:    eventType,
		Payload: data,
	}

	room := strings.Split(eventType, ":")[0]
	hub.BroadcastToRoom(room, outgoingEvent)
}
