package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/quantarax/backend/internal/chunker"
)

func main() {
	// Define flags
	chunkSize := flag.Int("chunk-size", 1048576, "Chunk size in bytes (default: 1 MiB)")
	output := flag.String("output", "", "Output manifest to file (default: stdout)")
	pretty := flag.Bool("pretty", true, "Pretty-print JSON output")
	flag.Parse()

	// Check for file argument
	if flag.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "Usage: chunker [options] <file_path>")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Options:")
		flag.PrintDefaults()
		os.Exit(1)
	}

	filePath := flag.Arg(0)

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Error: File not found: %s\n", filePath)
		os.Exit(2)
	}

	fmt.Fprintf(os.Stderr, "Processing file: %s\n", filePath)

	// Compute manifest
	options := chunker.ChunkOptions{
		ChunkSize: *chunkSize,
	}

	manifest, err := chunker.ComputeManifest(filePath, options)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error computing manifest: %v\n", err)
		os.Exit(3)
	}

	fmt.Fprintf(os.Stderr, "File size: %d bytes\n", manifest.FileSize)
	fmt.Fprintf(os.Stderr, "Chunk size: %d bytes\n", manifest.ChunkSize)
	fmt.Fprintf(os.Stderr, "Chunks: %d\n", manifest.ChunkCount)
	fmt.Fprintf(os.Stderr, "Computing manifest...\n\n")

	// Serialize to JSON
	var jsonData []byte
	if *pretty {
		jsonData, err = json.MarshalIndent(manifest, "", "  ")
	} else {
		jsonData, err = json.Marshal(manifest)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error serializing manifest: %v\n", err)
		os.Exit(4)
	}

	// Output
	if *output != "" {
		err = os.WriteFile(*output, jsonData, 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error writing to file: %v\n", err)
			os.Exit(5)
		}
		fmt.Fprintf(os.Stderr, "Manifest written to: %s\n", *output)
	} else {
		fmt.Println(string(jsonData))
	}
}