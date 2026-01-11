package client

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
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
	nodeobjects           *graphapi.NodeObjects
	initialized           bool
	queueditems           map[string]*QueueItem
	queuecount            int
	callbacks             *ComfyClientCallbacks
	lastProcessedPromptID string
	timeout               int
	httpclient            *http.Client
}

// NewComfyClientWithTimeout creates a new instance of a Comfy2go client with a connection timeout
func NewComfyClientWithTimeout(server_address string, server_port int, callbacks *ComfyClientCallbacks, timeout int, retry int) *ComfyClient {
	sbaseaddr := server_address + ":" + strconv.Itoa(server_port)
	cid := uuid.New().String()
	retv := &ComfyClient{
		serverBaseAddress: sbaseaddr,
		serverAddress:     server_address,
		serverPort:        server_port,
		clientid:          cid,
		queueditems:       make(map[string]*QueueItem),
		initialized:       false,
		queuecount:        0,
		callbacks:         callbacks,
		timeout:           timeout,
		httpclient:        &http.Client{},
	}
	return retv
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
		initialized:       false,
		queuecount:        0,
		callbacks:         callbacks,
		timeout:           -1,
		httpclient:        &http.Client{},
	}
	return retv
}

// IsInitialized returns true if the client's websocket is connected and initialized
func (c *ComfyClient) IsInitialized() bool {
	return c.initialized
}

// CheckConnection checks if the websocket connection is still active and tries to reinitialize if not
func (c *ComfyClient) CheckConnection() error {
	if !c.IsInitialized() {
		// try to initialize first
		err := c.Init()
		if err != nil {
			return err
		}
	}
	return nil
}

// Init starts the websocket connection (if not already connected) and retrieves the collection of node objects
func (c *ComfyClient) Init() error {
	// Get the object infos for the Comfy Server
	object_infos, err := c.GetObjectInfos()
	if err != nil {
		return err
	}

	c.nodeobjects = object_infos
	c.initialized = true
	return nil
}

// ClientID returns the unique client ID for the connection to the ComfyUI backend
func (c *ComfyClient) ClientID() string {
	return c.clientid
}

// return the underlying http client
func (c *ComfyClient) HttpClient() *http.Client {
	return c.httpclient
}

// set the underlying http client
func (c *ComfyClient) SetHttpClient(client *http.Client) {
	c.httpclient = client
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

// NewGraphFromJsonString creates a new graph from a JSON string
func (c *ComfyClient) NewGraphFromJsonString(path string) (*graphapi.Graph, *[]string, error) {
	if !c.IsInitialized() {
		// try to initialize first
		err := c.Init()
		if err != nil {
			return nil, nil, err
		}
	}
	return graphapi.NewGraphFromJsonString(path, c.nodeobjects)
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
		return nil, nil, errors.New("png does not contain workflow metadata")
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
func (c *ComfyClient) OnWindowSocketMessage(msg string, qi *QueueItem) {
	message := &WSStatusMessage{}
	err := json.Unmarshal([]byte(msg), &message)
	if err != nil {
		slog.Error("Deserializing Status Message:", "error", err)
	}

	// fmt.Println(msg)

	switch message.Type {
	case "status":
		s := message.Data.(*WSMessageDataStatus)
		if c.callbacks != nil && c.callbacks.ClientQueueCountChanged != nil {
			c.queuecount = s.Status.ExecInfo.QueueRemaining
			c.callbacks.ClientQueueCountChanged(c, s.Status.ExecInfo.QueueRemaining)
		}
	case "execution_start":
		s := message.Data.(*WSMessageDataExecutionStart)
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
				// qi.Close()
				qi.Messages <- m
			} else {
				// Try to find the node in the workflow
				// For compound IDs like "57:8", parse the first part
				var node *graphapi.GraphNode
				nodeIDStr := *s.Node
				if nodeID, err := strconv.Atoi(nodeIDStr); err == nil {
					// Simple integer ID
					node = qi.Workflow.GetNodeById(nodeID)
				} else if strings.Contains(nodeIDStr, ":") {
					// Compound ID like "57:8" - try to get the instance node
					parts := strings.Split(nodeIDStr, ":")
					if instanceID, err := strconv.Atoi(parts[0]); err == nil {
						node = qi.Workflow.GetNodeById(instanceID)
					}
				}

				if node != nil {
					m := PromptMessage{
						Type: "executing",
						Message: &PromptMessageExecuting{
							NodeID: *s.Node,
							Title:  node.DisplayName,
						},
					}
					qi.Messages <- m
				} else {
					m := PromptMessage{
						Type: "executing",
						Message: &PromptMessageExecuting{
							NodeID: *s.Node,
							Title:  *s.Node,
						},
					}
					qi.Messages <- m
				}
			}
		}
	case "progress":
		s := message.Data.(*WSMessageDataProgress)
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
		if qi != nil {
			// mdata := &PromptMessageData{
			// 	NodeID: s.Node,
			// 	Images: *s.Output["images"],
			// }

			// collect the data from the output
			mdata := &PromptMessageData{
				NodeID: s.Node,
				Data:   make(map[string][]DataOutput),
			}

			for k, v := range s.Output {
				mdata.Data[k] = *v
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
			qi.Close()
			qi.Messages <- m
		}
	case "execution_error":
		s := message.Data.(*WSMessageExecutionError)
		if qi != nil {
			// Try to find the node in the workflow
			var tnode *graphapi.GraphNode
			if nodeID, err := strconv.Atoi(s.Node); err == nil {
				tnode = qi.Workflow.GetNodeById(nodeID)
			} else if strings.Contains(s.Node, ":") {
				// Compound ID - try to get the instance node
				parts := strings.Split(s.Node, ":")
				if instanceID, err := strconv.Atoi(parts[0]); err == nil {
					tnode = qi.Workflow.GetNodeById(instanceID)
				}
			}

			nodeName := s.Node
			if tnode != nil {
				nodeName = tnode.Title
			}

			m := PromptMessage{
				Type: "stopped",
				Message: &PromptMessageStopped{
					QueueItem: qi,
					Exception: &PromptMessageStoppedException{
						NodeID:           s.Node,
						NodeType:         s.NodeType,
						NodeName:         nodeName,
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
			qi.Close()
			qi.Messages <- m
		}
	case "crystools.monitor":
	default:
		// Handle unknown data types or return a dedicated error here
		slog.Warn("Unhandled message type: ", "type", message.Type)
	}
}
