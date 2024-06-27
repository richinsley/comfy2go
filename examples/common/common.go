package common

import (
	"flag"
	"fmt"
	"os"
)

// process CLI arguments
func ProcCLI() (string, string, int) {
	serverAddress := flag.String("address", "localhost", "Server address")
	protocolType := flag.String("protocolType", "http", "http or https")
	serverPort := flag.Int("port", 8188, "Server port")
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage of %s:\n", os.Args[0])
		fmt.Printf("  %s [OPTIONS] filename", os.Args[0])
		fmt.Println("\nOptions:")
		flag.PrintDefaults()
		fmt.Println("\nfilename: Path to workflow json file")
	}
	flag.Parse()

	return *protocolType, *serverAddress, *serverPort
}

func ProcFileCLI() (string, string, int, string) {
	serverAddress := flag.String("address", "localhost", "Server address")
	protocolType := flag.String("protocolType", "http", "http or https")
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
	return *protocolType, *serverAddress, *serverPort, filename
}
