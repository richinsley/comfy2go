package client

import (
	"sync"

	"github.com/richinsley/comfy2go/graphapi"
)

type QueueItem struct {
	PromptID   string                 `json:"prompt_id"`
	Number     int                    `json:"number"`
	NodeErrors map[string]interface{} `json:"node_errors"`

	Messages chan PromptMessage `json:"-"`
	Workflow *graphapi.Graph    `json:"-"`

	webSocket   *WebSocketConnection `json:"-"`
	done        chan struct{}        `json:"-"`
	wsCloseOnce sync.Once            `json:"-"`
	closeOnce   sync.Once            `json:"-"`
}

func (qi *QueueItem) Done() <-chan struct{} {
	// If qi is nil, treat it as already closed.
	if qi == nil {
		ch := make(chan struct{})
		close(ch)
		return ch
	}
	// done should be initialized when the QueueItem is created.
	// If it's nil (e.g., constructed externally), we treat it as "never closed"
	// so it won't spuriously win in select and cause message loss.
	return qi.done
}

func (qi *QueueItem) send(msg PromptMessage) {
	if qi == nil || qi.Messages == nil {
		return
	}
	select {
	case qi.Messages <- msg:
	case <-qi.Done():
		// Client/QueueItem is closing; stop delivering messages.
	}
}

// CloseWebSocket closes the websocket connection associated with the QueueItem.
// It does NOT signal Done; callers should send any final messages first.
func (qi *QueueItem) CloseWebSocket() {
	if qi == nil {
		return
	}
	qi.wsCloseOnce.Do(func() {
		if qi.webSocket != nil {
			qi.webSocket.Close()
			qi.webSocket = nil
		}
	})
}

// Close releases QueueItem resources and signals all waiters.
func (qi *QueueItem) Close() {
	if qi == nil {
		return
	}
	qi.closeOnce.Do(func() {
		qi.CloseWebSocket()
		if qi.done != nil {
			close(qi.done)
		}
	})
}
