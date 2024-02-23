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
	NodeID int
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
	NodeID int
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
	NodeID           int
	NodeType         string
	NodeName         string
	ExceptionMessage string
	ExceptionType    string
	Traceback        []string
}

func (p *PromptMessage) ToPromptMessageStopped() *PromptMessageStopped {
	return p.Message.(*PromptMessageStopped)
}
