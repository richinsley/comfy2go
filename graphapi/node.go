package graphapi

import (
	"log/slog"
)

// GraphNode represents the encapsulation of an individual functionality within a Graph
type GraphNode struct {
	ID                 int                     `json:"id"`
	Type               string                  `json:"type"`
	Position           interface{}             `json:"pos"`
	Size               Size                    `json:"size"`
	Flags              *interface{}            `json:"flags"`
	Order              int                     `json:"order"`
	Mode               int                     `json:"mode"`
	Title              string                  `json:"title"`
	InternalProperties *map[string]interface{} `json:"properties"` // node properties, not value properties!
	// widgets_values can be an array of values, or a map of values
	// maps of values can represent cascading style properties in which the setting
	// of one property makes certain other properties available
	WidgetValues interface{}         `json:"widgets_values"`
	Color        string              `json:"color"`
	BGColor      string              `json:"bgcolor"`
	Inputs       []Slot              `json:"inputs,omitempty"`
	Outputs      []Slot              `json:"outputs,omitempty"`
	Graph        *Graph              `json:"-"`
	CustomData   *interface{}        `json:"-"`
	Widgets      []*Widget           `json:"-"`
	Properties   map[string]Property `json:"-"`
	DisplayName  string              `json:"-"`
	Description  string              `json:"-"`
	IsOutput     bool                `json:"-"`
}

func (n *GraphNode) WidgetValuesArray() []interface{} {
	if n.WidgetValues == nil {
		return nil
	}
	retv, ok := n.WidgetValues.([]interface{})
	if ok {
		return retv
	}
	return nil
}

func (n *GraphNode) WidgetValuesMap() map[string]interface{} {
	if n.WidgetValues == nil {
		return nil
	}
	retv, ok := n.WidgetValues.(map[string]interface{})
	if ok {
		return retv
	}
	return nil
}

func (n *GraphNode) IsWidgetValueArray() bool {
	return n.WidgetValuesArray() != nil
}

func (n *GraphNode) IsWidgetValueMap() bool {
	return n.WidgetValuesMap() != nil
}

func (n *GraphNode) WidgetValueCount() int {
	if n.IsWidgetValueArray() {
		return len(n.WidgetValuesArray())
	}
	if n.IsWidgetValueMap() {
		return len(n.WidgetValuesMap())
	}
	return 0
}

func (n *GraphNode) IsVirtual() bool {
	// current nodes that are 'virtual':
	switch n.Type {
	case "PrimitiveNode":
		return true
	case "Reroute":
		return true
	case "Note":
		return true
	}
	return false
}

// GetLinks returns a slice of Link Ids
func (n *GraphNode) GetLinks() []int {
	retv := make([]int, 0)
	for _, l := range *n.Outputs[0].Links {
		linkInfo := n.Graph.LinksByID[l]
		tn := n.Graph.GetNodeById(linkInfo.TargetID)
		if tn.Type == "Rerout" {
			retv = append(retv, tn.GetLinks()...)
		} else {
			retv = append(retv, l)
		}
	}
	return retv
}

func (n *GraphNode) GetPropertyWithName(name string) Property {
	retv, ok := n.Properties[name]
	if ok {
		return retv
	}

	// check n.Properties for an aliased property
	for _, p := range n.Properties {
		if p.GetAlias() == name {
			return p
		}
	}

	return nil
}

// GetPropertesByIndex returns a slice of Properties ordered by thier order in the node desciption
// Because a properties index is for it's index in the node description, and not the index of the property in the node's properties,
// non-indexed properties will be nil in the returned slice
func (n *GraphNode) GetPropertiesByIndex() []Property {
	// Initialize with zero length but with an initial capacity.
	temp := make([]Property, 0, len(n.Properties))
	lastindex := -1
	for _, v := range n.Properties {
		index := v.Index()
		if index >= 0 {
			lastindex = index
		} else {
			// Handle non-indexed properties, like IMAGEUPLOAD.
			if v.TypeString() != "IMAGEUPLOAD" {
				slog.Warn("Property with unknown target index of type", "type", v.TypeString())
			}
			lastindex++
		}

		// Ensure the slice is large enough by checking if we need to resize.
		for len(temp) <= lastindex {
			temp = append(temp, nil) // Append nil for interface type to increase slice size.
		}

		temp[lastindex] = v
	}

	// compact the slice to remove nils
	retv := make([]Property, 0, len(temp))
	for _, v := range temp {
		if v != nil {
			retv = append(retv, v)
		}
	}
	return retv
}

func (n *GraphNode) GetNodeForInput(slotIndex int) *GraphNode {
	if slotIndex >= len(n.Inputs) {
		return nil
	}

	slot := n.Inputs[slotIndex]
	l := n.Graph.GetLinkById(slot.Link)
	if l == nil {
		return nil
	}
	return n.Graph.GetNodeById(l.OriginID)
}

func (n *GraphNode) GetInputLink(slotIndex int) *Link {
	ncount := len(n.Inputs)
	if ncount == 0 || slotIndex >= ncount {
		return nil
	}

	slot := n.Inputs[slotIndex]
	return n.Graph.GetLinkById(slot.Link)
}

func (n *GraphNode) GetInputWithName(name string) *Slot {
	for i, s := range n.Inputs {
		if s.Name == name {
			return &n.Inputs[i]
		}
	}
	return nil
}

func (n *GraphNode) affixPropertyToInputSlot(name string, p Property) {
	slot := n.GetInputWithName(name)
	if slot != nil {
		slot.Property = p
	}
}

func (n *GraphNode) ApplyToGraph() {
	// only PrimitiveNode need apply
	if n.Type != "PrimitiveNode" {
		return
	}

	if n.Outputs[0].Links == nil || len(*n.Outputs[0].Links) == 0 {
		return
	}

	links := n.GetLinks()
	// For each output link copy our value over the original widget value
	for _, l := range links {
		linkinfo := n.Graph.LinksByID[l]
		node := n.Graph.GetNodeById(linkinfo.TargetID)
		input := node.Inputs[linkinfo.TargetSlot]
		widgetName := input.Widget.Name
		if widgetName != nil {
			// Nodes need a distinct Widget class
			// widget.value = this.widgets[0].value;
			// fmt.Print()
		}

		/*
			widgetName := input.Widget.
			const widgetName = input.widget.name;
			if (widgetName) {
				const widget = node.widgets.find((w) => w.name === widgetName);
				if (widget) {
					widget.value = this.widgets[0].value;
					if (widget.callback) {
						widget.callback(widget.value, app.canvas, node, app.canvas.graph_mouse, {});
					}
				}
			}
		*/
	}
}
