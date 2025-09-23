package main

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/duch94/type_speed/internal/speed"
)

const (
	// The port to listen on.
	port = ":8080"
	// The URL of the frontend.
	frontendURL = "http://127.0.0.1:8080"
)

func main() {
	r := chi.NewRouter()
	r.Use(middleware.Logger)

	dir := http.Dir("../frontend")

	fs := http.FileServer(dir)
	r.Handle("/client/*", http.StripPrefix("/client/", fs))
	r.Handle("/ws", http.HandlerFunc(speed.HandleConnections))

	slog.Info("WebSocket server started on " + frontendURL)
	slog.Info("Client started on " + frontendURL + "/client/index.html")

	err := http.ListenAndServe(port, r)
	if err != nil {
		slog.Error("ListenAndServe:", "err", err)
	}
}
