package client

import (
	"encoding/json"
	"io"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/richinsley/comfy2go/graphapi"
)

type QueuedItemStoppedReason string

const (
	QueuedItemStoppedReasonFinished    QueuedItemStoppedReason = "finished"
	QueuedItemStoppedReasonInterrupted QueuedItemStoppedReason = "interrupted"
	QueuedItemStoppedReasonError       QueuedItemStoppedReason = "error"
)

type ComfyClientCallbacks struct {
	ClientQueueCountChanged func(*ComfyClient, int)
	QueuedItemStarted       func(*ComfyClient, *QueueItem)
	QueuedItemStopped       func(*ComfyClient, *QueueItem, QueuedItemStoppedReason)
	QueuedItemDataAvailable func(*ComfyClient, *QueueItem, *PromptMessageData)
}

// ComfyClient is the top level object that allows for interaction with the ComfyUI backend
type ComfyClient struct {
	serverBaseAddress     string
	serverAddress         string
	serverPort            int
	clientid              string
	webSocket             *WebSocketConnection
	nodeobjects           *graphapi.NodeObjects
	initialized           bool
	queueditems           map[string]*QueueItem
	queuecount            int
	callbacks             *ComfyClientCallbacks
	lastProcessedPromptID string
}

// NewComfyClient creates a new instance of a Comfy2go client
func NewComfyClient(server_address string, server_port int, callbacks *ComfyClientCallbacks) *ComfyClient {
	sbaseaddr := server_address + ":" + strconv.Itoa(server_port)
	cid := uuid.New().String()
	retv := &ComfyClient{
		serverBaseAddress: sbaseaddr,
		serverAddress:     server_address,
		serverPort:        server_port,
		clientid:          cid,
		queueditems:       make(map[string]*QueueItem),
		webSocket: &WebSocketConnection{
			WebSocketURL:   "ws://" + sbaseaddr + "/ws?clientId=" + cid,
			ConnectionDone: make(chan bool),
			MaxRetry:       5, // Maximum number of retries
			managerstarted: false,
		},
		initialized: false,
		queuecount:  0,
		callbacks:   callbacks,
	}
	// golang uses mark-sweep GC, so this circular reference should be fine
	retv.webSocket.Callback = retv
	return retv
}

// IsInitialized returns true if the client's websocket is connected
func (c *ComfyClient) IsInitialized() bool {
	return c.initialized
}

// Init starts the websocket connection (if not already connected) and retrieves the collection of node objects
func (c *ComfyClient) Init() error {
	if !c.webSocket.IsConnected {
		// as soon as the ws is connected, it will receive a "status" message of the current status
		// of the ComfyUI server
		err := c.webSocket.ConnectWithManager()
		if err != nil {
			return err
		}
	}

	// 1. Get the object infos for the Comfy Server
	object_infos, err := c.GetObjectInfos()
	if err != nil {
		return err
	}

	c.nodeobjects = object_infos
	return nil
}

// ClientID returns the unique client ID for the connection to the ComfyUI backend
func (c *ComfyClient) ClientID() string {
	return c.clientid
}

// NewGraphFromJsonReader creates a new graph from the data read from an io.Reader
func (c *ComfyClient) NewGraphFromJsonReader(r io.Reader) (*graphapi.Graph, *[]string, error) {
	if !c.IsInitialized() {
		// try to initialize first
		err := c.Init()
		if err != nil {
			return nil, nil, err
		}
	}
	return graphapi.NewGraphFromJsonReader(r, c.nodeobjects)
}

// NewGraphFromJsonFile creates a new graph from a JSON file
func (c *ComfyClient) NewGraphFromJsonFile(path string) (*graphapi.Graph, *[]string, error) {
	if !c.IsInitialized() {
		// try to initialize first
		err := c.Init()
		if err != nil {
			return nil, nil, err
		}
	}
	return graphapi.NewGraphFromJsonFile(path, c.nodeobjects)
}

// NewGraphFromPNGReader extracts the workflow from PNG data read from an io.Reader and creates a new graph
func (c *ComfyClient) NewGraphFromPNGReader(r io.Reader) (*graphapi.Graph, *[]string, error) {
	metadata, err := GetPngMetadata(r)
	if err != nil {
		return nil, nil, err
	}

	// get the workflow from the PNG metadata
	workflow, ok := metadata["workflow"]
	if !ok {
		log.Fatal("PNG doen not contain workflow metadata")
	}
	reader := strings.NewReader(workflow)

	graph, missing, err := c.NewGraphFromJsonReader(reader)
	if err != nil {
		return nil, missing, err
	}
	return graph, missing, nil
}

// NewGraphFromPNGReader extracts the workflow from PNG data read from a file and creates a new graph
func (c *ComfyClient) NewGraphFromPNGFile(path string) (*graphapi.Graph, *[]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	defer file.Close()
	return c.NewGraphFromPNGReader(file)
}

// GetQueuedItem returns a QueueItem that was queued with the ComfyClient, that has not been processed yet
// or is currently being processed.  Once a QueueItem has been processed, it will not be available with this method.
func (c *ComfyClient) GetQueuedItem(prompt_id string) *QueueItem {
	val, ok := c.queueditems[prompt_id]
	if ok {
		return val
	}
	return nil
}

// OnWindowSocketMessage processes each message received from the websocket connection to ComfyUI.
// The messages are parsed, and translated into PromptMessage structs and placed into the correct QueuedItem's message channel.
func (c *ComfyClient) OnWindowSocketMessage(msg string) {
	message := &WSStatusMessage{}
	err := json.Unmarshal([]byte(msg), &message)
	if err != nil {
		log.Println("Deserializing Status Message:", err)
	}

	switch message.Type {
	case "status":
		s := message.Data.(*WSMessageDataStatus)
		if c.callbacks != nil && c.callbacks.ClientQueueCountChanged != nil {
			c.queuecount = s.Status.ExecInfo.QueueRemaining
			c.callbacks.ClientQueueCountChanged(c, s.Status.ExecInfo.QueueRemaining)
		}
	case "execution_start":
		s := message.Data.(*WSMessageDataExecutionStart)
		qi := c.GetQueuedItem(s.PromptID)
		// update lastProcessedPromptID to indicate we are processing a new prompt
		c.lastProcessedPromptID = s.PromptID
		if qi != nil {
			if c.callbacks != nil && c.callbacks.QueuedItemStarted != nil {
				c.callbacks.QueuedItemStarted(c, qi)
			}
			m := PromptMessage{
				Type: "started",
				Message: &PromptMessageStarted{
					PromptID: qi.PromptID,
				},
			}
			qi.Messages <- m
		}
	case "execution_cached":
		// this is probably not usefull for us
	case "executing":
		s := message.Data.(*WSMessageDataExecuting)
		qi := c.GetQueuedItem(s.PromptID)

		if qi != nil {
			if s.Node == nil {
				// final node was processed
				m := PromptMessage{
					Type: "stopped",
					Message: &PromptMessageStopped{
						QueueItem: qi,
						Exception: nil,
					},
				}
				// remove the Item from our Queue before sending the message
				// no other messages will be sent to the channel after this
				if c.callbacks != nil && c.callbacks.QueuedItemStopped != nil {
					c.callbacks.QueuedItemStopped(c, qi, QueuedItemStoppedReasonFinished)
				}
				delete(c.queueditems, qi.PromptID)
				qi.Messages <- m
			} else {
				node := qi.Workflow.GetNodeById(*s.Node)
				m := PromptMessage{
					Type: "executing",
					Message: &PromptMessageExecuting{
						NodeID: *s.Node,
						Title:  node.DisplayName,
					},
				}
				qi.Messages <- m
			}
		}
	case "progress":
		s := message.Data.(*WSMessageDataProgress)
		qi := c.GetQueuedItem(c.lastProcessedPromptID)
		if qi != nil {
			m := PromptMessage{
				Type: "progress",
				Message: &PromptMessageProgress{
					Value: s.Value,
					Max:   s.Max,
				},
			}
			qi.Messages <- m
		}
	case "executed":
		s := message.Data.(*WSMessageDataExecuted)
		qi := c.GetQueuedItem(s.PromptID)
		if qi != nil {
			mdata := &PromptMessageData{
				NodeID: s.Node,
				Images: *s.Output["images"],
			}
			m := PromptMessage{
				Type:    "data",
				Message: mdata,
			}
			if c.callbacks != nil && c.callbacks.QueuedItemDataAvailable != nil {
				c.callbacks.QueuedItemDataAvailable(c, qi, mdata)
			}
			qi.Messages <- m
		}
	case "execution_interrupted":
		s := message.Data.(*WSMessageExecutionInterrupted)
		qi := c.GetQueuedItem(s.PromptID)
		if qi != nil {
			m := PromptMessage{
				Type: "stopped",
				Message: &PromptMessageStopped{
					QueueItem: qi,
					Exception: nil,
				},
			}
			// remove the Item from our Queue before sending the message
			// no other messages will be sent to the channel after this
			if c.callbacks != nil && c.callbacks.QueuedItemStopped != nil {
				c.callbacks.QueuedItemStopped(c, qi, QueuedItemStoppedReasonInterrupted)
			}
			delete(c.queueditems, qi.PromptID)
			qi.Messages <- m
		}
	case "execution_error":
		s := message.Data.(*WSMessageExecutionError)
		qi := c.GetQueuedItem(s.PromptID)
		if qi != nil {
			nindex, _ := strconv.Atoi(s.Node) // the node id is serialized as a string
			tnode := qi.Workflow.GetNodeById(nindex)
			m := PromptMessage{
				Type: "stopped",
				Message: &PromptMessageStopped{
					QueueItem: qi,
					Exception: &PromptMessageStoppedException{
						NodeID:           nindex,
						NodeType:         s.NodeType,
						NodeName:         tnode.Title,
						ExceptionMessage: s.ExceptionMessage,
						ExceptionType:    s.ExceptionType,
						Traceback:        s.Traceback,
					},
				},
			}
			// remove the Item from our Queue before sending the message
			// no other messages will be sent to the channel after this
			if c.callbacks != nil && c.callbacks.QueuedItemStopped != nil {
				c.callbacks.QueuedItemStopped(c, qi, QueuedItemStoppedReasonError)
			}
			delete(c.queueditems, qi.PromptID)
			qi.Messages <- m
		}
	default:
		// Handle unknown data types or return a dedicated error here
		// sm.Data = nil
	}
}
