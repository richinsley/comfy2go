package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/richinsley/comfy2go/client"
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

func displayAvailableNodes(c *client.ComfyClient) {
	object_infos, err := c.GetObjectInfos()
	if err != nil {
		log.Println("Error decoding Object Infos:", err)
		os.Exit(1)
	}

	log.Println("Available Nodes:")
	for _, n := range object_infos.Objects {
		log.Printf("\tNode Name: \"%s\"\n", n.DisplayName)
		props := n.GetSettableProperties()
		log.Printf("\t\tProperties:\n")
		for _, p := range props {
			log.Printf("\t\t\t\"%s\"\tType: [%s]\n", p.Name(), p.TypeString())
			if p.TypeString() == "COMBO" {
				c, _ := p.ToComboProperty()
				for _, combo_item := range c.Values {
					log.Printf("\t\t\t\t\"%s\"\n", combo_item)
				}
			}
		}
	}
}

// displayExtensions gets the installed extensions
func displayExtensions(c *client.ComfyClient) {
	extensions, err := c.GetExtensions()
	if err != nil {
		log.Println("Error decoding System Stats:", err)
		os.Exit(1)
	}
	log.Println("Instaled extensions:")
	for _, e := range extensions {
		log.Printf("\t%s\n", e)
	}
	log.Println()
}

// displaySystemStats gets the system statistics for the client
func displaySystemStats(c *client.ComfyClient) {
	// Get System Stats
	system_info, err := c.GetSystemStats()
	if err != nil {
		log.Println("Error decoding System Stats:", err)
		os.Exit(1)
	}
	log.Println("System Stats:")
	log.Printf("\tOS: %s\n", system_info.System.OS)
	log.Printf("\tPython Version: %s\n", system_info.System.PythonVersion)
	log.Println("\tDevices:")
	for _, dev := range system_info.Devices {
		log.Printf("\t\tIndex: %d\n", dev.Index)
		log.Printf("\t\tName: %s\n", dev.Name)
		log.Printf("\t\tType: %s\n", dev.Type)
		log.Printf("\t\tVRAM Total %d\n", dev.VRAM_Total)
		log.Printf("\t\tVRAM Free %d\n", dev.VRAM_Free)
		log.Printf("\t\tTorch VRAM Total %d\n", dev.Torch_VRAM_Total)
		log.Printf("\t\tTorch VRAM Free %d\n", dev.Torch_VRAM_Free)
	}
	log.Println()
}

// displayPromptHistory gets the prompt history from a client and displays them
func displayPromptHistory(c *client.ComfyClient) {
	// Get prompt history in order
	prompt_history, err := c.GetPromptHistoryByIndex()
	if err != nil {
		log.Println("Error decoding Prompt Histories:", err)
		os.Exit(1)
	}

	// iterate over prompt history items and display
	log.Println("Prompt History:")
	for _, p := range prompt_history {
		log.Printf("\tPrompt index: %d Prompt ID: %s\n", p.Index, p.PromptID)
		log.Println("\tOutput nodes:")
		for nodeid, out := range p.Outputs {
			log.Printf("\t\tNode ID %d\n", nodeid)
			for _, img_data := range out {
				log.Printf("\t\t\tFilename: %s Type: \"%s\" Subfolder: %s\n", img_data.Filename, img_data.Type, img_data.Subfolder)
			}
		}
	}
	log.Println()
}

func main() {
	clientaddr, clientport := procCLI()

	// create a client
	c := client.NewComfyClient(clientaddr, clientport, nil)

	// Becuase we are not going to be creating or queuing prompts, we do not
	// need to initialize the client.

	// display system stats
	displaySystemStats(c)

	// display installed extensions
	displayExtensions(c)

	// display prompt history
	displayPromptHistory(c)

	// display available nodes with thier properties
	displayAvailableNodes(c)
}
