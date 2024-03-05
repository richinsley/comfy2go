package graphapi

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
)

type NodeObjects struct {
	Objects map[string]*NodeObject
}

// NodeObject represents the metadata that describes how to generate an instance of a node for a graph.
type NodeObject struct {
	Input               *NodeObjectInput     `json:"input"`
	Output              *[]interface{}       `json:"output"` // output type
	OutputIsList        *[]bool              `json:"output_is_list"`
	OutputName          *interface{}         `json:"output_name"`
	Name                string               `json:"name"`
	DisplayName         string               `json:"display_name"`
	Description         string               `json:"description"`
	Category            string               `json:"category"`
	OutputNode          bool                 `json:"output_node"`
	InputProperties     []*Property          `json:"-"`
	InputPropertiesByID map[string]*Property `json:"-"`
}

// GetSettablePropertiesByID returns a map of Properties that are settable.
func (n *NodeObject) GetSettablePropertiesByID() map[string]Property {
	retv := make(map[string]Property)
	for k, p := range n.InputPropertiesByID {
		if (*p).Settable() {
			retv[k] = *p
		}
	}
	return retv
}

// GetSettablePropertiesByID returns a slice of Properties that are settable.
func (n *NodeObject) GetSettableProperties() []Property {
	retv := make([]Property, 0)
	for _, p := range n.InputProperties {
		if (*p).Settable() {
			retv = append(retv, *p)
		}
	}
	return retv
}

type NodeObjectInput struct {
	Required        map[string]*interface{} `json:"required"`
	Optional        map[string]*interface{} `json:"optional,omitempty"`
	OrderedRequired []string                `json:"-"`
	OrderedOptional []string                `json:"-"`
}

// NodeObjectInput custom UnmarshalJSON deserializtion maintains the order of the properties in the JSON
func (noi *NodeObjectInput) UnmarshalJSON(b []byte) error {
	dec := json.NewDecoder(strings.NewReader(string(b)))
	dec.UseNumber()

	if _, err := dec.Token(); err != nil {
		return err
	} // consume opening brace

	for dec.More() {
		t, err := dec.Token()
		if err != nil {
			return err
		}

		key := t.(string)
		switch key {
		case "required", "optional":
			if _, err := dec.Token(); err != nil { // consume opening brace of nested object
				return err
			}

			currentMap := make(map[string]*interface{})
			currentOrder := make([]string, 0)
			for dec.More() {
				entryKeyToken, err := dec.Token()
				if err != nil {
					return err
				}

				entryKey := entryKeyToken.(string)
				currentOrder = append(currentOrder, entryKey)

				rawValue := &json.RawMessage{}
				if err := dec.Decode(rawValue); err != nil {
					return err
				}

				var i interface{}
				if err := json.Unmarshal(*rawValue, &i); err != nil {
					return err
				}

				currentMap[entryKey] = &i
			}

			if _, err := dec.Token(); err != nil { // consume closing brace of nested object
				return err
			}

			if key == "required" {
				noi.Required = currentMap
				noi.OrderedRequired = currentOrder
			} else if key == "optional" {
				noi.Optional = currentMap
				noi.OrderedOptional = currentOrder
			}
		default:
			// consume and ignore non-expected field (typically 'hidden' fields)
			if err := dec.Decode(new(interface{})); err != nil {
				return err
			}
		}
	}

	if _, err := dec.Token(); err != nil { // consume closing brace
		return err
	}

	return nil
}

var control_after_generate_text string = `
[
	[
		"fixed",
		"increment",
		"decrement",
		"randomize"
	]
]
`

func (n *NodeObjects) PopulateInputProperties() {
	var cdata []interface{}
	json.Unmarshal([]byte(control_after_generate_text), &cdata)
	var car interface{} = cdata

	for _, o := range n.Objects {
		o.InputPropertiesByID = make(map[string]*Property)
		o.InputProperties = make([]*Property, 0)
		index := int(0)

		for _, k := range o.Input.OrderedRequired {
			p := o.Input.Required[k]
			nprop := NewPropertyFromInput(k, false, p, index)
			index++
			if nprop != nil {
				o.InputProperties = append(o.InputProperties, nprop)
				o.InputPropertiesByID[k] = nprop
			} else {
				slog.Error(fmt.Sprintf("Cannot create property %s for object %s", k, o.Name))
				continue
			}

			// handle seed and noise_seed int controls
			if ((*nprop).Name() == "seed" || (*nprop).Name() == "noise_seed") && (*nprop).TypeString() == "INT" {
				ns_prop := NewPropertyFromInput("control_after_generate", (*nprop).Optional(), &car, index)
				index++
				(*ns_prop).SetSerializable(false)
				o.InputProperties = append(o.InputProperties, ns_prop)
				o.InputPropertiesByID["control_after_generate"] = ns_prop
			}
		}

		if o.Input.Optional != nil {
			for _, k := range o.Input.OrderedOptional {
				p := o.Input.Optional[k]
				nprop := NewPropertyFromInput(k, true, p, index)
				index++
				if nprop != nil {
					o.InputProperties = append(o.InputProperties, nprop)
					o.InputPropertiesByID[k] = nprop
				} else {
					slog.Error(fmt.Sprintf("Cannot create property %s for object %s", k, o.Name))
					continue
				}

				// handle seed and noise_seed int controls
				if (*nprop).Name() == "seed" || (*nprop).Name() == "noise_seed" && (*nprop).TypeString() == "INT" {
					ns_prop := NewPropertyFromInput("control_after_generate", (*nprop).Optional(), &car, index)
					index++
					o.InputProperties = append(o.InputProperties, ns_prop)
					o.InputPropertiesByID["control_after_generate"] = ns_prop
				}
			}
		}
	}
}

func (n *NodeObjects) GetNodeObjectByName(name string) *NodeObject {
	val, ok := n.Objects[name]
	if ok {
		return val
	}
	return nil
}
