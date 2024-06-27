package client

import "github.com/richinsley/comfy2go/graphapi"

type QueueItem struct {
	PromptID   string                 `json:"prompt_id"`
	Number     int                    `json:"number"`
	NodeErrors map[string]interface{} `json:"node_errors"`
	Messages   chan PromptMessage     `json:"-"`
	Workflow   *graphapi.Graph        `json:"-"`
	Error      struct {
		Type      string `json:"type"`
		Message   string `json:"message"`
		Details   string `json:"details"`
		ExtraInfo struct {
		} `json:"extra_info"`
	} `json:"error"`
}
