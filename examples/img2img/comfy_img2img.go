package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"log"
	"math"
	"os"

	"github.com/richinsley/comfy2go/client"
	"github.com/schollz/progressbar/v3"
)

// process CLI arguments
func procCLI() (string, int) {
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

	return *serverAddress, *serverPort
}

func main() {
	clientaddr, clientport := procCLI()

	callbacks := &client.ComfyClientCallbacks{
		ClientQueueCountChanged: func(c *client.ComfyClient, queuecount int) {
			log.Printf("Client %s at %s Queue size: %d", c.ClientID(), clientaddr, queuecount)
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
	graph, _, err := c.NewGraphFromJsonFile("img2img.json")
	if err != nil {
		log.Println("Error loading graph JSON:", err)
		os.Exit(1)
	}

	// Create a simple 512x512 image of some rolling green hills
	img := image.NewRGBA(image.Rect(0, 0, 512, 512))

	// Draw the blue sky
	skyColor := color.RGBA{135, 206, 250, 255} // Sky blue color
	draw.Draw(img, image.Rect(0, 0, 512, 256), &image.Uniform{skyColor}, image.Point{}, draw.Src)

	// Draw green hills
	mountainColor := color.RGBA{34, 139, 34, 255} // Forest green color
	for x := 0; x < 512; x++ {
		y := 256 + int(50*math.Sin(float64(x)*0.02)) // Sine wave for hills
		draw.Draw(img, image.Rect(x, y, x+1, 512), &image.Uniform{mountainColor}, image.Point{}, draw.Src)
	}

	// upload our image.  The workflow should only have one "Load Image"
	loadImageNode := graph.GetFirstNodeWithTitle("Load Image")
	if loadImageNode == nil {
		log.Println("missing Load Image node")
	} else {
		// get the property interface for "choose file to upload" or the alias "file"
		prop := loadImageNode.GetPropertyWithName("choose file to upload")
		if prop == nil {
			log.Println("missing property \"choose file to upload\"")
		} else {
			// the ImageUploadProperty value is not directly settable.  We need to pass the property to the call to client.UploadImage
			uploadprop, _ := prop.ToImageUploadProperty()

			// because we set it to not overwrite existing, the returned filename may
			// be different than the one we provided
			_, err := c.UploadImage(img, "mountains.png", false, client.InputImageType, "", uploadprop)
			if err != nil {
				log.Println("Uploading image:", err)
				os.Exit(1)
			}
		}
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
