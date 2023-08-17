package graphapi

type Group struct {
	Title    string `json:"title"`
	Bounding []int  `json:"bounding"`
	Color    string `json:"color"`
}

func (r *Group) IntersectsOrContains(node *GraphNode) bool {
	// the geometry is stored differently for nodes and groups
	rx := float64(r.Bounding[0])
	ry := float64(r.Bounding[1])
	rw := float64(r.Bounding[2])
	rh := float64(r.Bounding[3])

	nx := node.Position.X
	ny := node.Position.Y
	nw := node.Size.Width
	nh := node.Size.Width

	return !(rx > nx+nw ||
		rx+rw < nx ||
		ry > ny+nh ||
		ry+rh < ny)
}
