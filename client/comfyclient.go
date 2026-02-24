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
// The messages are parsed and translated into PromptMessage structs and routed into the QueueItem.
func (c *ComfyClient) OnWindowSocketMessage(msg string, qi *QueueItem) {
	message := &WSStatusMessage{}
	if err := json.Unmarshal([]byte(msg), &message); err != nil {
		slog.Error("Deserializing Status Message", "error", err)
		return
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
		// Update lastProcessedPromptID to indicate we are processing a new prompt.
		c.lastProcessedPromptID = s.PromptID
		if qi == nil {
			return
		}
		if c.callbacks != nil && c.callbacks.QueuedItemStarted != nil {
			c.callbacks.QueuedItemStarted(c, qi)
		}
		qi.send(PromptMessage{
			Type: "started",
			Message: &PromptMessageStarted{
				PromptID: qi.PromptID,
			},
		})

	case "execution_cached":
		// Intentionally ignored.

	case "executing":
		s := message.Data.(*WSMessageDataExecuting)
		if qi == nil {
			return
		}
		if s.Node == nil {
			// Final node was processed.
			if c.callbacks != nil && c.callbacks.QueuedItemStopped != nil {
				c.callbacks.QueuedItemStopped(c, qi, QueuedItemStoppedReasonFinished)
			}
			delete(c.queueditems, qi.PromptID)
			qi.send(PromptMessage{
				Type: "stopped",
				Message: &PromptMessageStopped{
					QueueItem: qi,
					Exception: nil,
				},
			})
			// Release websocket resources for this item.
			qi.CloseWebSocket()
			return
		}

		// Try to find the node in the workflow.
		// For compound IDs like "57:8", parse the first part.
		var node *graphapi.GraphNode
		nodeIDStr := *s.Node
		if nodeID, err := strconv.Atoi(nodeIDStr); err == nil {
			// Simple integer ID.
			node = qi.Workflow.GetNodeById(nodeID)
		} else if strings.Contains(nodeIDStr, ":") {
			// Compound ID like "57:8" - try to get the instance node.
			parts := strings.Split(nodeIDStr, ":")
			if instanceID, err := strconv.Atoi(parts[0]); err == nil {
				node = qi.Workflow.GetNodeById(instanceID)
			}
		}

		title := *s.Node
		if node != nil {
			title = node.DisplayName
		}
		qi.send(PromptMessage{
			Type: "executing",
			Message: &PromptMessageExecuting{
				NodeID: *s.Node,
				Title:  title,
			},
		})

	case "progress":
		s := message.Data.(*WSMessageDataProgress)
		if qi == nil {
			return
		}
		qi.send(PromptMessage{
			Type: "progress",
			Message: &PromptMessageProgress{
				Value: s.Value,
				Max:   s.Max,
			},
		})

	case "executed":
		s := message.Data.(*WSMessageDataExecuted)
		if qi == nil {
			return
		}
		// Collect the data from the output.
		mdata := &PromptMessageData{
			NodeID: s.Node,
			Data:   make(map[string][]DataOutput),
		}
		for k, v := range s.Output {
			mdata.Data[k] = *v
		}
		if c.callbacks != nil && c.callbacks.QueuedItemDataAvailable != nil {
			c.callbacks.QueuedItemDataAvailable(c, qi, mdata)
		}
		qi.send(PromptMessage{Type: "data", Message: mdata})

	case "execution_interrupted":
		if qi == nil {
			return
		}
		if c.callbacks != nil && c.callbacks.QueuedItemStopped != nil {
			c.callbacks.QueuedItemStopped(c, qi, QueuedItemStoppedReasonInterrupted)
		}
		delete(c.queueditems, qi.PromptID)
		qi.send(PromptMessage{
			Type: "stopped",
			Message: &PromptMessageStopped{
				QueueItem: qi,
				Exception: nil,
			},
		})
		qi.CloseWebSocket()

	case "execution_error":
		s := message.Data.(*WSMessageExecutionError)
		if qi == nil {
			return
		}

		// Try to find the node in the workflow.
		var tnode *graphapi.GraphNode
		if nodeID, err := strconv.Atoi(s.Node); err == nil {
			tnode = qi.Workflow.GetNodeById(nodeID)
		} else if strings.Contains(s.Node, ":") {
			// Compound ID - try to get the instance node.
			parts := strings.Split(s.Node, ":")
			if instanceID, err := strconv.Atoi(parts[0]); err == nil {
				tnode = qi.Workflow.GetNodeById(instanceID)
			}
		}

		nodeName := s.Node
		if tnode != nil {
			nodeName = tnode.Title
		}

		if c.callbacks != nil && c.callbacks.QueuedItemStopped != nil {
			c.callbacks.QueuedItemStopped(c, qi, QueuedItemStoppedReasonError)
		}
		delete(c.queueditems, qi.PromptID)
		qi.send(PromptMessage{
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
		})
		qi.CloseWebSocket()

	case "progress_state":
		s := message.Data.(*WSMessageDataProgressState)
		if qi == nil {
			return
		}
		// Convert the map of node progress states to application-level format.
		nodes := make(map[string]NodeProgressInfo)
		for nodeID, nodeState := range s.Nodes {
			nodes[nodeID] = NodeProgressInfo{
				Value:         nodeState.Value,
				Max:           nodeState.Max,
				State:         nodeState.State,
				NodeID:        nodeState.NodeID,
				DisplayNodeID: nodeState.DisplayNodeID,
				ParentNodeID:  nodeState.ParentNodeID,
				RealNodeID:    nodeState.RealNodeID,
			}
		}
		qi.send(PromptMessage{
			Type: "progress_state",
			Message: &PromptMessageProgressState{
				PromptID: s.PromptID,
				Nodes:    nodes,
			},
		})

	case "execution_success":
		s := message.Data.(*WSMessageDataExecutionSuccess)
		if qi == nil {
			return
		}
		qi.send(PromptMessage{
			Type: "execution_success",
			Message: &PromptMessageExecutionSuccess{
				PromptID:  s.PromptID,
				Timestamp: s.Timestamp,
			},
		})

	case "crystools.monitor":
		// Intentionally ignored.

	default:
		slog.Warn("Unhandled message type", "type", message.Type)
	}
}

// Close closes all websocket connections and cleans up resources
func (c *ComfyClient) Close() error {
	var lastErr error

	// Close all queued items' websocket connections.
	// Copy items first to avoid holding locks while closing resources.
	items := make([]*QueueItem, 0, len(c.queueditems))
	for _, item := range c.queueditems {
		items = append(items, item)
	}

	for _, item := range items {
		if item != nil {
			item.Close()
		}
	}

	// Close idle HTTP keep-alive connections (best effort).
	if c.httpclient != nil {
		c.httpclient.CloseIdleConnections()
	}

	// Clear the queue.
	c.queueditems = make(map[string]*QueueItem)
	c.initialized = false

	return lastErr
}
