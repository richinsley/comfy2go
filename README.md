# Comfy2go

Comfy2go is a Go-based API that acts as a bridge to ComfyUI, a powerful and modular stable diffusion GUI and backend. Designed to alleviate the complexities of working directly with ComfyUI's intricate API, Comfy2go offers a more user-friendly way to access the advanced features and functionalities of ComfyUI.


## Table of Contents

- [Overview](#overview)
- [Installation](#installation)
- [Usage](#usage)

## Overview

Comfy2go allows for developers to harness ComfyUI's powerful features in a more accessible way. Comfy2go is comprised of two main parts:

### GraphAPI
The GraphAPI approximates the functionality of ComfyUI's front-end graph-based pipeline.  While it does not allow for creating or editing existing workflows, it does allow for quickly finding, and setting the various inputs of each node in a workflow.

### ClientAPI
The ClientAPI interoperates with the ComfyUI backend, offering:
- Backend system statistics
- Concurrent access to mulitple instances of ComfyUI
- Image and mask uploading/downloading
- Creating and queuing prompts from GraphAPI workflows
- Managing Queues
- Retreival of Prompt histories
- Loading workflows from PNG
- and quite a bit more

## Installation
First, use 'go get' to install the latest version of the library.
```bash
go get -u github.com/richinsley/comfy2go@latest
```
Next, include Comfy2go client (and optionally the graph) APIs in your application:
```go
import "github.com/richinsley/comfy2go/client"
import "github.com/richinsley/comfy2go/graphapi"
```
## Usage
An **IMPORTANT** note is that Comfy2go works with full ComfyUI workflows, not workflows saved with "Save (API Format)"

#### Load a workflow from a png and queue it to a ComfyUI instance
```go
package main

import (
	"log"
	"os"

	"github.com/richinsley/comfy2go/client"
)

func main() {
	clientaddr := "127.0.0.1"
	clientport := 8188
	pngpath := "my_cool_workflow.png"

	// create a new ComgyGo client
	c := client.NewComfyClient(clientaddr, clientport, nil)

	// the ComgyGo client needs to be in an initialized state before
	// we can create and queue graphs
	if !c.IsInitialized() {
		log.Printf("Initialize Client with ID: %s\n", c.ClientID())
		err := c.Init()
		if err != nil {
			log.Println("Error initializing client:", err)
			os.Exit(1)
		}
	}

	// create a graph from the png file
	graph, err := c.NewGraphFromPNGFile(pngpath)
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

	// continuously read messages from the QueuedItem until we get the "stopped" message type
	for continueLoop := true; continueLoop; {
		msg := <-item.Messages
		switch msg.Type {
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
						log.Println("Got image: ", output.Filename)
					}
				}
			}
		}
	}
}

```
