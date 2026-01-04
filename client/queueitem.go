package client

import "github.com/richinsley/comfy2go/graphapi"

type QueueItem struct {
	PromptID    string                 `json:"prompt_id"`
	Number      int                    `json:"number"`
	NodeErrors  map[string]interface{} `json:"node_errors"`
	Messages    chan PromptMessage     `json:"-"`
	Workflow    *graphapi.Graph        `json:"-"`
	webSocket *WebSocketConnection   `json:"-"`
}

// Close closes the websocket connection associated with the QueueItem
func (qi *QueueItem) Close() {
	if qi.webSocket != nil {
		qi.webSocket.Close()
	}
}
