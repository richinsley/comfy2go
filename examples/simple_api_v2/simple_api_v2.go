package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/richinsley/comfy2go/client"
)

// process CLI arguments
func procCLI() (string, int, string) {
	serverAddress := flag.String("address", "localhost", "Server address")
	serverPort := flag.Int("port", 8188, "Server port")
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage of %s:\n", os.Args[0])
		fmt.Printf("  %s [OPTIONS] filename", os.Args[0])
		fmt.Println("\nOptions:")
		flag.PrintDefaults()
		fmt.Println("\nfilename: Path to workflow json file")
	}
	flag.Parse()

	// Check for required filename argument
	if len(flag.Args()) != 1 {
		flag.Usage()
		os.Exit(1)
	}
	filename := flag.Arg(0)
	return *serverAddress, *serverPort, filename
}

func main() {
	clientaddr, clientport, workflow := procCLI()

	// callbacks can be used respond to QueuedItem updates, or client status changes
	callbacks := &client.ComfyClientCallbacks{
		ClientQueueCountChanged: func(c *client.ComfyClient, queuecount int) {
			log.Printf("Client %s at %s Queue size: %d", c.ClientID(), clientaddr, queuecount)
		},
		QueuedItemStarted: func(c *client.ComfyClient, qi *client.QueueItem) {
			log.Printf("Queued item %s started\n", qi.PromptID)
		},
		QueuedItemStopped: func(cc *client.ComfyClient, qi *client.QueueItem, reason client.QueuedItemStoppedReason) {
			log.Printf("Queued item %s stopped\n", qi.PromptID)
		},
		QueuedItemDataAvailable: func(cc *client.ComfyClient, qi *client.QueueItem, pmd *client.PromptMessageData) {
			log.Printf("image data available:\n")
		},
	}

	// create a client
	c := client.NewComfyClient(clientaddr, clientport, callbacks)

	// the client needs to be in an initialized state before usage
	if !c.IsInitialized() {
		log.Printf("Initialize Client with ID: %s\n", c.ClientID())
		err := c.Init()
		if err != nil {
			log.Println("Error initializing client:", err)
			os.Exit(1)
		}
	}

	// load the workflow
	graph, _, err := c.NewGraphFromJsonFile(workflow)
	if err != nil {
		log.Println("Error loading graph JSON:", err)
		os.Exit(1)
	}

	// Get the nodes that are within the "API" Group.  GetSimpleAPI takes each
	// node and exposes it's first (and only it's first) property, with the title of the node as the key
	// in the Properties field.
	simple_api := graph.GetSimpleAPI(nil)
	width := simple_api.Properties["Width"]
	height := simple_api.Properties["Height"]
	positive := simple_api.Properties["Positive"]
	negative := simple_api.Properties["Negative"]
	width.SetValue(1024)
	height.SetValue(1024)
	positive.SetValue("a dive bar, dimly lit, zombies, dancing, mosh pit")
	negative.SetValue("text, watermark")

	// or we can set it directly
	simple_api.Properties["Seed"].SetValue(2290222)

	// RECOMMENDED APPROACH: Use QueuePromptAndProcess to avoid race conditions
	// This atomically queues the prompt and starts processing messages
	err = c.QueuePromptAndProcess(graph,
		client.DefaultMessageHandlers().
			WithDataHandler(func(msg *client.PromptMessageData) {
				// Handle output data - save images/gifs
				for k, v := range msg.Data {
					if k == "images" || k == "gifs" {
						for _, output := range v {
							img_data, err := c.GetImage(output)
							if err != nil {
								log.Println("Failed to get image:", err)
								os.Exit(1)
							}
							f, err := os.Create(output.Filename)
							if err != nil {
								log.Println("Failed to write image:", err)
								os.Exit(1)
							}
							f.Write(*img_data)
							f.Close()
							log.Println("Saved:", output.Filename)
						}
					}
				}
			}),
	)

	if err != nil {
		log.Fatal("Execution failed:", err)
	}
}
