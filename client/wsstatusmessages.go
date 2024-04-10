package client

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
)

type WSStatusMessage struct {
	Type string      `json:"type"`
	Data interface{} `json:"Data"`
}

func (sm *WSStatusMessage) UnmarshalJSON(b []byte) error {
	// Unmarshal into an anonymous type equivalent to StatusMessage
	// to avoid infinite recursion
	var temp struct {
		Type string          `json:"type"`
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(b, &temp); err != nil {
		return err
	}

	sm.Type = temp.Type

	// Determine the type of Data and unmarshal it accordingly
	switch sm.Type {
	case "status":
		sm.Data = &WSMessageDataStatus{}
	case "execution_start":
		sm.Data = &WSMessageDataExecutionStart{}
	case "execution_cached":
		sm.Data = &WSMessageDataExecutionCached{}
	case "executing":
		sm.Data = &WSMessageDataExecuting{}
	case "progress":
		sm.Data = &WSMessageDataProgress{}
	case "executed":
		// this is a special case because the data type is not always the same
		// so we need to unmarshal it manually
		sm.Data = &WSMessageDataExecuted{}
	case "execution_interrupted":
		sm.Data = &WSMessageExecutionInterrupted{}
	case "execution_error":
		sm.Data = &WSMessageExecutionError{}
	default:
		// Handle unknown data types or return a dedicated error here
		sm.Data = nil
	}

	if sm.Data != nil {
		// Unmarshal the data into the selected type
		if err := json.Unmarshal(temp.Data, sm.Data); err != nil {
			return err
		}
	}

	return nil
}

type WSMessageDataStatus struct {
	Status struct {
		ExecInfo struct {
			QueueRemaining int `json:"queue_remaining"`
		} `json:"exec_info"`
	} `json:"status"`
}

/*
{"type": "status", "data": {"status": {"exec_info": {"queue_remaining": 1}}}}
*/

type WSMessageDataExecutionStart struct {
	PromptID string `json:"prompt_id"`
}

/*
{"type": "execution_start", "data": {"prompt_id": "ed986d60-2a27-4d28-8871-2fdb36582902"}}
*/

type WSMessageDataExecutionCached struct {
	Nodes    []interface{} `json:"nodes"`
	PromptID string        `json:"prompt_id"`
}

/*
{"type": "execution_cached", "data": {"nodes": [], "prompt_id": "ed986d60-2a27-4d28-8871-2fdb36582902"}}
*/

type WSMessageDataExecuting struct {
	Node     *int   `json:"node"`
	PromptID string `json:"prompt_id"`
}

func (mde *WSMessageDataExecuting) UnmarshalJSON(b []byte) error {
	var temp struct {
		Node     *string `json:"node"`
		PromptID string  `json:"prompt_id"`
	}
	if err := json.Unmarshal(b, &temp); err != nil {
		return err
	}

	mde.PromptID = temp.PromptID

	// Convert string to int
	if temp.Node != nil {
		i, err := strconv.Atoi(*temp.Node)
		if err != nil {
			return err
		}
		mde.Node = &i
	} else {
		mde.Node = nil
	}

	return nil
}

/*
{"type": "executing", "data": {"node": "12", "prompt_id": "ed986d60-2a27-4d28-8871-2fdb36582902"}}
*/

type WSMessageDataProgress struct {
	Value int `json:"value"`
	Max   int `json:"max"`
}

/*
{"type": "progress", "data": {"value": 1, "max": 20}}
*/

type WSMessageDataExecuted struct {
	Node     int                      `json:"node"`
	Output   map[string]*[]DataOutput `json:"output"`
	PromptID string                   `json:"prompt_id"`
}

func (mde *WSMessageDataExecuted) UnmarshalJSON(b []byte) error {
	var temp struct {
		Node      string                 `json:"node"`
		OutputRaw map[string]interface{} `json:"output"`
		PromptID  string                 `json:"prompt_id"`
	}
	if err := json.Unmarshal(b, &temp); err != nil {
		return err
	}

	// iterrate over Outputraw and see if it can be cast to a slice of interface{}
	mde.Output = make(map[string]*[]DataOutput)
	for k, v := range temp.OutputRaw {
		if val, ok := v.([]interface{}); ok {
			mde.Output[k] = &[]DataOutput{}
			for _, i := range val {
				if outmap, ok := i.(map[string]interface{}); ok {
					// ensure the output map has the required fields
					outputentry := DataOutput{}
					val, ok := outmap["filename"]
					if ok {
						outputentry.Filename = val.(string)
					} else {
						slog.Warn(fmt.Sprintf("WSMessageDataExecuted output entry %v unknown type", i))
						continue
					}

					val, ok = outmap["subfolder"]
					if ok {
						outputentry.Subfolder = val.(string)
					} else {
						// we can ignore this if it's absent
						outputentry.Subfolder = ""
					}

					val, ok = outmap["type"]
					if ok {
						outputentry.Type = val.(string)
					} else {
						slog.Warn(fmt.Sprintf("WSMessageDataExecuted output entry %v unknown type", i))
						continue
					}

					*mde.Output[k] = append(*mde.Output[k], outputentry)
				} else if outstring, ok := i.(string); ok {
					// handle raw text output
					textout := DataOutput{
						Filename:  "",
						Subfolder: "",
						Type:      "text",
						Text:      outstring,
					}
					*mde.Output[k] = append(*mde.Output[k], textout)
				} else {
					slog.Warn(fmt.Sprintf("WSMessageDataExecuted output entry %v unknown type", i))
					// create an "unknown" type
					// convert i to a string and store it as text
					outstring := i.(string)
					textout := DataOutput{
						Filename:  "",
						Subfolder: "",
						Type:      "unknown",
						Text:      outstring,
					}
					*mde.Output[k] = append(*mde.Output[k], textout)
				}
			}
		}

	}

	mde.PromptID = temp.PromptID

	// Convert string to int
	i, err := strconv.Atoi(temp.Node)
	if err != nil {
		return err
	}
	mde.Node = i

	return nil
}

/*
{"type": "executed", "data": {"node": "19", "output": {"images": [{"filename": "ComfyUI_00046_.png", "subfolder": "", "type": "output"}]}, "prompt_id": "ed986d60-2a27-4d28-8871-2fdb36582902"}}

// when there are multiple outputs, each output will receive an "executed"
{"type": "executed", "data": {"node": "53", "output": {"images": [{"filename": "ComfyUI_temp_mynbi_00001_.png", "subfolder": "", "type": "temp"}]}, "prompt_id": "3bcf5bac-19e1-4219-a0eb-50a84e4db2ea"}}
{"type": "executed", "data": {"node": "19", "output": {"images": [{"filename": "ComfyUI_00052_.png", "subfolder": "", "type": "output"}]}, "prompt_id": "3bcf5bac-19e1-4219-a0eb-50a84e4db2ea"}}
*/

type WSMessageExecutionInterrupted struct {
	PromptID string   `json:"prompt_id"`
	Node     string   `json:"node_id"`
	NodeType string   `json:"node_type"`
	Executed []string `json:"executed"`
}

/*
{"type": "execution_interrupted", "data": {"prompt_id": "dc7093d7-980a-4fe6-bf0c-f6fef932c74b", "node_id": "19", "node_type": "SaveImage", "executed": ["5", "17", "10", "11"]}}
*/

type WSMessageExecutionError struct {
	PromptID         string                 `json:"prompt_id"`
	Node             string                 `json:"node_id"`
	NodeType         string                 `json:"node_type"`
	Executed         []string               `json:"executed"`
	ExceptionMessage string                 `json:"exception_message"`
	ExceptionType    string                 `json:"exception_type"`
	Traceback        []string               `json:"traceback"`
	CurrentInputs    map[string]interface{} `json:"current_inputs"`
	CurrentOutputs   map[int]interface{}    `json:"current_outputs"`
}
