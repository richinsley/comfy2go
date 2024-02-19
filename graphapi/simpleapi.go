package graphapi

type SimpleAPI struct {
	Properties map[string]Property
}

func getImageUploader(props []Property) Property {
	for _, p := range props {
		if p.TypeString() == "IMAGEUPLOAD" {
			return p
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
	retv := &SimpleAPI{
		Properties: make(map[string]Property),
	}
	nodes := t.GetNodesInGroup(group)
	for _, n := range nodes {
		props := n.GetPropertesByIndex()
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
