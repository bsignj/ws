package main

import (
	"fmt"
	"go.uber.org/zap"
	"net/http"
	"webs/config"
	"webs/events"
	log "webs/pkg/logger"
	"webs/ws"

	"github.com/go-chi/chi/v5"
)

func main() {

	// Configuration
	cfg, err := config.NewConfig()
	if err != nil {
		fmt.Errorf("config error: %+v", err)
		panic(err)
	}

	zapLogger := log.NewLogger(cfg.Log.Level, cfg.Log.OutputPath, cfg.Log.ErrOutputPath)
	defer zapLogger.Sync()

	// Router erstellen
	router := chi.NewRouter()

	// Hub erstellen
	hub := ws.NewHub(&cfg.Hub)
	hub.CreateRooms([]string{
		"chat",
	}, &cfg.Room)

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
	log.ZLogger.Info("Webserver startet auf Port", zap.String("Port", cfg.App.Port))
	err = http.ListenAndServe(":"+cfg.App.Port, router)
	if err != nil {
		log.ZLogger.Fatal("ListenAndServe: ", zap.Error(err))
	}
}
