package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"github.com/richinsley/comfy2go/graphapi"
)

/*
@routes.get("/embeddings")
@routes.get("/extensions")
@routes.get("/view")
@routes.get("/view_metadata/{folder_name}")
@routes.get("/system_stats")
@routes.get("/prompt")
@routes.get("/object_info")
@routes.get("/object_info/{node_class}")
@routes.get("/history")
@routes.get("/history/{prompt_id}")
@routes.get("/queue")

@routes.post("/prompt")
@routes.post("/queue")
@routes.post("/interrupt")
@routes.post("/history")
@routes.post("/upload/image")
@routes.post("/upload/mask")
*/

func (c *ComfyClient) GetSystemStats() (*SystemStats, error) {
	err := c.CheckConnection()
	if err != nil {
		return nil, err
	}

	resp, err := http.Get(fmt.Sprintf("http://%s/system_stats", c.serverBaseAddress))
	if err != nil {
		return nil, err
	}

	body, _ := io.ReadAll(resp.Body)
	retv := &SystemStats{}
	err = json.Unmarshal(body, &retv)
	if err != nil {
		return nil, err
	}

	return retv, nil
}

func (c *ComfyClient) GetPromptHistoryByIndex() ([]PromptHistoryItem, error) {
	history, err := c.GetPromptHistoryByID()
	if err != nil {
		return nil, err
	}

	retv := make([]PromptHistoryItem, len(history))
	index := 0
	// ComfyUI does not recalculate the indicies of prompt history items,
	// so the indecies may not always be ordered 0..n
	// We'll create a slice out of the map items, and then sort them
	for _, h := range history {
		retv[index] = h
		index++
	}

	sort.Slice(retv, func(i, j int) bool {
		return retv[i].Index < retv[j].Index
	})

	return retv, nil
}

func (c *ComfyClient) GetPromptHistoryByID() (map[string]PromptHistoryItem, error) {
	resp, err := http.Get(fmt.Sprintf("http://%s/history", c.serverBaseAddress))
	if err != nil {
		return nil, err
	}

	// we need to re-arrange the data into something more coherent
	// We're going to have to make an adapter that reconstructs an actual prompt
	// from the mangled data
	type internalOutputs struct {
		Images *[]DataOutput `json:"images"`
	}
	type internalPromptHistoryItem struct {
		// The prompt is stored as an array layed out like this:
		// [
		// 	[0] index 		int,
		// 	[1] promptID 	string,
		// 	[2] prompt 		map[string]graphapi.PromptNode, // we'll ignore this
		// 	[3] extra_data 	graphapi.PromptExtraData,       // the graph is in here
		//  [4] outputs     []string 						// array of nodeIDs that have outputs
		// ]
		Prompt  []interface{}              `json:"prompt"`
		Outputs map[string]internalOutputs `json:"outputs"`
	}

	// read in the body, and deserialize to our temp internalPromptHistoryItem type
	body, _ := io.ReadAll(resp.Body)
	history := make(map[string]internalPromptHistoryItem)
	err = json.Unmarshal(body, &history)
	if err != nil {
		return nil, err
	}

	// try to reconstruct the data into PromptHistoryItem
	ret := make(map[string]PromptHistoryItem)
	for k, ph := range history {
		index := int(ph.Prompt[0].(float64))

		// extract the graph from ph.Prompt[3]["extra_pnginfo"]["workflow"]
		extra_data, _ := ph.Prompt[3].(map[string]interface{})
		extra_pnginfo, _ := extra_data["extra_pnginfo"].(map[string]interface{})
		workflow := extra_pnginfo["workflow"]
		// workflow is now an interface{}
		// serialize it back and re-deserialize as a graph
		// this could be more efficient with raw json, but ugh!
		gdata, _ := json.Marshal(workflow)
		graph := &graphapi.Graph{}
		err = json.Unmarshal(gdata, &graph)
		if err != nil {
			return nil, err
		}

		// reconstruct
		item := &PromptHistoryItem{
			PromptID: k,
			Index:    index,
			Graph:    graph,
			Outputs:  make(map[int][]DataOutput),
		}

		// rebuild the images output map
		for k, o := range ph.Outputs {
			oid, _ := strconv.Atoi(k)
			item.Outputs[oid] = *o.Images
		}
		ret[k] = *item
	}
	return ret, nil
}

// GetViewMetadata retrieves the '__metadata__' field in a safetensors file.
// checkpoints
// vae
// loras
// clip
// unet
// controlnet
// style_models
// clip_vision
// gligen
// configs
// hypernetworks
// upscale_models
// onnx
// fonts
func (c *ComfyClient) GetViewMetadata(folder string, file string) (string, error) {
	resp, err := http.Get(fmt.Sprintf("http://%s/view_metadata/%s?filename=%s", c.serverBaseAddress, folder, file))
	if err != nil {
		return "", err
	}

	body, _ := io.ReadAll(resp.Body)
	return string(body), nil
}

// GetImage
func (c *ComfyClient) GetImage(image_data DataOutput) (*[]byte, error) {
	params := url.Values{}
	params.Add("filename", image_data.Filename)
	params.Add("subfolder", image_data.Subfolder)
	params.Add("type", image_data.Type)
	resp, err := http.Get(fmt.Sprintf("http://%s/view?%s", c.serverBaseAddress, params.Encode()))

	if err != nil {
		return nil, err
	}

	body, _ := io.ReadAll(resp.Body)
	return &body, nil
}

// GetEmbeddings retrieves the list of Embeddings models installed on the ComfyUI server.
func (c *ComfyClient) GetEmbeddings() ([]string, error) {
	resp, err := http.Get(fmt.Sprintf("http://%s/embeddings", c.serverBaseAddress))
	if err != nil {
		return nil, err
	}

	body, _ := io.ReadAll(resp.Body)
	retv := make([]string, 0)
	err = json.Unmarshal(body, &retv)
	if err != nil {
		return nil, err
	}

	return retv, nil
}

func (c *ComfyClient) GetQueueExecutionInfo() (*QueueExecInfo, error) {
	resp, err := http.Get(fmt.Sprintf("http://%s/prompt", c.serverBaseAddress))
	if err != nil {
		return nil, err
	}

	body, _ := io.ReadAll(resp.Body)
	queue_exec := &QueueExecInfo{}
	err = json.Unmarshal(body, &queue_exec)
	if err != nil {
		return nil, err
	}

	return queue_exec, nil
}

// GetExtensions retrieves the list of extensions installed on the ComfyUI server.
func (c *ComfyClient) GetExtensions() ([]string, error) {
	resp, err := http.Get(fmt.Sprintf("http://%s/extensions", c.serverBaseAddress))
	if err != nil {
		return nil, err
	}

	body, _ := io.ReadAll(resp.Body)
	retv := make([]string, 0)
	err = json.Unmarshal(body, &retv)
	if err != nil {
		return nil, err
	}

	return retv, nil
}

func (c *ComfyClient) GetObjectInfos() (*graphapi.NodeObjects, error) {
	resp, err := http.Get(fmt.Sprintf("http://%s/object_info", c.serverBaseAddress))

	if err != nil {
		return nil, err
	}

	body, _ := io.ReadAll(resp.Body)
	result := &graphapi.NodeObjects{}
	err = json.Unmarshal(body, &result.Objects)
	if err != nil {
		return nil, err
	}

	result.PopulateInputProperties()
	return result, nil
}

func (c *ComfyClient) QueuePrompt(graph *graphapi.Graph) (*QueueItem, error) {
	err := c.CheckConnection()
	if err != nil {
		return nil, err
	}

	prompt, err := graph.GraphToPrompt(c.clientid)
	if err != nil {
		return nil, err
	}

	// prevent a race where the ws may provide messages about a queued item before
	// we add the item to our internal map
	c.webSocket.LockRead()
	defer c.webSocket.UnlockRead()

	data, _ := json.Marshal(prompt)
	resp, err := http.Post(fmt.Sprintf("http://%s/prompt", c.serverBaseAddress), "application/json", strings.NewReader(string(data)))

	if err != nil {
		return nil, err
	}

	body, _ := io.ReadAll(resp.Body)

	// create the queue item
	item := &QueueItem{
		Workflow: graph,
		Messages: make(chan PromptMessage),
	}

	err = json.Unmarshal(body, &item)
	if err != nil {
		// mmm-k, is it one of these:
		// {"error": {"type": "prompt_no_outputs",
		//				"message": "Prompt has no outputs",
		//				"details": "",
		//				"extra_info": {}
		//			  },
		// "node_errors": []
		// }
		perror := &PromptErrorMessage{}
		perr := json.Unmarshal(body, &perror)
		if perr != nil {
			// return the original error
			slog.Error("error unmarshalling prompt error", "body", string(body))
			return nil, err
		} else {
			return nil, errors.New(perror.Error.Message)
		}
	}
	c.queueditems[item.PromptID] = item
	return item, nil
}

func (c *ComfyClient) Interrupt() error {
	resp, err := http.Post(fmt.Sprintf("http://%s/interrupt", c.serverBaseAddress), "application/json", strings.NewReader("{}"))
	if err != nil {
		return err
	}

	io.ReadAll(resp.Body)
	return nil
}

func (c *ComfyClient) EraseHistory() error {
	// delete post takes an array of IDs. We'll provide a single ID in a json array
	data := "{\"clear\": \"clear\"}"
	resp, err := http.Post(fmt.Sprintf("http://%s/history", c.serverBaseAddress), "application/json", strings.NewReader(data))
	if err != nil {
		return err
	}

	io.ReadAll(resp.Body)
	return nil
}

func (c *ComfyClient) EraseHistoryItem(promptID string) error {
	// delete post takes an array of IDs. We'll provide a single ID in a json array
	item := fmt.Sprintf("{\"delete\": [\"%s\"]}", promptID)
	resp, err := http.Post(fmt.Sprintf("http://%s/history", c.serverBaseAddress), "application/json", strings.NewReader(item))
	if err != nil {
		return err
	}

	io.ReadAll(resp.Body)
	return nil
}
