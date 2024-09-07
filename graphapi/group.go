package graphapi

import (
	"fmt"
	"log/slog"
)

type Group struct {
	Title    string `json:"title"`
	Bounding []int  `json:"bounding"`
	Color    string `json:"color"`
}

func (r *Group) IntersectsOrContains(node *GraphNode) bool {
	if len(r.Bounding) != 4 {
		slog.Warn("Bounding box does not have exactly 4 elements")
		return false
	}

	// the geometry is stored differently for nodes and groups
	rx := float64(r.Bounding[0])
	ry := float64(r.Bounding[1])
	rw := float64(r.Bounding[2])
	rh := float64(r.Bounding[3])

	// The structure of the pos has changed with a newer version of ComfyUi
	var pos []interface{}
	switch v := node.Position.(type) {
	case []interface{}:
		pos = v
	case map[string]interface{}:
		pos = make([]interface{}, len(v))
		for i := 0; i < len(v); i++ {
			pos[i] = v[fmt.Sprintf("%d", i)]
		}
	default:
		slog.Warn("Node position is not of expected type []interface{} or map[string]interface{}", "type", fmt.Sprintf("%T", node.Position))
		return false
	}

	nx, ok := pos[0].(float64)
	if !ok {
		slog.Warn("Node position x is not of type float64")
		return false
	}
	ny, ok := pos[1].(float64)
	if !ok {
		slog.Warn("Node position y is not of type float64")
		return false
	}
	nw := node.Size.Width
	nh := node.Size.Height

	return !(rx > nx+nw ||
		rx+rw < nx ||
		ry > ny+nh ||
		ry+rh < ny)
}
