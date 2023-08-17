package graphapi

// Widget represents the input points for setting properties within a Node
type Widget struct {
	Name   *string      `json:"name"`
	Config *interface{} `json:"config"`
}
