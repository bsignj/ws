package ws

import (
	"encoding/json"
	"go.uber.org/zap"
	"strings"
	log "webs/pkg/logger"
)

// EventHandler function type
type EventHandler func(*Event)

// Event struct for handling WebSocket messages
type Event struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

// NewEventFromRaw creates an Event from a byte array
func NewEventFromRaw(message []byte) (*Event, error) {
	var rawMessage []interface{}
	err := json.Unmarshal(message, &rawMessage)
	if err != nil {
		log.ZLogger.Error("[ERR-Event] Unmarshal %s -> %v", zap.Error(err))
		return nil, err
	}

	event := new(Event)

	// Ensure the array has two elements: the type and payload
	if len(rawMessage) != 2 {
		log.ZLogger.Error("[ERR-Event] Invalid message format")
		return nil, err
	}

	// First element should be the type (string)
	eventType, ok := rawMessage[0].(string)
	if !ok {
		log.ZLogger.Error("[ERR-Event] Invalid event type format")
		return nil, err
	}
	event.Type = eventType

	// Second element is the payload (as raw JSON)
	payload, err := json.Marshal(rawMessage[1])
	if err != nil {
		log.ZLogger.Error("[ERR-Event] Marshal payload %s -> %v", zap.Error(err))
		return nil, err
	}
	event.Payload = payload

	return event, nil
}

// UnmarshalPayload unmarshals the Payload into the given target
func (event *Event) UnmarshalPayload(target interface{}) error {
	return json.Unmarshal(event.Payload, target)
}

// CreateAndSendEvent sends an event to the specified room
func CreateAndSendEvent(hub *Hub, eventType string, payload interface{}) {
	// Marshal the payload into JSON
	data, err := json.Marshal(payload)
	if err != nil {
		log.ZLogger.Error("[ERR-Event] Marshal %s -> %v", zap.Error(err))
		return
	}

	// Create the outgoing event
	outgoingEvent := []interface{}{
		eventType,
		json.RawMessage(data),
	}

	// Marshal the outgoing event as an array format
	message, err := json.Marshal(outgoingEvent)
	if err != nil {
		log.ZLogger.Error("[ERR-Event] Marshal outgoing event %s -> %v", zap.Error(err))
		return
	}

	room := strings.Split(eventType, ":")[0]

	hub.BroadcastToRoom(room, message)
}
