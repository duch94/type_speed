package speed

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// The URL of the frontend.
	frontendURL = "http://127.0.0.1:8080"
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
	textToMeasure := getRandomText()
	textMsg := fmt.Appendf(nil, "<div id=\"textToType\" hx-swap-oob=\"true\">%s</div>", textToMeasure)
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
		outputMsg := fmt.Appendf(nil, "<span id=\"speed\" hx-swap-oob=\"true\">%d</span>", speed)
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

	outputMsg := fmt.Appendf(nil, template, successMsg)
	if !isSuccess {
		outputMsg = fmt.Appendf(nil, template, failMsg)
	}
	err := ws.WriteMessage(websocket.TextMessage, outputMsg)
	if err != nil {
		slog.Error("final msg write error", "err", err)
		return err
	}

	outputMsg = []byte("<div id=\"textInput\" hx-swap-oob=\"true\"></div>")
	err = ws.WriteMessage(websocket.TextMessage, outputMsg)
	if err != nil {
		slog.Error("final msg write error", "err", err)
		return err
	}

	return nil
}

// measureSpeed calculates the typing speed in characters per minute.
func measureSpeed(timestampsNs []int64) int64 {
	if len(timestampsNs) == 0 {
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

func getRandomText() string {
	textBank := []string{
		"The quick brown fox jumps over the lazy dog while the wind blows softly through the quiet evening forest.",
		"Typing fast requires focus, rhythm, and regular practice. Start slow, stay accurate, and your speed will naturally improve over time.",
		"Every mistake is a lesson. Keep your hands relaxed, eyes on the screen, and trust your muscle memory.",
		"Technology changes quickly, but good typing skills remain useful for work, study, and everyday communication.",
		"Consistency matters more than talent. Ten minutes of daily typing can bring better results than long sessions once a week.",
		"In 2024, I practiced typing for 15 minutes a day and increased my speed from 42 to 68 words per minute.",
		"The meeting starts at 9:30, ends at 11:45, and includes 3 main topics and 12 action items.",
		"She bought 2 monitors, 1 keyboard, and 5 cables for $120, saving 20% during the sale.",
		"Version 3.1.4 was released after 27 tests, 8 bug fixes, and 0 critical errors.",
		"Room 404 is on floor 7, code 9832 opens the door, and the timer locks it again after 60 seconds.",
	}
	randomIndex := rand.Intn(len(textBank))
	return textBank[randomIndex]
}
