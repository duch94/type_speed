package speed

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// The number of characters to measure the speed after.
	measureAfter = 5
	// The URL of the frontend.
	frontendURL = "http://127.0.0.1:8080"
	// The number of seconds in a minute.
	secondsInAMinute = 60
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		return origin == frontendURL
	},
}

type InputMessage struct {
	Text string `json:"userInput"`
}

// HandleConnections handles the WebSocket connections.
func HandleConnections(w http.ResponseWriter, r *http.Request) {
	// Upgrade initial GET request to a WebSocket
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("upgrade error", "err", err)
		return
	}
	defer ws.Close()

	timestamps := make([]int64, 0)

	// TODO: send message with statistics after text have been written.
	textToMeasure := "The quick brown fox jumps over the lazy dog."
	textMsg := []byte(fmt.Sprintf("<div id=\"textToType\" hx-swap-oob=\"true\">%s</div>", textToMeasure))
	err = ws.WriteMessage(websocket.TextMessage, textMsg)
	if err != nil {
		slog.Error("textToType write error", "err", err)
	}
	sessionSucceeded := false
	maxErrors := len(textToMeasure)
	errorsCount := 0
	for {
		// Read message from browser
		_, inputMsg, err := ws.ReadMessage()
		if err != nil {
			slog.Error("read error", "err", err)
			return
		}

		timestamps = append(timestamps, time.Now().UnixNano())
		var parsedMsg InputMessage
		err = json.Unmarshal(inputMsg, &parsedMsg)
		if err != nil {
			slog.Error("unmarshal error", "err", err)
			continue
		}

		if !strings.HasPrefix(textToMeasure, parsedMsg.Text) || len(parsedMsg.Text) == 0 {
			errorsCount++
			if errorsCount >= maxErrors {
				err = finalizeSession(ws, sessionSucceeded)
				if err != nil {
					slog.Error("ending session error", "err", err, "sessionSucceeded", sessionSucceeded)
					return
				}
			}
			continue
		}

		// Write speed back to browser
		speed := measureSpeed(timestamps)
		slog.Info("Measured", "speed", speed, "timestamps", timestamps, "msg", parsedMsg.Text)
		// TODO: Send errors count as well
		outputMsg := []byte(fmt.Sprintf("<span id=\"speed\" hx-swap-oob=\"true\">%d</span>", speed))
		err = ws.WriteMessage(websocket.TextMessage, outputMsg)
		if err != nil {
			slog.Error("speed write error", "err", err)
			return
		}

		if textToMeasure == parsedMsg.Text {
			break
		}
		sessionSucceeded = true
	}
	err = finalizeSession(ws, sessionSucceeded)
	if err != nil {
		slog.Error("ending session error", "err", err, "sessionSucceeded", sessionSucceeded)
		return
	}
}

func finalizeSession(ws *websocket.Conn, isSuccess bool) error {
	successMsg := "You did it, congrats!"
	failMsg := "You did too much errors, try again!"
	template := "<span id=\"congrats\" hx-swap-oob=\"true\">%s</span>"

	outputMsg := []byte(fmt.Sprintf(template, successMsg))
	if !isSuccess {
		outputMsg = []byte(fmt.Sprintf(template, failMsg))
	}
	err := ws.WriteMessage(websocket.TextMessage, outputMsg)
	if err != nil {
		slog.Error("final msg write error", "err", err)
		return err
	}

	outputMsg = []byte(fmt.Sprintf("<div id=\"textInput\" hx-swap-oob=\"true\"></div>"))
	err = ws.WriteMessage(websocket.TextMessage, outputMsg)
	if err != nil {
		slog.Error("final msg write error", "err", err)
		return err
	}

	return nil
}

// measureSpeed calculates the typing speed in characters per minute.
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
	duration := time.Duration(speedNs)
	if duration == 0 {
		return 0
	}

	speed := 60 * time.Second / duration
	return int64(speed)
}
