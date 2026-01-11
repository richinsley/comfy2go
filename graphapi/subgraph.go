// graphapi/subgraph.go

package graphapi

import (
	"fmt"
	"strconv"
)

// SubgraphDefinition represents a reusable component defined in definitions.subgraphs
type SubgraphDefinition struct {
	ID         string                 `json:"id"`
	Version    int                    `json:"version"`
	State      SubgraphState          `json:"state"`
	Revision   int                    `json:"revision"`
	Config     map[string]interface{} `json:"config"`
	Name       string                 `json:"name"`
	InputNode  SubgraphIONode         `json:"inputNode"`
	OutputNode SubgraphIONode         `json:"outputNode"`
	Inputs     []SubgraphPort         `json:"inputs"`
	Outputs    []SubgraphPort         `json:"outputs"`
	Widgets    []interface{}          `json:"widgets"`
	Nodes      []*GraphNode           `json:"nodes"`
	Groups     []*Group               `json:"groups"`
	Links      []*Link                `json:"links"`
	Extra      map[string]interface{} `json:"extra,omitempty"`

	// Runtime maps (populated after unmarshal)
	NodesByID map[int]*GraphNode `json:"-"`
	LinksByID map[int]*Link      `json:"-"`

	// Reference back to parent graph for subgraph lookups
	ParentGraph *Graph `json:"-"`
}

type SubgraphState struct {
	LastGroupId   int `json:"lastGroupId"`
	LastNodeId    int `json:"lastNodeId"`
	LastLinkId    int `json:"lastLinkId"`
	LastRerouteId int `json:"lastRerouteId"`
}

type SubgraphIONode struct {
	ID       int       `json:"id"`
	Bounding []float64 `json:"bounding"`
}

type SubgraphPort struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Type          string    `json:"type"`
	LinkIds       []int     `json:"linkIds"`
	Pos           []float64 `json:"pos"`
	LocalizedName string    `json:"localized_name,omitempty"`
}

// GraphDefinitions holds the definitions section of a workflow
type GraphDefinitions struct {
	Subgraphs []*SubgraphDefinition `json:"subgraphs,omitempty"`
}

// BuildInternalMaps populates the runtime lookup maps for a subgraph
func (sg *SubgraphDefinition) BuildInternalMaps() {
	sg.NodesByID = make(map[int]*GraphNode)
	sg.LinksByID = make(map[int]*Link)

	for _, node := range sg.Nodes {
		sg.NodesByID[node.ID] = node
	}

	for _, link := range sg.Links {
		sg.LinksByID[link.ID] = link
	}
}

// GetNodeById returns a node from within the subgraph
func (sg *SubgraphDefinition) GetNodeById(id int) *GraphNode {
	return sg.NodesByID[id]
}

// GetLinkById returns a link from within the subgraph
func (sg *SubgraphDefinition) GetLinkById(id int) *Link {
	return sg.LinksByID[id]
}

// GetInputBySlot returns the input port at the given slot index
func (sg *SubgraphDefinition) GetInputBySlot(slot int) *SubgraphPort {
	if slot >= 0 && slot < len(sg.Inputs) {
		return &sg.Inputs[slot]
	}
	return nil
}

// GetOutputBySlot returns the output port at the given slot index
func (sg *SubgraphDefinition) GetOutputBySlot(slot int) *SubgraphPort {
	if slot >= 0 && slot < len(sg.Outputs) {
		return &sg.Outputs[slot]
	}
	return nil
}

// GetLinkToInput finds the internal link that connects to a subgraph input port
func (sg *SubgraphDefinition) GetLinkFromInput(inputSlot int) *Link {
	for _, link := range sg.Links {
		if link.OriginID == sg.InputNode.ID && link.OriginSlot == inputSlot {
			return link
		}
	}
	return nil
}

// GetLinkToOutput finds the internal link that connects to a subgraph output port
func (sg *SubgraphDefinition) GetLinkToOutput(outputSlot int) *Link {
	for _, link := range sg.Links {
		if link.TargetID == sg.OutputNode.ID && link.TargetSlot == outputSlot {
			return link
		}
	}
	return nil
}

// IsNodeSubgraph checks if an internal node is itself a subgraph instance
func (sg *SubgraphDefinition) IsNodeSubgraph(node *GraphNode) bool {
	if sg.ParentGraph == nil {
		return false
	}
	_, exists := sg.ParentGraph.SubgraphsByID[node.Type]
	return exists
}

// GetSubgraphForNode returns the SubgraphDefinition for a node if it's a subgraph instance
func (sg *SubgraphDefinition) GetSubgraphForNode(node *GraphNode) *SubgraphDefinition {
	if sg.ParentGraph == nil {
		return nil
	}
	return sg.ParentGraph.SubgraphsByID[node.Type]
}

// ExpandedNode represents a node after subgraph expansion with remapped IDs
type ExpandedNode struct {
	OriginalID    int
	ExpandedID    string              // Compound ID like "57:30" for subgraph internals, or "9" for top-level
	Node          *GraphNode
	SubgraphDef   *SubgraphDefinition // If this node is from a subgraph
	InstanceNode  *GraphNode          // The subgraph instance node in parent
	InputMapping  map[int]interface{} // Maps input slot -> external value or [nodeID, slot]
	OutputMapping map[int][]int       // Maps output slot -> [expandedNodeID, slot]
}

// SubgraphExpander handles recursive expansion of subgraphs for prompt generation
type SubgraphExpander struct {
	Graph            *Graph
	ExpandedNodes    map[string]*ExpandedNode // Keyed by string ID (compound for subgraphs)
	NextID           int
	OutputResolution map[string][]interface{} // Maps "instanceID:slot" -> [expandedNodeID (string), slot (int)]

	// Track ID mappings per subgraph instance: instanceNodeID -> (internalID -> compound string expandedID)
	InstanceIDMaps map[int]map[int]string
}

// NewSubgraphExpander creates a new expander for the given graph
func NewSubgraphExpander(g *Graph) *SubgraphExpander {
	return &SubgraphExpander{
		Graph:            g,
		ExpandedNodes:    make(map[string]*ExpandedNode),
		NextID:           g.LastNodeID + 1,
		OutputResolution: make(map[string][]interface{}),
		InstanceIDMaps:   make(map[int]map[int]string),
	}
}

// ExpandAll expands all nodes, recursively handling subgraphs
func (e *SubgraphExpander) ExpandAll() error {
	for _, node := range e.Graph.NodesInExecutionOrder {
		if node.IsVirtual() || node.Mode == 2 {
			continue
		}

		if node.IsSubgraph {
			if err := e.expandSubgraphNode(node, nil, nil); err != nil {
				return err
			}
		} else {
			// Regular node - just assign an expanded ID (same as original, as string)
			nodeIDStr := strconv.Itoa(node.ID)
			e.ExpandedNodes[nodeIDStr] = &ExpandedNode{
				OriginalID: node.ID,
				ExpandedID: nodeIDStr,
				Node:       node,
			}
		}
	}
	return nil
}

// expandSubgraphNode recursively expands a subgraph instance
func (e *SubgraphExpander) expandSubgraphNode(
	instanceNode *GraphNode,
	parentSubgraph *SubgraphDefinition,
	parentInputMapping map[int]interface{},
) error {
	sg := instanceNode.SubgraphDef
	if sg == nil {
		return fmt.Errorf("node %d has no subgraph definition", instanceNode.ID)
	}

	// Create ID mapping for this subgraph instance
	// Use compound IDs like "57:30" for internal nodes
	idMap := make(map[int]string)
	for _, internalNode := range sg.Nodes {
		// Create compound ID: "instanceID:internalID"
		compoundID := fmt.Sprintf("%d:%d", instanceNode.ID, internalNode.ID)
		idMap[internalNode.ID] = compoundID
	}

	// Store the ID map for this instance
	e.InstanceIDMaps[instanceNode.ID] = idMap

	// Build input mapping for this subgraph instance
	inputMapping := e.buildInputMapping(instanceNode, sg, parentSubgraph, parentInputMapping)

	// Process each internal node
	for _, internalNode := range sg.Nodes {
		// Skip virtual nodes - they shouldn't be in the prompt
		if internalNode.IsVirtual() {
			continue
		}

		// Skip muted nodes
		if internalNode.Mode == 2 {
			continue
		}

		expandedID := idMap[internalNode.ID]

		// Check if this internal node is itself a subgraph
		nestedSg := sg.GetSubgraphForNode(internalNode)
		if nestedSg != nil {
			internalNode.SubgraphDef = nestedSg
			internalNode.IsSubgraph = true

			nestedInputMapping := e.buildNestedInputMapping(internalNode, sg, idMap, inputMapping)

			if err := e.expandSubgraphNode(internalNode, sg, nestedInputMapping); err != nil {
				return err
			}
		} else {
			// Regular node inside subgraph
			expanded := &ExpandedNode{
				OriginalID:   internalNode.ID,
				ExpandedID:   expandedID,
				Node:         internalNode,
				SubgraphDef:  sg,
				InstanceNode: instanceNode,
				InputMapping: make(map[int]interface{}),
			}

			// Process all inputs for this node
			for i, slot := range internalNode.Inputs {
				if slot.Link == 0 {
					continue
				}
				link := sg.GetLinkById(slot.Link)
				if link == nil {
					continue
				}

				if link.OriginID == sg.InputNode.ID {
					// Connected to subgraph input - use input mapping
					if val, ok := inputMapping[link.OriginSlot]; ok {
						expanded.InputMapping[i] = val
					}
				} else {
					// Internal link - check if origin is a nested subgraph
					originNode := sg.GetNodeById(link.OriginID)
					if originNode != nil && originNode.IsSubgraph {
						// Resolve through nested subgraph's output
						key := fmt.Sprintf("%d:%d", originNode.ID, link.OriginSlot)
						if resolved, ok := e.OutputResolution[key]; ok {
							// resolved is now the string node ID
							expanded.InputMapping[i] = []interface{}{resolved, link.OriginSlot}
						}
					} else {
						// Regular internal link - remap to expanded IDs (now string compound IDs)
						expanded.InputMapping[i] = []interface{}{idMap[link.OriginID], link.OriginSlot}
					}
				}
			}

			e.ExpandedNodes[expandedID] = expanded
		}
	}

	// Register output mappings for this subgraph instance
	for outputSlot := range sg.Outputs {
		link := sg.GetLinkToOutput(outputSlot)
		if link != nil {
			key := fmt.Sprintf("%d:%d", instanceNode.ID, outputSlot)

			originNode := sg.GetNodeById(link.OriginID)
			if originNode != nil && originNode.IsSubgraph {
				// Origin is a nested subgraph
				nestedKey := fmt.Sprintf("%d:%d", originNode.ID, link.OriginSlot)
				if resolved, ok := e.OutputResolution[nestedKey]; ok {
					e.OutputResolution[key] = resolved
				}
			} else {
				e.OutputResolution[key] = []interface{}{idMap[link.OriginID], link.OriginSlot}
			}
		}
	}

	return nil
}

// buildInputMapping creates the input value mapping for a subgraph instance
func (e *SubgraphExpander) buildInputMapping(
	instanceNode *GraphNode,
	sg *SubgraphDefinition,
	parentSubgraph *SubgraphDefinition,
	parentInputMapping map[int]interface{},
) map[int]interface{} {
	mapping := make(map[int]interface{})

	for i, input := range sg.Inputs {
		// Check if there's an external link to this input
		var externalLink *Link

		if parentSubgraph != nil {
			// We're inside a parent subgraph - find link in parent's links
			for _, slot := range instanceNode.Inputs {
				if slot.Link != 0 {
					link := parentSubgraph.GetLinkById(slot.Link)
					if link != nil && link.TargetSlot == i {
						externalLink = link
						break
					}
				}
			}
		} else {
			// Top-level - find link in main graph
			for _, slot := range instanceNode.Inputs {
				if slot.Link != 0 {
					link := e.Graph.GetLinkById(slot.Link)
					if link != nil && link.TargetSlot == i {
						externalLink = link
						break
					}
				}
			}
		}

		if externalLink != nil {
			if parentSubgraph != nil && externalLink.OriginID == parentSubgraph.InputNode.ID {
				// Linked to parent subgraph's input - cascade from parent mapping
				if val, ok := parentInputMapping[externalLink.OriginSlot]; ok {
					mapping[i] = val
				}
			} else {
				// Linked to another node - resolve the actual node
				mapping[i] = e.resolveExternalLink(externalLink, parentSubgraph)
			}
		} else {
			// No external link - use widget value from instance node
			if instanceNode.WidgetValues != nil {
				mapping[i] = e.getWidgetValue(instanceNode, input.Name, i)
			}
		}
	}

	return mapping
}

// buildNestedInputMapping creates input mapping for a nested subgraph
func (e *SubgraphExpander) buildNestedInputMapping(
	nestedNode *GraphNode,
	parentSg *SubgraphDefinition,
	parentIdMap map[int]string,
	parentInputMapping map[int]interface{},
) map[int]interface{} {
	nestedSg := nestedNode.SubgraphDef
	mapping := make(map[int]interface{})

	for i := range nestedSg.Inputs {
		// Find the link to this input in the parent subgraph
		for _, slot := range nestedNode.Inputs {
			if slot.Link == 0 {
				continue
			}
			link := parentSg.GetLinkById(slot.Link)
			if link == nil || link.TargetSlot != i {
				continue
			}

			if link.OriginID == parentSg.InputNode.ID {
				// Connected to parent's input - cascade
				if val, ok := parentInputMapping[link.OriginSlot]; ok {
					mapping[i] = val
				}
			} else {
				// Connected to sibling node in parent - parentIdMap now returns string IDs
				mapping[i] = []interface{}{parentIdMap[link.OriginID], link.OriginSlot}
			}
			break
		}
	}

	return mapping
}

// resolveExternalLink resolves a link to its expanded node reference
func (e *SubgraphExpander) resolveExternalLink(link *Link, parentSubgraph *SubgraphDefinition) interface{} {
	var originNode *GraphNode

	if parentSubgraph != nil {
		originNode = parentSubgraph.GetNodeById(link.OriginID)
	} else {
		originNode = e.Graph.GetNodeById(link.OriginID)
	}

	if originNode != nil && originNode.IsSubgraph {
		// Origin is a subgraph - need to resolve to actual output
		key := fmt.Sprintf("%d:%d", link.OriginID, link.OriginSlot)
		if resolved, ok := e.OutputResolution[key]; ok {
			return resolved
		}
	}

	// Regular node - use original ID (will be same as expanded for top-level nodes)
	return []int{link.OriginID, link.OriginSlot}
}

// getWidgetValue extracts a widget value from a node
func (e *SubgraphExpander) getWidgetValue(node *GraphNode, name string, index int) interface{} {
	// First, try to get value from properties if they exist
	if node.Properties != nil && len(node.Properties) > 0 {
		prop := node.GetPropertyWithName(name)
		if prop != nil {
			return prop.GetValue()
		}
	}

	// Fall back to raw widget values
	if node.IsWidgetValueArray() {
		arr := node.WidgetValuesArray()
		if index < len(arr) {
			return arr[index]
		}
	} else if node.IsWidgetValueMap() {
		m := node.WidgetValuesMap()
		if val, ok := m[name]; ok {
			return val
		}
	}
	return nil
}

// ToPromptNodes converts all expanded nodes to prompt format
func (e *SubgraphExpander) ToPromptNodes() map[string]PromptNode {
	result := make(map[string]PromptNode)

	for expandedID, expanded := range e.ExpandedNodes {
		node := expanded.Node

		pn := PromptNode{
			ClassType: node.Type,
			Inputs:    make(map[string]interface{}),
		}

		// Add widget values from properties if available, otherwise use raw widget_values
		if node.Properties != nil && len(node.Properties) > 0 {
			// Properties have been created via CreateNodeProperties
			for k, prop := range node.Properties {
				if prop.Serializable() {
					pn.Inputs[k] = prop.GetValue()
				}
			}
		} else {
			// Properties not created - use raw widget_values
			// This is typical when GraphToPrompt is called without CreateNodeProperties
			// Widget values will be added via input mapping or can be extracted from WidgetValues if needed
		}

		// Handle inputs - either from subgraph mapping or direct links
		if expanded.SubgraphDef != nil {
			// Node from inside a subgraph
			for i, slot := range node.Inputs {
				if val, ok := expanded.InputMapping[i]; ok {
					switch v := val.(type) {
					case []int:
						// Link reference [nodeID, slot]
						linfo := make([]interface{}, 2)
						linfo[0] = strconv.Itoa(v[0])
						linfo[1] = v[1]
						pn.Inputs[slot.Name] = linfo
					default:
						// Direct value - this overrides widget value
						pn.Inputs[slot.Name] = val
					}
				} else if slot.Link != 0 {
					// Internal link not from subgraph input
					link := expanded.SubgraphDef.GetLinkById(slot.Link)
					if link != nil && link.OriginID != expanded.SubgraphDef.InputNode.ID {
						// Need to find the expanded ID for the origin node
						originExpandedID := e.findExpandedID(expanded.SubgraphDef, expanded.InstanceNode, link.OriginID)
						if originExpandedID != "" {
							linfo := make([]interface{}, 2)
							linfo[0] = originExpandedID // Already a string
							linfo[1] = link.OriginSlot
							pn.Inputs[slot.Name] = linfo
						}
					}
				}
			}
		} else {
			// Top-level node - process links normally
			for _, slot := range node.Inputs {
				if slot.Link == 0 {
					continue
				}
				link := e.Graph.GetLinkById(slot.Link)
				if link == nil {
					continue
				}

				originNode := e.Graph.GetNodeById(link.OriginID)
				var resolvedIDStr string
				var resolvedSlot int

				if originNode != nil && originNode.IsSubgraph {
					key := fmt.Sprintf("%d:%d", link.OriginID, link.OriginSlot)
					if resolved, ok := e.OutputResolution[key]; ok {
						// resolved is []interface{}{nodeIDString, slot}
						resolvedIDStr = resolved[0].(string)
						resolvedSlot = resolved[1].(int)
					} else {
						continue
					}
				} else {
					resolvedIDStr = strconv.Itoa(link.OriginID)
					resolvedSlot = link.OriginSlot
				}

				linfo := make([]interface{}, 2)
				linfo[0] = resolvedIDStr
				linfo[1] = resolvedSlot
				pn.Inputs[slot.Name] = linfo
			}
		}

		result[expandedID] = pn
	}

	return result
}

// findExpandedID finds the expanded ID for an internal node within a subgraph instance
func (e *SubgraphExpander) findExpandedID(sg *SubgraphDefinition, instanceNode *GraphNode, internalNodeID int) string {
	// Look through expanded nodes to find one that matches
	for expandedID, expanded := range e.ExpandedNodes {
		if expanded.SubgraphDef == sg &&
			expanded.InstanceNode == instanceNode &&
			expanded.OriginalID == internalNodeID {
			return expandedID
		}
	}
	return ""
}
