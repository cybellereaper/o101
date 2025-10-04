package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/cybellereaper/wizturtle/v2/internal/messagesorter"
)

func main() {
	var outputDir string
	flag.StringVar(&outputDir, "out", "", "optional directory for the generated message listing")
	flag.Parse()

	if flag.NArg() != 1 {
		fmt.Fprintf(os.Stderr, "usage: %s [flags] <capture-file>\n", filepath.Base(os.Args[0]))
		flag.PrintDefaults()
		os.Exit(2)
	}

	input := flag.Arg(0)
	outputPath, result, err := messagesorter.ProcessFile(input, outputDir)
	if err != nil {
		log.Fatalf("failed to process capture: %v", err)
	}

	fmt.Printf("wrote %d messages for service %s (%s) to %s\n", len(result.Messages), result.ServiceName, result.ServiceID, outputPath)
}
