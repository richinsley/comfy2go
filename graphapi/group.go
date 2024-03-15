package graphapi

type Group struct {
	Title    string `json:"title"`
	Bounding []int  `json:"bounding"`
	Color    string `json:"color"`
}

func (r *Group) IntersectsOrContains(node *GraphNode) bool {
	if len(r.Bounding) != 4 {
		return false
	}

	// the geometry is stored differently for nodes and groups
	rx := float64(r.Bounding[0])
	ry := float64(r.Bounding[1])
	rw := float64(r.Bounding[2])
	rh := float64(r.Bounding[3])

	pos, ok := node.Position.([]interface{})
	if !ok {
		return false
	}

	nx := pos[0].(float64)
	ny := pos[1].(float64)
	nw := node.Size.Width
	nh := node.Size.Width

	return !(rx > nx+nw ||
		rx+rw < nx ||
		ry > ny+nh ||
		ry+rh < ny)
}
