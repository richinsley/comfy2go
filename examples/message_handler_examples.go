package examples

// This file demonstrates various patterns for using the MessageHandlers API

import (
	"log"
	"os"

	"github.com/richinsley/comfy2go/client"
)

// Example 1: Minimal - only handle what you care about
func MinimalExample(item *client.QueueItem, c *client.ComfyClient) {
	err := item.ProcessMessages(&client.MessageHandlers{
		OnData: func(msg *client.PromptMessageData) {
			for _, images := range msg.Data["images"] {
				log.Printf("Got image: %s", images.Filename)
			}
		},
	})

	if err != nil {
		log.Fatal(err)
	}
}

// Example 2: Use defaults with builder pattern
func DefaultsWithCustomDataHandler(item *client.QueueItem, c *client.ComfyClient) {
	err := item.ProcessMessages(
		client.DefaultMessageHandlers().
			WithDataHandler(func(msg *client.PromptMessageData) {
				// Custom data handling
				for k, v := range msg.Data {
					if k == "images" {
						for _, output := range v {
							imgData, _ := c.GetImage(output)
							os.WriteFile(output.Filename, *imgData, 0644)
							log.Printf("Saved: %s", output.Filename)
						}
					}
				}
			}),
	)

	if err != nil {
		log.Fatal(err)
	}
}

// Example 3: Custom progress tracking
func CustomProgressExample(item *client.QueueItem) {
	type NodeProgress struct {
		NodeID   string
		Progress int
		Max      int
	}

	var currentNode NodeProgress

	err := item.ProcessMessages(&client.MessageHandlers{
		OnExecuting: func(msg *client.PromptMessageExecuting) {
			currentNode = NodeProgress{
				NodeID: msg.NodeID,
			}
			log.Printf("→ %s", msg.Title)
		},
		OnProgress: func(msg *client.PromptMessageProgress) {
			currentNode.Progress = msg.Value
			currentNode.Max = msg.Max
			percent := float64(msg.Value) / float64(msg.Max) * 100
			log.Printf("  %.1f%% (%d/%d)", percent, msg.Value, msg.Max)
		},
	})

	if err != nil {
		log.Fatal(err)
	}
}

// Example 4: Using new progress_state message for detailed tracking
func ProgressStateExample(item *client.QueueItem) {
	err := item.ProcessMessages(&client.MessageHandlers{
		OnProgressState: func(msg *client.PromptMessageProgressState) {
			log.Printf("Progress state update for prompt %s:", msg.PromptID)
			for nodeID, nodeInfo := range msg.Nodes {
				log.Printf("  Node %s: %s (%.0f/%.0f)",
					nodeID, nodeInfo.State, nodeInfo.Value, nodeInfo.Max)
			}
		},
		OnExecutionSuccess: func(msg *client.PromptMessageExecutionSuccess) {
			log.Printf("Execution completed successfully at %d", msg.Timestamp)
		},
	})

	if err != nil {
		log.Fatal(err)
	}
}

// Example 5: Progress bar (application-level UI)
// Note: Progress bars are NOT included in DefaultMessageHandlers to reduce noise
// Add them in your application code if desired
func ProgressBarExample(item *client.QueueItem) {
	// Import: "github.com/schollz/progressbar/v3"
	var bar *progressbar.ProgressBar
	var currentNodeTitle string

	err := item.ProcessMessages(
		client.DefaultMessageHandlers().
			WithExecutingHandler(func(msg *client.PromptMessageExecuting) {
				bar = nil // Reset bar for new node
				currentNodeTitle = msg.Title
				log.Printf("Executing: %s", msg.Title)
			}).
			WithProgressHandler(func(msg *client.PromptMessageProgress) {
				if bar == nil {
					bar = progressbar.Default(int64(msg.Max), currentNodeTitle)
				}
				bar.Set(msg.Value)
			}),
	)

	if err != nil {
		log.Fatal(err)
	}
}

// Example 6: Error handling with cleanup
func ErrorHandlingExample(item *client.QueueItem) {
	tempFiles := []string{}

	err := item.ProcessMessages(&client.MessageHandlers{
		OnData: func(msg *client.PromptMessageData) {
			for _, outputs := range msg.Data {
				for _, output := range outputs {
					if output.Type == "temp" {
						tempFiles = append(tempFiles, output.Filename)
					}
				}
			}
		},
		OnError: func(err *client.PromptMessageStoppedException) {
			log.Printf("❌ Error in node %s (%s): %s",
				err.NodeName, err.NodeType, err.ExceptionMessage)
			log.Printf("Stack trace:")
			for _, line := range err.Traceback {
				log.Println(line)
			}
		},
		OnComplete: func() {
			// Cleanup temp files
			for _, f := range tempFiles {
				os.Remove(f)
			}
			log.Printf("Cleaned up %d temp files", len(tempFiles))
		},
	})

	if err != nil {
		log.Printf("Execution failed: %v", err)
	}
}

// Example 7: Comprehensive monitoring
func ComprehensiveExample(item *client.QueueItem, c *client.ComfyClient) {
	imageCount := 0
	nodeCount := 0

	handlers := &client.MessageHandlers{
		OnStarted: func(msg *client.PromptMessageStarted) {
			log.Printf("🚀 Starting execution: %s", msg.PromptID)
		},

		OnExecuting: func(msg *client.PromptMessageExecuting) {
			nodeCount++
			log.Printf("⚙️  [%d] Executing: %s", nodeCount, msg.Title)
		},

		OnProgress: func(msg *client.PromptMessageProgress) {
			// Log every 10% progress
			if msg.Value%max(1, msg.Max/10) == 0 {
				percent := float64(msg.Value) / float64(msg.Max) * 100
				log.Printf("   %.0f%% complete", percent)
			}
		},

		OnProgressState: func(msg *client.PromptMessageProgressState) {
			// Count how many nodes are in each state
			states := make(map[string]int)
			for _, node := range msg.Nodes {
				states[node.State]++
			}
			log.Printf("📊 Node states: %v", states)
		},

		OnData: func(msg *client.PromptMessageData) {
			for dataType, outputs := range msg.Data {
				for _, output := range outputs {
					imageCount++
					log.Printf("💾 Saving %s: %s", dataType, output.Filename)

					imgData, err := c.GetImage(output)
					if err != nil {
						log.Printf("⚠️  Failed to get image: %v", err)
						continue
					}

					err = os.WriteFile(output.Filename, *imgData, 0644)
					if err != nil {
						log.Printf("⚠️  Failed to save image: %v", err)
						continue
					}
				}
			}
		},

		OnExecutionSuccess: func(msg *client.PromptMessageExecutionSuccess) {
			log.Printf("✅ Execution succeeded at timestamp %d", msg.Timestamp)
		},

		OnStopped: func(msg *client.PromptMessageStopped) {
			if msg.Exception == nil {
				log.Printf("✨ Completed successfully!")
				log.Printf("   Processed %d nodes", nodeCount)
				log.Printf("   Generated %d images", imageCount)
			}
		},

		OnError: func(err *client.PromptMessageStoppedException) {
			log.Printf("❌ Error in %s: %s", err.NodeName, err.ExceptionMessage)
		},

		OnComplete: func() {
			log.Println("🏁 Message processing complete")
		},
	}

	err := item.ProcessMessages(handlers)
	if err != nil {
		log.Fatalf("Execution failed: %v", err)
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
