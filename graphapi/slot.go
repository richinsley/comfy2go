package graphapi

// Slot represents a connection point within a GraphNode.
// It holds various properties that define the behavior and appearance
// of the connection, such as the name, type, associated widget, and more.
type Slot struct {
	Name       string     `json:"name"` // The name of the slot
	CustomType int        `json:"-"`
	Node       *GraphNode `json:"-"`                // The node the slot belongs to
	Type       string     `json:"type"`             // The type of the data the slot accepts
	Link       int        `json:"link,omitempty"`   // Index of the link for an input slot
	Links      *[]int     `json:"links,omitempty"`  // Array of links for output slots
	Widget     *Widget    `json:"widget,omitempty"` // Collection of widgets that allow setting properties
	Shape      *int       `json:"shape,omitempty"`
	SlotIndex  *int       `json:"slot_index,omitempty"` // Index of the Slot in relation to other Slots
	Property   Property   `json:"-"`                    // non-null for inputs that are exported widgets
}
