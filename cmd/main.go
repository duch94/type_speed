package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true }, // Allow all connections
}

type InputMessage struct {
	Text string `json:"userInput"`
}

func handleConnections(w http.ResponseWriter, r *http.Request) {
	// Upgrade initial GET request to a WebSocket
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer ws.Close()

	timestamps := make([]int64, 0)
	measureAfter := 5

	for {
		// Read message from browser
		_, msg, err := ws.ReadMessage()
		if err != nil {
			slog.Error("read error", "err", err)
			break
		}
		timestamps = append(timestamps, time.Now().UnixNano())

		parsedMsg := InputMessage{}
		json.Unmarshal(msg, &parsedMsg)

		if len(timestamps) == measureAfter {
			// Write speed back to browser
			speed := measureSpeed(timestamps)
			slog.Info("Measured", "speed", speed, "timestamps", timestamps)
			timestamps = make([]int64, 0)

			msg := []byte(fmt.Sprintf("<span id=\"speed\" hx-swap-oob=\"true\">%d</span>", speed))
			err = ws.WriteMessage(websocket.TextMessage, msg)
			if err != nil {
				fmt.Println("write error:", err)
				break
			}
		}
	}
}

func measureSpeed(timestampsNs []int64) int64 {
	if timestampsNs == nil || len(timestampsNs) == 0 {
		return 0
	}

	var speedNs int64
	length := len(timestampsNs)
	intervals := make([]int64, 0)
	for i := length - 1; i > 1; i-- {
		interval := timestampsNs[i] - timestampsNs[i-1]
		intervals = append(intervals, interval)
	}

	var accumulator int64
	for _, i := range intervals {
		accumulator += i
	}
	speedNs = accumulator / int64(length)

	speed := 60 * time.Second / time.Duration(speedNs)

	return int64(speed)
}

func main() {
	r := chi.NewRouter()
	r.Use(middleware.Logger)

	dir := http.Dir("../frontend")

	fs := http.FileServer(dir)
	r.Handle("/client/*", http.StripPrefix("/client/", fs))
	r.Handle("/ws", http.HandlerFunc(handleConnections))

	slog.Info("WebSocket server started on http://127.0.0.1:8080")
	slog.Info("Client started on http://127.0.0.1:8080/client/index.html")

	err := http.ListenAndServe(":8080", r)
	if err != nil {
		slog.Error("ListenAndServe:", "err", err)
	}
}
