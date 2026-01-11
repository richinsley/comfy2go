package client

type PromptMessage struct {
	Type    string
	Message interface{}
}

// our cast of characters:
// queued
// started
// executing
// progress
// data
// stopped
// progress_state
// execution_success

type PromptMessageQueued struct {
}

func (p *PromptMessage) ToPromptMessageQueued() *PromptMessageQueued {
	return p.Message.(*PromptMessageQueued)
}

type PromptMessageStarted struct {
	PromptID string `json:"prompt_id"`
}

func (p *PromptMessage) ToPromptMessageStarted() *PromptMessageStarted {
	return p.Message.(*PromptMessageStarted)
}

type PromptMessageExecuting struct {
	NodeID string
	Title  string
}

func (p *PromptMessage) ToPromptMessageExecuting() *PromptMessageExecuting {
	return p.Message.(*PromptMessageExecuting)
}

type PromptMessageProgress struct {
	Max   int
	Value int
}

func (p *PromptMessage) ToPromptMessageProgress() *PromptMessageProgress {
	return p.Message.(*PromptMessageProgress)
}

type PromptMessageData struct {
	NodeID string
	Data   map[string][]DataOutput
}

func (p *PromptMessage) ToPromptMessageData() *PromptMessageData {
	return p.Message.(*PromptMessageData)
}

type PromptMessageStopped struct {
	QueueItem *QueueItem
	Exception *PromptMessageStoppedException
}

type PromptMessageStoppedException struct {
	NodeID           string
	NodeType         string
	NodeName         string
	ExceptionMessage string
	ExceptionType    string
	Traceback        []string
}

func (p *PromptMessage) ToPromptMessageStopped() *PromptMessageStopped {
	return p.Message.(*PromptMessageStopped)
}

type PromptMessageProgressState struct {
	PromptID string
	Nodes    map[string]NodeProgressInfo
}

type NodeProgressInfo struct {
	Value         float64
	Max           float64
	State         string
	NodeID        string
	DisplayNodeID string
	ParentNodeID  *string
	RealNodeID    string
}

func (p *PromptMessage) ToPromptMessageProgressState() *PromptMessageProgressState {
	return p.Message.(*PromptMessageProgressState)
}

type PromptMessageExecutionSuccess struct {
	PromptID  string
	Timestamp int64
}

func (p *PromptMessage) ToPromptMessageExecutionSuccess() *PromptMessageExecutionSuccess {
	return p.Message.(*PromptMessageExecutionSuccess)
}
