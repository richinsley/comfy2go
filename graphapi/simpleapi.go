package graphapi

type SimpleAPI struct {
	Properties map[string]Property
}

func (t *Graph) GetSimpleAPI() *SimpleAPI {
	group := t.GetGroupWithTitle("API")
	retv := &SimpleAPI{
		Properties: make(map[string]Property),
	}
	nodes := t.GetNodesInGroup(group)
	for _, n := range nodes {
		props := n.GetPropertesByIndex()
		if len(props) > 0 {
			retv.Properties[n.Title] = props[0]
		}
	}

	return retv
}
