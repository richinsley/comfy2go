package client

import (
	"fmt"
	"log/slog"
	"math"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Callback interface for handling incoming WebSocket messages
type WebSocketCallback interface {
	OnMessage(message string)
}

type WebSocketConnection struct {
	WebSocketURL   string
	Conn           *websocket.Conn
	ConnectionDone chan bool
	IsConnected    bool
	MaxRetry       int
	RetryCount     int
	ManagerStarted bool
	mu             sync.Mutex // For thread-safe access to the WebSocket connection
	Callback       WebSocketCallback

	// Exponential backoff configuration
	BaseDelay time.Duration // The initial delay, e.g., 1 second
	MaxDelay  time.Duration // The maximum delay, e.g., 1 minute
	Dialer    websocket.Dialer
}

// ConnectWithManager connects to the WebSocket using a connection manager
// timeoutSeconds is the maximum time to wait for a successful connection (0 for no timeout)
func (w *WebSocketConnection) ConnectWithManager(timeoutSeconds int) error {
	// time.Duration
	// Channel to signal successful connection
	connected := make(chan bool, 1)
	// Channel for connection attempts (ensures connect() is not called concurrently)
	attemptConnect := make(chan bool, 1)
	attemptConnect <- true // Trigger the first connection attempt immediately

	go func() {
		retries := 0
		for {
			select {
			case <-attemptConnect:
				err := w.connect()
				if err != nil {
					slog.Error("Connection attempt failed: ", "error", err)
					w.IsConnected = false

					// Check if the maximum number of retries has been reached
					retries++
					if retries > w.MaxRetry {
						slog.Error(fmt.Sprintf("Maximum number of retries reached (%d)", w.MaxRetry))
						close(connected) // Signal that the connection failed
						return
					}

					// Wait a bit before retrying to connect
					time.AfterFunc(w.getReconnectDelay(), func() {
						attemptConnect <- true
					})
				} else {
					w.IsConnected = true
					close(connected) // Signal that the connection was successful
					w.handleMessages()
					return // Exit the goroutine once connected
				}
			case <-w.ConnectionDone:
				// Handle graceful shutdown if needed
				return
			}
		}
	}()

	// Block until either a successful connection or timeout
	if timeoutSeconds > 0 {
		timeout := time.Duration(timeoutSeconds) * time.Second
		select {
		case <-connected:
			// Connection was successful before the timeout
			return nil
		case <-time.After(timeout):
			// Timeout occurred before a successful connection
			return fmt.Errorf("connection timeout after %v", timeout)
		}
	} else if timeoutSeconds < 0 {
		// wait indefinitely
		<-connected
	}

	return nil
}

// Initial connection logic with exponential backoff for reconnections
func (w *WebSocketConnection) connect() error {
	conn, _, err := w.Dialer.Dial(w.WebSocketURL, nil)
	if err != nil {
		slog.Error("Failed to connect: ", "error", err)
		return err
	}

	w.Conn = conn
	return nil
}

func (w *WebSocketConnection) Ping() error {
	// Attempt to send a ping message
	err := w.Conn.WriteMessage(websocket.PingMessage, nil)
	if err != nil {
		return err
	}
	return nil
}

// Handle incoming WebSocket messages
func (w *WebSocketConnection) handleMessages() {
	defer func() {
		w.Conn.Close()
		w.ConnectionDone <- true
	}()
	for {
		_, message, err := w.Conn.ReadMessage()
		if err != nil {
			slog.Warn(fmt.Sprintf("Read error: %v", err))
			break
		}
		if w.Callback != nil {
			w.Callback.OnMessage(string(message))
		}
	}
}

// exponential backoff calculation
func (w *WebSocketConnection) getReconnectDelay() time.Duration {
	// Calculate the delay as BaseDelay * 2^(RetryCount), capped at MaxDelay
	delay := w.BaseDelay * time.Duration(math.Pow(2, float64(w.RetryCount)))
	if delay > w.MaxDelay {
		delay = w.MaxDelay
	}
	w.RetryCount++ // Increment the retry counter for the next attempt
	return delay
}

func (w *WebSocketConnection) LockRead() {
	w.mu.Lock()
}

func (w *WebSocketConnection) UnlockRead() {
	w.mu.Unlock()
}
