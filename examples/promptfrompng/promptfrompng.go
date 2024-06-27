package main

import (
	"github.com/richinsley/comfy2go/examples/common"
	"log"
	"os"

	"github.com/richinsley/comfy2go/client"
	"github.com/schollz/progressbar/v3"
)

func main() {
	protocolType, clientaddr, clientport, workflow := common.ProcFileCLI()

	callbacks := &client.ComfyClientCallbacks{
		ClientQueueCountChanged: func(c *client.ComfyClient, queuecount int) {
			log.Printf("Client %s at %s Queue size: %d", c.ClientID(), clientaddr, queuecount)
		},
	}

	// create a client
	c := client.NewComfyClient(clientaddr, clientport, callbacks, protocolType)

	// the client needs to be in an initialized state before usage
	if !c.IsInitialized() {
		log.Printf("Initialize Client with ID: %s\n", c.ClientID())
		err := c.Init()
		if err != nil {
			log.Println("Error initializing client:", err)
			os.Exit(1)
		}
	}

	// create a graph from the png file
	graph, _, err := c.NewGraphFromPNGFile(workflow)
	if err != nil {
		log.Println("Failed to get workflow graph from png file:", err)
		os.Exit(1)
	}

	// queue the prompt and get the resulting image
	item, err := c.QueuePrompt(graph)
	if err != nil {
		log.Println("Failed to queue prompt:", err)
		os.Exit(1)
	}

	// we'll provide a progress bar
	var bar *progressbar.ProgressBar = nil

	// continuously read messages from the QueuedItem until we get the "stopped" message type
	var currentNodeTitle string
	for continueLoop := true; continueLoop; {
		msg := <-item.Messages
		switch msg.Type {
		case "started":
			qm := msg.ToPromptMessageStarted()
			log.Printf("Start executing prompt ID %s\n", qm.PromptID)
		case "executing":
			bar = nil
			qm := msg.ToPromptMessageExecuting()
			// store the node's title so we can use it in the progress bar
			currentNodeTitle = qm.Title
			log.Printf("Executing Node: %d\n", qm.NodeID)
		case "progress":
			// update our progress bar
			qm := msg.ToPromptMessageProgress()
			if bar == nil {
				bar = progressbar.Default(int64(qm.Max), currentNodeTitle)
			}
			bar.Set(qm.Value)
		case "stopped":
			// if we were stopped for an exception, display the exception message
			qm := msg.ToPromptMessageStopped()
			if qm.Exception != nil {
				log.Println(qm.Exception)
				os.Exit(1)
			}
			continueLoop = false
		case "data":
			qm := msg.ToPromptMessageData()
			// data objects have the fields: Filename, Subfolder, Type
			// * Subfolder is the subfolder in the output directory
			// * Type is the type of the image temp/
			for k, v := range qm.Data {
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
						log.Println("Got data: ", output.Filename)
					}
				}
			}
		}
	}
}
