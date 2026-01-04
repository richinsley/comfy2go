package client

import (
	"fmt"
	"log/slog"

	"github.com/gorilla/websocket"
)

type WebSocketConnection struct {
	WebSocketURL string
	Conn         *websocket.Conn
	IsConnected  bool
	Dialer       websocket.Dialer
}

// Connect connects to the WebSocket
func (w *WebSocketConnection) Connect(timeoutSeconds int) error {
	conn, _, err := w.Dialer.Dial(w.WebSocketURL, nil)
	if err != nil {
		slog.Error("Failed to connect: ", "error", err)
		return err
	}
	w.Conn = conn
	w.IsConnected = true
	return nil
}

// Handle incoming WebSocket messages
func (w *WebSocketConnection) HandleMessages(handler func(message string)) {
	defer func() {
		w.Close()
	}()
	for {
		_, message, err := w.Conn.ReadMessage()
		if err != nil {
			// It's normal to get a close error when we're done, so we'll log it as a warning.
			slog.Warn(fmt.Sprintf("WebSocket read error: %v", err))
			break
		}
		handler(string(message))
	}
}

// Close the WebSocket connection
func (w *WebSocketConnection) Close() {
	if w.IsConnected && w.Conn != nil {
		w.Conn.Close()
		w.IsConnected = false
	}
}
