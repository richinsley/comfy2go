package client

import "github.com/richinsley/comfy2go/graphapi"

// There may be other DataOutput types.  We definitely need a text type

type DataOutput struct {
	Filename  string `json:"filename"`
	Subfolder string `json:"subfolder"`
	Type      string `json:"type"`
	Text      string `json:"-"` // for "text" type data output
}

type SystemStats struct {
	System  System `json:"system"`
	Devices []GPU  `json:"devices"`
}

type System struct {
	OS             string `json:"os"`
	PythonVersion  string `json:"python_version"`
	EmbeddedPython bool   `json:"embedded_python"`
}

type GPU struct {
	Name             string `json:"name"`
	Type             string `json:"type"`
	Index            int    `json:"index"`
	VRAM_Total       int64  `json:"vram_total"`
	VRAM_Free        int64  `json:"vram_free"`
	Torch_VRAM_Total int64  `json:"torch_vram_total"`
	Torch_VRAM_Free  int64  `json:"torch_vram_free"`
}

type QueueExecInfo struct {
	ExecInfo struct {
		QueueRemaining int `json:"queue_remaining"`
	} `json:"exec_info"`
}

type PromptHistoryItem struct {
	PromptID string
	Index    int
	Graph    *graphapi.Graph
	Outputs  map[int][]DataOutput
}

type PromptError struct {
	Type      string                 `json:"type"`
	Message   string                 `json:"message"`
	Details   string                 `json:"details"`
	ExtraInfo map[string]interface{} `json:"extra_info"`
}

type PromptErrorMessage struct {
	Error      PromptError   `json:"error"`
	NodeErrors []interface{} `json:"node_errors"`
}
