package graphapi

type SimpleAPI struct {
	Properties  map[string]Property
	OutputNodes []*GraphNode
}

func getImageUploader(props []Property) Property {
	for _, p := range props {
		if p != nil {
			if p.TypeString() == "IMAGEUPLOAD" {
				return p
			}
		}
	}
	return nil
}

// GetSimpleAPI returns an instance of SimpleAPI
// SimpleAPI is a collection of nodes in the graph that are contained within a group with the given title.
// When title is nil, the default "API" group will be used
func (t *Graph) GetSimpleAPI(title *string) *SimpleAPI {
	if title == nil {
		defaultAPI := "API"
		title = &defaultAPI
	}
	group := t.GetGroupWithTitle(*title)
	if group == nil {
		return nil
	}
	retv := &SimpleAPI{
		Properties: make(map[string]Property),
	}
	nodes := t.GetNodesInGroup(group)
	for _, n := range nodes {
		// is the node an output node?  Get the *graphapi.NodeObjects for the node
		if n.IsOutput {
			retv.OutputNodes = append(retv.OutputNodes, n)
		}

		props := n.GetPropertiesByIndex()
		if len(props) > 0 {
			// if a node has an image uploader property, we want that one
			uploader := getImageUploader(props)
			if uploader != nil {
				retv.Properties[n.Title] = uploader
			} else {
				// otherwise, take the first property in the node
				retv.Properties[n.Title] = props[0]
			}
		}
	}

	return retv
}
