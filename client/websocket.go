package client

import (
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type WSCallback interface {
	OnWindowSocketMessage(string)
}

type WebSocketConnection struct {
	WebSocketURL   string // The URL of the WebSocket
	Conn           *websocket.Conn
	ConnectionDone chan bool
	IsConnected    bool
	MaxRetry       int
	RetryCount     int
	managerstarted bool
	mu             sync.Mutex
	Callback       WSCallback
}

func (w *WebSocketConnection) ConnectWithManager() error {
	if !w.managerstarted {
		w.managerstarted = true
		go w.ConnectionManager()
	}
	return nil
}

// Connect to the WebSocket
func (w *WebSocketConnection) connect() error {
	var err error
	w.Conn, _, err = websocket.DefaultDialer.Dial(w.WebSocketURL, nil)
	if err != nil {
		return err
	}
	w.IsConnected = true
	return nil
}

// Close the WebSocket connection
func (w *WebSocketConnection) Close() error {
	err := w.Conn.Close()
	if err != nil {
		return err
	}
	w.IsConnected = false
	return nil
}

// Manage connection lifecycle
func (w *WebSocketConnection) ConnectionManager() {
	for {
		select {
		case <-w.ConnectionDone:
			err := w.Close()
			if err != nil {
				log.Println("Error when closing WebSocket: ", err)
			}
		default:
			err := w.connect()
			if err != nil {
				log.Println("Error when connecting to WebSocket: ", err)
				w.RetryCount++
				if w.RetryCount <= w.MaxRetry {
					time.Sleep(5 * time.Second) // constant delay before retrying
					continue
				}
				w.IsConnected = false
				return
			} else {
				w.RetryCount = 0 // reset retry count after successful connection
				w.listen()
			}
		}
	}
}

// listen listens for incoming messages and calls OnWindowSocketMessage for each message received
func (w *WebSocketConnection) listen() {
	defer w.Conn.Close()
	for {
		_, message, err := w.Conn.ReadMessage()
		if err != nil {
			log.Println("Error when reading from WebSocket: ", err)
			w.IsConnected = false
			w.ConnectionDone <- true
			break
		}
		w.mu.Lock()
		if w.Callback != nil {
			w.Callback.OnWindowSocketMessage(string(message))
		}
		w.mu.Unlock()
	}
}

func (w *WebSocketConnection) LockRead() {
	w.mu.Lock()
}

func (w *WebSocketConnection) UnlockRead() {
	w.mu.Unlock()
}
