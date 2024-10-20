package ws

import (
	"bytes"
	"github.com/alphadose/haxmap"
	"go.uber.org/zap"
	"time"
	log "webs/pkg/logger"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const (
	pongWait       = 60 * time.Second
	pingInterval   = (pongWait * 9) / 10
	maxMessageSize = 512
)

var (
	newline = []byte{'\n'}
	space   = []byte{' '}
)

type Client struct {
	ID    string
	hub   *Hub
	conn  *websocket.Conn
	rooms *haxmap.Map[string, *Room]
	send  chan *Event
}

func newClient(hub *Hub, conn *websocket.Conn) *Client {
	return &Client{
		ID:    uuid.New().String(),
		hub:   hub,
		conn:  conn,
		rooms: haxmap.New[string, *Room](100),
		send:  make(chan *Event, 1000),
	}
}

func (client *Client) readMessage() {
	defer func() {
		client.hub.unregister <- client
	}()

	client.conn.SetReadLimit(maxMessageSize)
	client.conn.SetReadDeadline(time.Now().Add(pongWait))
	client.conn.SetPongHandler(func(string) error {
		client.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		// Nachricht lesen
		_, message, err := client.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.ZLogger.Error("[ERR-Client] Nachricht lesen -> %v\n", zap.Error(err))
			}
			return
		}

		// Nachricht trimmen
		message = bytes.TrimSpace(bytes.Replace(message, newline, space, -1))

		// Nachricht in das Event Struct einlesen
		event, err := NewEventFromRaw(message)
		if err != nil {
			log.ZLogger.Error("[ERR-Client] Event erstellen -> %v", zap.Error(err))
			continue
		}
		log.ZLogger.Debug("[MSG-Client] Eingehende Nachricht -> %v", zap.String("Message", string(message)))

		// Event weiterleiten
		if eventHandler, ok := client.hub.events.Get(event.Type); ok {
			eventHandler(event)
		}
	}
}

func (client *Client) writeMessage() {
	// Ping/Pong Ticker erstellen
	ticker := time.NewTicker(pingInterval)

	defer func() {
		ticker.Stop()
		client.hub.unregister <- client
	}()

	for {
		select {
		// Ping an den Client senden
		case <-ticker.C:
			if err := client.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.ZLogger.Error("[ERR-Client] Fehler Ping senden -> %v\n", zap.Error(err))
				return
			}

		// Nachricht an den Client senden
		case event, ok := <-client.send:
			if !ok {
				if err := client.conn.WriteMessage(websocket.CloseMessage, nil); err != nil {
					log.ZLogger.Error("[ERR-Client] Websocket Verbindung geschlossen -> %v\n", zap.Error(err))
				}
				return
			}

			message, err := event.Raw()
			if err != nil {
				log.ZLogger.Error("[ERR-Client] Nachricht erstellen -> %v", zap.Error(err))
				continue
			}
			log.ZLogger.Debug("[MSG-Client] Ausgehende Nachricht -> %v", zap.String("Message", string(message)))

			if err := client.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				log.ZLogger.Error("[ERR-Client] Fehler beim Schreiben der Nachricht -> %v\n", zap.Error(err))
				return
			}
		}
	}
}
