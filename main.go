package main

import (
	"log"
	"net/http"
	"webs/events"
	"webs/ws"

	"github.com/go-chi/chi/v5"
)

func main() {
	// Router erstellen
	router := chi.NewRouter()

	// Hub erstellen
	hub := ws.NewHub()
	hub.CreateRooms([]string{
		"chat",
	})

	// Statische Dateien einbinden
	router.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "static/chat.html")
	})

	// WebSocket
	router.Get("/ws", func(w http.ResponseWriter, r *http.Request) {
		// Client registrieren
		client, err := hub.OnConnect(w, r)
		if err != nil {
			panic(err)
		}

		// Event-Handler registrieren
		eventHandlers := map[string]func(*ws.Hub, *ws.Client, *ws.Event){
			"subscribe:chat":     events.SubscribeHandler,
			"subscribe:roulette": events.SubscribeHandler,

			"unsubscribe:chat":     events.UnsubscribeHandler,
			"unsubscribe:roulette": events.UnsubscribeHandler,

			"chat:message": events.ChatMessageHandler,
		}

		// Event-Handler ausführen
		for event, handler := range eventHandlers {
			hub.On(event, func(event *ws.Event) {
				handler(hub, client, event)
			})
		}
	})

	// Webserver starten
	log.Println("Webserver startet auf Port: 8383")
	err := http.ListenAndServe(":8383", router)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
