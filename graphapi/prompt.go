package graphapi

// Prompt is the data that is enqueued to an instance of ComfyUI
type Prompt struct {
	ClientID  string             `json:"client_id"`
	Nodes     map[int]PromptNode `json:"prompt"`
	ExtraData PromptExtraData    `json:"extra_data"`
	PID       string             `json:"pid"`
}

type PromptNode struct {
	// Inputs can be one of:
	//	float64
	//	string
	//	[]interface{} where: [0] is string of target node
	//					     [1] is float64 (int) of slot index
	Inputs    map[string]interface{} `json:"inputs"`
	ClassType string                 `json:"class_type"`
}

type PromptExtraData struct {
	PngInfo PromptWorkflow `json:"extra_pnginfo"`
}

// PromptWorkflow is the original Graph that was used to create the Prompt.
// It is added to generated PNG files such that the information needed to
// recreate the image is available.
type PromptWorkflow struct {
	Workflow *Graph `json:"workflow"`
}
