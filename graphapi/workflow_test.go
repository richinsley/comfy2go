package graphapi

import (
	"encoding/json"
	"os"
	"testing"
)

// TestRoundtripSubgraphWorkflow tests that we can deserialize and re-serialize
// a workflow with subgraphs and get equivalent JSON output
func TestRoundtripSubgraphWorkflow(t *testing.T) {
	// Read the test workflow file
	data, err := os.ReadFile("../examples/testdata/zimage-subgraph.json")
	if err != nil {
		t.Fatalf("Failed to read test file: %v", err)
	}

	// Deserialize into Graph
	var graph Graph
	err = json.Unmarshal(data, &graph)
	if err != nil {
		t.Fatalf("Failed to unmarshal graph: %v", err)
	}

	// Verify basic structure was loaded
	if len(graph.Nodes) == 0 {
		t.Error("Expected nodes to be loaded")
	}
	if len(graph.Links) == 0 {
		t.Error("Expected links to be loaded")
	}
	if graph.Definitions == nil {
		t.Fatal("Expected definitions to be loaded")
	}
	if graph.Definitions.Subgraphs == nil || len(graph.Definitions.Subgraphs) == 0 {
		t.Fatal("Expected subgraphs to be loaded")
	}

	// Verify subgraph was loaded correctly
	sg := graph.Definitions.Subgraphs[0]
	if sg.ID != "f2fdebf6-dfaf-43b6-9eb2-7f70613cfdc1" {
		t.Errorf("Expected subgraph ID to be f2fdebf6-dfaf-43b6-9eb2-7f70613cfdc1, got %s", sg.ID)
	}
	if sg.Name != "Text to Image (Z-Image-Turbo)" {
		t.Errorf("Expected subgraph name to be 'Text to Image (Z-Image-Turbo)', got %s", sg.Name)
	}
	if len(sg.Nodes) == 0 {
		t.Error("Expected subgraph to have nodes")
	}
	if len(sg.Links) == 0 {
		t.Error("Expected subgraph to have links")
	}

	// Verify node references subgraph
	var subgraphNode *GraphNode
	for _, node := range graph.Nodes {
		if node.Type == "f2fdebf6-dfaf-43b6-9eb2-7f70613cfdc1" {
			subgraphNode = node
			break
		}
	}
	if subgraphNode == nil {
		t.Fatal("Expected to find a node with subgraph type")
	}
	if !subgraphNode.IsSubgraph {
		t.Error("Expected node to be marked as subgraph")
	}
	if subgraphNode.SubgraphDef == nil {
		t.Error("Expected node to have subgraph definition reference")
	}

	// Verify links in subgraph have object format flag set
	for i, link := range sg.Links {
		if !link.isObjectFormat {
			t.Errorf("Subgraph link %d should be in object format", i)
		}
	}

	// Verify top-level links have tuple format flag set
	for i, link := range graph.Links {
		if link.isObjectFormat {
			t.Errorf("Top-level link %d should be in tuple format", i)
		}
	}

	// Re-serialize to JSON
	output, err := json.MarshalIndent(&graph, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal graph: %v", err)
	}

	// Deserialize both original and output to compare structure
	var original map[string]interface{}
	err = json.Unmarshal(data, &original)
	if err != nil {
		t.Fatalf("Failed to unmarshal original data: %v", err)
	}

	var result map[string]interface{}
	err = json.Unmarshal(output, &result)
	if err != nil {
		t.Fatalf("Failed to unmarshal output data: %v", err)
	}

	// Verify key fields match
	compareField(t, "nodes count", len(original["nodes"].([]interface{})), len(result["nodes"].([]interface{})))
	compareField(t, "links count", len(original["links"].([]interface{})), len(result["links"].([]interface{})))

	// Verify definitions exist
	if result["definitions"] == nil {
		t.Fatal("Output missing definitions section")
	}

	origDefs := original["definitions"].(map[string]interface{})
	resultDefs := result["definitions"].(map[string]interface{})

	origSubgraphs := origDefs["subgraphs"].([]interface{})
	resultSubgraphs := resultDefs["subgraphs"].([]interface{})

	compareField(t, "subgraphs count", len(origSubgraphs), len(resultSubgraphs))

	// Verify first subgraph structure
	origSg := origSubgraphs[0].(map[string]interface{})
	resultSg := resultSubgraphs[0].(map[string]interface{})

	compareField(t, "subgraph ID", origSg["id"], resultSg["id"])
	compareField(t, "subgraph name", origSg["name"], resultSg["name"])
	compareField(t, "subgraph nodes count", len(origSg["nodes"].([]interface{})), len(resultSg["nodes"].([]interface{})))
	compareField(t, "subgraph links count", len(origSg["links"].([]interface{})), len(resultSg["links"].([]interface{})))

	// Verify subgraph links are in object format
	origLinks := origSg["links"].([]interface{})
	resultLinks := resultSg["links"].([]interface{})

	for i := 0; i < len(origLinks); i++ {
		origLink := origLinks[i].(map[string]interface{})
		resultLink := resultLinks[i].(map[string]interface{})

		if origLink["id"] != resultLink["id"] {
			t.Errorf("Subgraph link %d: expected id %v, got %v", i, origLink["id"], resultLink["id"])
		}
		if origLink["origin_id"] != resultLink["origin_id"] {
			t.Errorf("Subgraph link %d: expected origin_id %v, got %v", i, origLink["origin_id"], resultLink["origin_id"])
		}
	}

	// Verify top-level links are in tuple format
	origTopLinks := original["links"].([]interface{})
	resultTopLinks := result["links"].([]interface{})

	for i := 0; i < len(origTopLinks); i++ {
		origLink := origTopLinks[i].([]interface{})
		resultLink := resultTopLinks[i].([]interface{})

		if len(origLink) != 6 || len(resultLink) != 6 {
			t.Errorf("Top-level link %d: expected tuple format with 6 elements", i)
		}

		// Compare link IDs
		if origLink[0] != resultLink[0] {
			t.Errorf("Top-level link %d: expected id %v, got %v", i, origLink[0], resultLink[0])
		}
	}
}

// TestRoundtripWorkflowWithoutSubgraphs tests backward compatibility
func TestRoundtripWorkflowWithoutSubgraphs(t *testing.T) {
	// Create a simple workflow without subgraphs
	input := `{
		"nodes": [
			{
				"id": 1,
				"type": "KSampler",
				"pos": [100, 200],
				"size": [300, 400],
				"flags": {},
				"order": 0,
				"mode": 0,
				"inputs": [],
				"outputs": [],
				"properties": {},
				"widgets_values": []
			}
		],
		"links": [
			[1, 1, 0, 2, 0, "IMAGE"]
		],
		"groups": [],
		"last_node_id": 1,
		"last_link_id": 1,
		"version": 0.4
	}`

	var graph Graph
	err := json.Unmarshal([]byte(input), &graph)
	if err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if len(graph.Nodes) != 1 {
		t.Errorf("Expected 1 node, got %d", len(graph.Nodes))
	}
	if len(graph.Links) != 1 {
		t.Errorf("Expected 1 link, got %d", len(graph.Links))
	}
	if graph.Definitions != nil && graph.Definitions.Subgraphs != nil && len(graph.Definitions.Subgraphs) > 0 {
		t.Error("Expected no subgraphs")
	}

	// Re-serialize
	output, err := json.Marshal(&graph)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	// Should be able to deserialize again
	var graph2 Graph
	err = json.Unmarshal(output, &graph2)
	if err != nil {
		t.Fatalf("Failed to unmarshal output: %v", err)
	}

	if len(graph2.Nodes) != 1 {
		t.Errorf("Expected 1 node after roundtrip, got %d", len(graph2.Nodes))
	}
}

// TestRoundtripMultipleSubgraphs tests workflows with multiple subgraph definitions
func TestRoundtripMultipleSubgraphs(t *testing.T) {
	// Read the test workflow file with 2 subgraphs
	data, err := os.ReadFile("../examples/testdata/zimage-2-subgraphs.json")
	if err != nil {
		t.Fatalf("Failed to read test file: %v", err)
	}

	// Deserialize into Graph
	var graph Graph
	err = json.Unmarshal(data, &graph)
	if err != nil {
		t.Fatalf("Failed to unmarshal graph: %v", err)
	}

	// Verify we have 2 subgraphs
	if graph.Definitions == nil || graph.Definitions.Subgraphs == nil {
		t.Fatal("Expected definitions with subgraphs")
	}
	if len(graph.Definitions.Subgraphs) != 2 {
		t.Fatalf("Expected 2 subgraphs, got %d", len(graph.Definitions.Subgraphs))
	}

	// Verify both subgraphs are in the lookup map
	if len(graph.SubgraphsByID) != 2 {
		t.Errorf("Expected 2 subgraphs in lookup map, got %d", len(graph.SubgraphsByID))
	}

	// Verify nodes that reference subgraphs are marked correctly
	subgraphNodeCount := 0
	for _, node := range graph.Nodes {
		if node.IsSubgraph {
			subgraphNodeCount++
			if node.SubgraphDef == nil {
				t.Errorf("Node %d is marked as subgraph but has no definition", node.ID)
			}
		}
	}
	if subgraphNodeCount == 0 {
		t.Error("Expected at least one node to reference a subgraph")
	}

	// Re-serialize to JSON
	output, err := json.MarshalIndent(&graph, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal graph: %v", err)
	}

	// Deserialize both original and output
	var original map[string]interface{}
	err = json.Unmarshal(data, &original)
	if err != nil {
		t.Fatalf("Failed to unmarshal original data: %v", err)
	}

	var result map[string]interface{}
	err = json.Unmarshal(output, &result)
	if err != nil {
		t.Fatalf("Failed to unmarshal output data: %v", err)
	}

	// Verify subgraphs are preserved
	origDefs := original["definitions"].(map[string]interface{})
	resultDefs := result["definitions"].(map[string]interface{})

	origSubgraphs := origDefs["subgraphs"].([]interface{})
	resultSubgraphs := resultDefs["subgraphs"].([]interface{})

	compareField(t, "subgraphs count", len(origSubgraphs), len(resultSubgraphs))

	// Verify each subgraph has the correct link format
	for i, sg := range resultSubgraphs {
		sgMap := sg.(map[string]interface{})
		links := sgMap["links"].([]interface{})

		for j, link := range links {
			linkMap, ok := link.(map[string]interface{})
			if !ok {
				t.Errorf("Subgraph %d link %d should be in object format, got %T", i, j, link)
			}
			if linkMap["id"] == nil || linkMap["origin_id"] == nil {
				t.Errorf("Subgraph %d link %d missing required object format fields", i, j)
			}
		}
	}
}

// TestGraphToPromptWithSubgraphs tests that subgraph nodes are expanded in prompts
func TestGraphToPromptWithSubgraphs(t *testing.T) {
	// Read the test workflow file
	data, err := os.ReadFile("../examples/testdata/zimage-subgraph.json")
	if err != nil {
		t.Fatalf("Failed to read test file: %v", err)
	}

	// Deserialize into Graph
	var graph Graph
	err = json.Unmarshal(data, &graph)
	if err != nil {
		t.Fatalf("Failed to unmarshal graph: %v", err)
	}

	// Generate prompt
	prompt, err := graph.GraphToPrompt("test-client-id")
	if err != nil {
		t.Fatalf("Failed to generate prompt: %v", err)
	}

	// Verify basic structure
	if prompt.ClientID != "test-client-id" {
		t.Errorf("Expected client ID 'test-client-id', got %s", prompt.ClientID)
	}

	if len(prompt.Nodes) == 0 {
		t.Fatal("Expected prompt to have nodes")
	}

	// The workflow has 4 top-level nodes:
	// - Node 9: SaveImage (regular node)
	// - Node 35: MarkdownNote (virtual, should be skipped)
	// - Node 58: PrimitiveStringMultiline (regular node)
	// - Node 57: Subgraph instance (should be expanded to internal nodes)
	//
	// The subgraph has 8 internal nodes (not counting virtual -10/-20)
	// So we expect: 1 (SaveImage) + 1 (PrimitiveStringMultiline) + 8 (subgraph internals) = 10 nodes

	// Count nodes that should appear in prompt
	expectedNodes := 0
	for _, node := range graph.Nodes {
		if !node.IsVirtual() && node.Mode != 2 {
			if node.IsSubgraph {
				// Count internal non-virtual nodes
				if node.SubgraphDef != nil {
					for _, internalNode := range node.SubgraphDef.Nodes {
						if !internalNode.IsVirtual() && internalNode.Mode != 2 {
							expectedNodes++
						}
					}
				}
			} else {
				expectedNodes++
			}
		}
	}

	if len(prompt.Nodes) != expectedNodes {
		t.Errorf("Expected %d nodes in prompt, got %d", expectedNodes, len(prompt.Nodes))
	}

	// Verify that we have nodes from the subgraph (e.g., KSampler, CLIPTextEncode)
	foundKSampler := false
	foundCLIPTextEncode := false
	foundSaveImage := false

	for _, pnode := range prompt.Nodes {
		if pnode.ClassType == "KSampler" {
			foundKSampler = true
		}
		if pnode.ClassType == "CLIPTextEncode" {
			foundCLIPTextEncode = true
		}
		if pnode.ClassType == "SaveImage" {
			foundSaveImage = true
		}
	}

	if !foundKSampler {
		t.Error("Expected to find KSampler node from subgraph expansion")
	}
	if !foundCLIPTextEncode {
		t.Error("Expected to find CLIPTextEncode node from subgraph expansion")
	}
	if !foundSaveImage {
		t.Error("Expected to find SaveImage node from top-level")
	}

	// Verify that original subgraph node ID (57) is NOT in the prompt
	if _, exists := prompt.Nodes["57"]; exists {
		t.Error("Subgraph instance node should not appear in prompt, only its expanded internals")
	}

	// Verify all prompt nodes have valid inputs
	for nodeID, pnode := range prompt.Nodes {
		if pnode.Inputs == nil {
			t.Errorf("Node %s has nil inputs", nodeID)
		}
		// Check that link references are strings
		for inputName, inputVal := range pnode.Inputs {
			if arr, ok := inputVal.([]interface{}); ok {
				if len(arr) != 2 {
					t.Errorf("Node %s input %s: link reference should have 2 elements, got %d", nodeID, inputName, len(arr))
				}
				// First element should be string (node ID)
				if _, ok := arr[0].(string); !ok {
					t.Errorf("Node %s input %s: link reference first element should be string, got %T", nodeID, inputName, arr[0])
				}
			}
		}
	}
}

// TestGraphToPromptWithoutSubgraphs verifies backward compatibility
func TestGraphToPromptWithoutSubgraphs(t *testing.T) {
	input := `{
		"nodes": [
			{
				"id": 1,
				"type": "KSampler",
				"pos": [100, 200],
				"size": [300, 400],
				"flags": {},
				"order": 0,
				"mode": 0,
				"inputs": [
					{
						"name": "model",
						"type": "MODEL",
						"link": 1
					}
				],
				"outputs": [
					{
						"name": "LATENT",
						"type": "LATENT",
						"links": [2]
					}
				],
				"properties": {},
				"widgets_values": [42, "fixed", 20, 8.0, "euler", "normal", 1.0]
			},
			{
				"id": 2,
				"type": "VAEDecode",
				"pos": [500, 200],
				"size": [200, 100],
				"flags": {},
				"order": 1,
				"mode": 0,
				"inputs": [
					{
						"name": "samples",
						"type": "LATENT",
						"link": 2
					}
				],
				"outputs": [],
				"properties": {},
				"widgets_values": []
			}
		],
		"links": [
			[1, 3, 0, 1, 0, "MODEL"],
			[2, 1, 0, 2, 0, "LATENT"]
		],
		"groups": [],
		"last_node_id": 2,
		"last_link_id": 2,
		"version": 0.4
	}`

	var graph Graph
	err := json.Unmarshal([]byte(input), &graph)
	if err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	prompt, err := graph.GraphToPrompt("test-client")
	if err != nil {
		t.Fatalf("Failed to generate prompt: %v", err)
	}

	if len(prompt.Nodes) != 2 {
		t.Errorf("Expected 2 nodes in prompt, got %d", len(prompt.Nodes))
	}

	// Verify node IDs match original
	if _, exists := prompt.Nodes["1"]; !exists {
		t.Error("Expected node 1 in prompt")
	}
	if _, exists := prompt.Nodes["2"]; !exists {
		t.Error("Expected node 2 in prompt")
	}

	// Verify class types
	if prompt.Nodes["1"].ClassType != "KSampler" {
		t.Errorf("Expected node 1 to be KSampler, got %s", prompt.Nodes["1"].ClassType)
	}
	if prompt.Nodes["2"].ClassType != "VAEDecode" {
		t.Errorf("Expected node 2 to be VAEDecode, got %s", prompt.Nodes["2"].ClassType)
	}

	// Verify link in node 2 points to node 1
	samplesInput := prompt.Nodes["2"].Inputs["samples"]
	if samplesInput == nil {
		t.Fatal("Expected samples input in VAEDecode")
	}
	linkRef, ok := samplesInput.([]interface{})
	if !ok || len(linkRef) != 2 {
		t.Fatalf("Expected link reference [nodeID, slot], got %v", samplesInput)
	}
	if linkRef[0].(string) != "1" {
		t.Errorf("Expected link to node 1, got %s", linkRef[0])
	}
}

// TestPromptStructure verifies the detailed structure of generated prompts
func TestPromptStructure(t *testing.T) {
	// Read the test workflow file
	data, err := os.ReadFile("../examples/testdata/zimage-subgraph.json")
	if err != nil {
		t.Fatalf("Failed to read test file: %v", err)
	}

	// Deserialize into Graph
	var graph Graph
	err = json.Unmarshal(data, &graph)
	if err != nil {
		t.Fatalf("Failed to unmarshal graph: %v", err)
	}

	// Generate prompt
	prompt, err := graph.GraphToPrompt("test-client-id")
	if err != nil {
		t.Fatalf("Failed to generate prompt: %v", err)
	}

	// Serialize the prompt to see the structure
	promptJSON, err := json.MarshalIndent(prompt, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal prompt: %v", err)
	}

	// Just log it for inspection during test runs with -v
	t.Logf("Generated prompt structure:\n%s", string(promptJSON))

	// Verify SaveImage node (ID 9) exists and has correct input link
	saveImageNode, exists := prompt.Nodes["9"]
	if !exists {
		t.Fatal("Expected SaveImage node (ID 9) in prompt")
	}

	if saveImageNode.ClassType != "SaveImage" {
		t.Errorf("Expected SaveImage class type, got %s", saveImageNode.ClassType)
	}

	// SaveImage should have an 'images' input that links to the subgraph's output
	// The subgraph (node 57) outputs IMAGE, which comes from internal node 8 (VAEDecode)
	imagesInput := saveImageNode.Inputs["images"]
	if imagesInput == nil {
		t.Fatal("SaveImage should have 'images' input")
	}

	linkRef, ok := imagesInput.([]interface{})
	if !ok {
		t.Fatalf("Expected images input to be a link reference, got %T", imagesInput)
	}

	if len(linkRef) != 2 {
		t.Fatalf("Expected link reference with 2 elements, got %d", len(linkRef))
	}

	// The link should now point to the expanded internal node, not to node 57
	linkedNodeID := linkRef[0].(string)
	if linkedNodeID == "57" {
		t.Error("Link should point to expanded internal node, not to subgraph instance node 57")
	}

	// Verify the linked node exists and is VAEDecode
	linkedNode, exists := prompt.Nodes[linkedNodeID]
	if !exists {
		t.Fatalf("Linked node %s should exist in prompt", linkedNodeID)
	}

	if linkedNode.ClassType != "VAEDecode" {
		t.Errorf("Expected linked node to be VAEDecode (subgraph's output), got %s", linkedNode.ClassType)
	}
}

// TestCreateNodePropertiesWithSubgraphs tests that properties are created for subgraph inputs
func TestCreateNodePropertiesWithSubgraphs(t *testing.T) {
	// Read the test workflow file
	data, err := os.ReadFile("../examples/testdata/zimage-subgraph.json")
	if err != nil {
		t.Fatalf("Failed to read test file: %v", err)
	}

	// Deserialize into Graph
	var graph Graph
	err = json.Unmarshal(data, &graph)
	if err != nil {
		t.Fatalf("Failed to unmarshal graph: %v", err)
	}

	// Create a minimal NodeObjects with just the non-subgraph nodes we need
	var imageData interface{} = []interface{}{"IMAGE"}
	var textData interface{} = []interface{}{"STRING", map[string]interface{}{"multiline": true}}
	stringOutput := []interface{}{"STRING"}

	nodeObjects := &NodeObjects{
		Objects: map[string]*NodeObject{
			"SaveImage": {
				Name:        "SaveImage",
				DisplayName: "Save Image",
				Category:    "image",
				OutputNode:  true,
				Input: &NodeObjectInput{
					Required: map[string]*interface{}{
						"images": &imageData,
					},
				},
			},
			"PrimitiveStringMultiline": {
				Name:        "PrimitiveStringMultiline",
				DisplayName: "Primitive String Multiline",
				Category:    "utils",
				Input: &NodeObjectInput{
					Required: map[string]*interface{}{
						"text": &textData,
					},
				},
				Output: &stringOutput,
			},
			"MarkdownNote": {
				Name:        "MarkdownNote",
				DisplayName: "Markdown Note",
				Category:    "utils",
			},
		},
	}

	// Create properties
	missing := graph.CreateNodeProperties(nodeObjects)

	// The subgraph UUID should NOT be in the missing list since we handle it specially
	if missing != nil {
		for _, m := range *missing {
			if m == "f2fdebf6-dfaf-43b6-9eb2-7f70613cfdc1" {
				t.Error("Subgraph UUID should not be in missing nodes list")
			}
		}
	}

	// Find the subgraph instance node (ID 57)
	var subgraphNode *GraphNode
	for _, node := range graph.Nodes {
		if node.ID == 57 {
			subgraphNode = node
			break
		}
	}

	if subgraphNode == nil {
		t.Fatal("Could not find subgraph node 57")
	}

	// Verify it's marked as a subgraph
	if !subgraphNode.IsSubgraph {
		t.Error("Node 57 should be marked as subgraph")
	}

	// Verify properties were created based on subgraph inputs
	if subgraphNode.Properties == nil || len(subgraphNode.Properties) == 0 {
		t.Fatal("Subgraph node should have properties created")
	}

	// The subgraph has 4 inputs: text, width, height, seed
	expectedProperties := []string{"text", "width", "height", "seed"}
	for _, propName := range expectedProperties {
		prop := subgraphNode.GetPropertyWithName(propName)
		if prop == nil {
			t.Errorf("Expected property %s to exist", propName)
		}
	}

	// Verify property types match subgraph input types
	textProp := subgraphNode.GetPropertyWithName("text")
	if textProp != nil && textProp.TypeString() != "STRING" {
		t.Errorf("Expected text property to be STRING, got %s", textProp.TypeString())
	}

	widthProp := subgraphNode.GetPropertyWithName("width")
	if widthProp != nil && widthProp.TypeString() != "INT" {
		t.Errorf("Expected width property to be INT, got %s", widthProp.TypeString())
	}

	heightProp := subgraphNode.GetPropertyWithName("height")
	if heightProp != nil && heightProp.TypeString() != "INT" {
		t.Errorf("Expected height property to be INT, got %s", heightProp.TypeString())
	}

	seedProp := subgraphNode.GetPropertyWithName("seed")
	if seedProp != nil && seedProp.TypeString() != "INT" {
		t.Errorf("Expected seed property to be INT, got %s", seedProp.TypeString())
	}

	// Verify property values from widget_values
	// Node 57's widget_values are ["", 1280, 720, 0]
	if widthProp != nil {
		widthVal := widthProp.GetValue()
		// widget_values[1] = 1280
		if widthVal != 1280 && widthVal != float64(1280) {
			t.Errorf("Expected width value to be 1280, got %v", widthVal)
		}
	}

	if heightProp != nil {
		heightVal := heightProp.GetValue()
		// widget_values[2] = 720
		if heightVal != 720 && heightVal != float64(720) {
			t.Errorf("Expected height value to be 720, got %v", heightVal)
		}
	}
}

// TestSubgraphPropertiesInPrompt tests that subgraph properties appear correctly in prompts
func TestSubgraphPropertiesInPrompt(t *testing.T) {
	// Read the test workflow file
	data, err := os.ReadFile("../examples/testdata/zimage-subgraph.json")
	if err != nil {
		t.Fatalf("Failed to read test file: %v", err)
	}

	// Deserialize into Graph
	var graph Graph
	err = json.Unmarshal(data, &graph)
	if err != nil {
		t.Fatalf("Failed to unmarshal graph: %v", err)
	}

	// Create a minimal NodeObjects
	nodeObjects := &NodeObjects{
		Objects: map[string]*NodeObject{
			"SaveImage": {
				Name:        "SaveImage",
				DisplayName: "Save Image",
				Category:    "image",
				OutputNode:  true,
			},
			"PrimitiveStringMultiline": {
				Name:        "PrimitiveStringMultiline",
				DisplayName: "Primitive String Multiline",
				Category:    "utils",
			},
		},
	}

	// Create properties
	graph.CreateNodeProperties(nodeObjects)

	// Generate prompt
	prompt, err := graph.GraphToPrompt("test-client")
	if err != nil {
		t.Fatalf("Failed to generate prompt: %v", err)
	}

	// The subgraph's internal EmptySD3LatentImage node should have width/height from the instance
	// Find the EmptySD3LatentImage in the prompt
	var latentNode *PromptNode
	var latentNodeID string
	for nodeID, pnode := range prompt.Nodes {
		if pnode.ClassType == "EmptySD3LatentImage" {
			latentNode = &pnode
			latentNodeID = nodeID
			break
		}
	}

	if latentNode == nil {
		t.Fatal("Expected to find EmptySD3LatentImage in prompt")
	}

	// Verify the width and height inputs are present and have correct values
	// These should come from the subgraph instance's widget_values: [empty, 1280, 720, 0]
	widthInput := latentNode.Inputs["width"]
	heightInput := latentNode.Inputs["height"]

	if widthInput == nil {
		t.Errorf("EmptySD3LatentImage node %s should have width input", latentNodeID)
	} else {
		// The value should be 1280 (from subgraph instance)
		widthVal := widthInput
		if widthVal != 1280 && widthVal != float64(1280) {
			t.Errorf("Expected width to be 1280, got %v", widthVal)
		}
	}

	if heightInput == nil {
		t.Errorf("EmptySD3LatentImage node %s should have height input", latentNodeID)
	} else {
		// The value should be 720 (from subgraph instance)
		heightVal := heightInput
		if heightVal != 720 && heightVal != float64(720) {
			t.Errorf("Expected height to be 720, got %v", heightVal)
		}
	}
}

// TestInternalNodePropertiesInPrompt verifies that internal nodes of subgraphs have their widget values serialized
func TestInternalNodePropertiesInPrompt(t *testing.T) {
	// Read the test workflow file
	data, err := os.ReadFile("../examples/testdata/zimage-subgraph.json")
	if err != nil {
		t.Fatalf("Failed to read test file: %v", err)
	}

	// Deserialize into Graph
	var graph Graph
	err = json.Unmarshal(data, &graph)
	if err != nil {
		t.Fatalf("Failed to unmarshal graph: %v", err)
	}

	// Create a more complete NodeObjects with definitions for internal nodes
	var imageData interface{} = []interface{}{"IMAGE"}
	var stringData interface{} = []interface{}{"STRING", map[string]interface{}{"multiline": true}}
	stringOutput := []interface{}{"STRING"}
	clipOutput := []interface{}{"CLIP"}
	vaeOutput := []interface{}{"VAE"}
	modelOutput := []interface{}{"MODEL"}
	imageOutput := []interface{}{"IMAGE"}
	var latentData interface{} = []interface{}{"LATENT"}
	var vaeData interface{} = []interface{}{"VAE"}

	var clipNameData interface{} = []interface{}{[]interface{}{"qwen_3_4b.safetensors"}}
	var typeData interface{} = []interface{}{[]interface{}{"lumina2", "default"}}
	var vaeNameData interface{} = []interface{}{[]interface{}{"ae.safetensors"}}
	var unetNameData interface{} = []interface{}{[]interface{}{"z_image_turbo_bf16.safetensors"}}
	var weightDtypeData interface{} = []interface{}{[]interface{}{"default", "fp8_e4m3fn", "fp8_e5m2"}}
	var conditioningData interface{} = []interface{}{"CONDITIONING"}
	var clipData interface{} = []interface{}{"CLIP"}
	var textData interface{} = []interface{}{"STRING", map[string]interface{}{"multiline": true}}
	var modelData interface{} = []interface{}{"MODEL"}
	conditioningOutput := []interface{}{"CONDITIONING"}
	latentOutput := []interface{}{"LATENT"}
	var widthData interface{} = []interface{}{"INT", map[string]interface{}{"default": float64(1280), "min": float64(16), "max": float64(16384)}}
	var heightData interface{} = []interface{}{"INT", map[string]interface{}{"default": float64(720), "min": float64(16), "max": float64(16384)}}
	var batchData interface{} = []interface{}{"INT", map[string]interface{}{"default": float64(1), "min": float64(1), "max": float64(4096)}}
	var seedData interface{} = []interface{}{"INT", map[string]interface{}{"default": float64(0), "min": float64(0), "max": float64(4294967295)}}
	var stepsData interface{} = []interface{}{"INT", map[string]interface{}{"default": float64(6), "min": float64(1), "max": float64(10000)}}
	var cfgData interface{} = []interface{}{"FLOAT", map[string]interface{}{"default": 1.0, "min": 0.0, "max": 100.0}}
	var samplerNameData interface{} = []interface{}{[]interface{}{"euler", "euler_ancestral", "heun", "heunpp2", "dpm_2", "dpm_2_ancestral", "lms", "dpm_fast", "dpm_adaptive", "dpmpp_2s_ancestral", "dpmpp_sde", "dpmpp_sde_gpu", "dpmpp_2m", "dpmpp_2m_sde", "dpmpp_2m_sde_gpu", "dpmpp_3m_sde", "dpmpp_3m_sde_gpu", "ddpm", "lcm", "ddim", "uni_pc", "uni_pc_bh2"}}
	var schedulerData interface{} = []interface{}{[]interface{}{"normal", "karras", "exponential", "sgm_uniform", "simple", "ddim_uniform"}}
	var denoise interface{} = []interface{}{"FLOAT", map[string]interface{}{"default": 1.0, "min": 0.0, "max": 1.0}}
	var shiftData interface{} = []interface{}{"FLOAT", map[string]interface{}{"default": 3.0, "min": 0.0, "max": 100.0}}

	nodeObjects := &NodeObjects{
		Objects: map[string]*NodeObject{
			"SaveImage": {
				Name:        "SaveImage",
				DisplayName: "Save Image",
				Category:    "image",
				OutputNode:  true,
				Input: &NodeObjectInput{
					Required: map[string]*interface{}{
						"images": &imageData,
					},
				},
			},
			"PrimitiveStringMultiline": {
				Name:        "PrimitiveStringMultiline",
				DisplayName: "Primitive String Multiline",
				Category:    "utils",
				Input: &NodeObjectInput{
					Required: map[string]*interface{}{
						"text": &stringData,
					},
				},
				Output: &stringOutput,
			},
			"CLIPLoader": {
				Name:        "CLIPLoader",
				DisplayName: "Load CLIP",
				Category:    "loaders",
				Input: &NodeObjectInput{
					Required: map[string]*interface{}{
						"clip_name": &clipNameData,
						"type":      &typeData,
					},
					OrderedRequired: []string{"clip_name", "type"},
				},
				Output: &clipOutput,
			},
			"VAELoader": {
				Name:        "VAELoader",
				DisplayName: "Load VAE",
				Category:    "loaders",
				Input: &NodeObjectInput{
					Required: map[string]*interface{}{
						"vae_name": &vaeNameData,
					},
					OrderedRequired: []string{"vae_name"},
				},
				Output: &vaeOutput,
			},
			"UNETLoader": {
				Name:        "UNETLoader",
				DisplayName: "Load Diffusion Model",
				Category:    "loaders",
				Input: &NodeObjectInput{
					Required: map[string]*interface{}{
						"unet_name":    &unetNameData,
						"weight_dtype": &weightDtypeData,
					},
					OrderedRequired: []string{"unet_name", "weight_dtype"},
				},
				Output: &modelOutput,
			},
			"VAEDecode": {
				Name:        "VAEDecode",
				DisplayName: "VAE Decode",
				Category:    "latent",
				Input: &NodeObjectInput{
					Required: map[string]*interface{}{
						"samples": &latentData,
						"vae":     &vaeData,
					},
					OrderedRequired: []string{"samples", "vae"},
				},
				Output: &imageOutput,
			},
			"ConditioningZeroOut": {
				Name:        "ConditioningZeroOut",
				DisplayName: "Conditioning (Zero Out)",
				Category:    "conditioning",
				Input: &NodeObjectInput{
					Required: map[string]*interface{}{
						"conditioning": &conditioningData,
					},
					OrderedRequired: []string{"conditioning"},
				},
				Output: &conditioningOutput,
			},
			"CLIPTextEncode": {
				Name:        "CLIPTextEncode",
				DisplayName: "CLIP Text Encode",
				Category:    "conditioning",
				Input: &NodeObjectInput{
					Required: map[string]*interface{}{
						"text": &textData,
						"clip": &clipData,
					},
					OrderedRequired: []string{"text", "clip"},
				},
				Output: &conditioningOutput,
			},
			"EmptySD3LatentImage": {
				Name:        "EmptySD3LatentImage",
				DisplayName: "Empty SD3 Latent Image",
				Category:    "latent",
				Input: &NodeObjectInput{
					Required: map[string]*interface{}{
						"width":  &widthData,
						"height": &heightData,
						"batch":  &batchData,
					},
					OrderedRequired: []string{"width", "height", "batch"},
				},
				Output: &latentOutput,
			},
			"ModelSamplingAuraFlow": {
				Name:        "ModelSamplingAuraFlow",
				DisplayName: "Model Sampling AuraFlow",
				Category:    "model_patches",
				Input: &NodeObjectInput{
					Required: map[string]*interface{}{
						"model": &modelData,
						"shift": &shiftData,
					},
					OrderedRequired: []string{"model", "shift"},
				},
				Output: &modelOutput,
			},
			"KSampler": {
				Name:        "KSampler",
				DisplayName: "KSampler",
				Category:    "sampling",
				Input: &NodeObjectInput{
					Required: map[string]*interface{}{
						"model":        &modelData,
						"seed":         &seedData,
						"steps":        &stepsData,
						"cfg":          &cfgData,
						"sampler_name": &samplerNameData,
						"scheduler":    &schedulerData,
						"positive":     &conditioningData,
						"negative":     &conditioningData,
						"latent_image": &latentData,
						"denoise":      &denoise,
					},
					OrderedRequired: []string{"model", "seed", "steps", "cfg", "sampler_name", "scheduler", "positive", "negative", "latent_image", "denoise"},
				},
				Output: &latentOutput,
			},
		},
	}

	// IMPORTANT: Populate input properties before creating node properties
	// This is what populates the InputProperties field of each NodeObject
	nodeObjects.PopulateInputProperties()

	// Create properties for all nodes (including subgraph internals)
	missing := graph.CreateNodeProperties(nodeObjects)
	if missing != nil && len(*missing) > 0 {
		t.Logf("Missing node types: %v", *missing)
	}

	// Generate prompt
	prompt, err := graph.GraphToPrompt("test-client")
	if err != nil {
		t.Fatalf("Failed to generate prompt: %v", err)
	}

	// Find VAELoader in the prompt (should be expanded from subgraph)
	var vaeLoaderNode *PromptNode
	var vaeLoaderNodeID string
	for nodeID, pnode := range prompt.Nodes {
		if pnode.ClassType == "VAELoader" {
			vaeLoaderNode = &pnode
			vaeLoaderNodeID = nodeID
			break
		}
	}

	if vaeLoaderNode == nil {
		t.Fatal("Expected to find VAELoader in prompt")
	}

	// Verify VAELoader has vae_name input
	vaeNameInput := vaeLoaderNode.Inputs["vae_name"]
	if vaeNameInput == nil {
		t.Errorf("VAELoader node %s should have vae_name input", vaeLoaderNodeID)
	} else {
		// The value should be "ae.safetensors" from the subgraph's internal node widget_values
		if vaeNameInput != "ae.safetensors" {
			t.Errorf("Expected vae_name to be 'ae.safetensors', got %v", vaeNameInput)
		}
	}

	// Find CLIPLoader in the prompt
	var clipLoaderNode *PromptNode
	var clipLoaderNodeID string
	for nodeID, pnode := range prompt.Nodes {
		if pnode.ClassType == "CLIPLoader" {
			clipLoaderNode = &pnode
			clipLoaderNodeID = nodeID
			break
		}
	}

	if clipLoaderNode == nil {
		t.Fatal("Expected to find CLIPLoader in prompt")
	}

	// Verify CLIPLoader has clip_name and type inputs
	clipNameInput := clipLoaderNode.Inputs["clip_name"]
	if clipNameInput == nil {
		t.Errorf("CLIPLoader node %s should have clip_name input", clipLoaderNodeID)
	} else {
		if clipNameInput != "qwen_3_4b.safetensors" {
			t.Errorf("Expected clip_name to be 'qwen_3_4b.safetensors', got %v", clipNameInput)
		}
	}

	typeInput := clipLoaderNode.Inputs["type"]
	if typeInput == nil {
		t.Errorf("CLIPLoader node %s should have type input", clipLoaderNodeID)
	} else {
		if typeInput != "lumina2" {
			t.Errorf("Expected type to be 'lumina2', got %v", typeInput)
		}
	}

	t.Logf("Internal nodes have their widget values properly serialized")
}

func compareField(t *testing.T, name string, expected, actual interface{}) {
	if expected != actual {
		t.Errorf("%s mismatch: expected %v, got %v", name, expected, actual)
	}
}
