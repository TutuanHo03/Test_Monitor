package main

import (
	"flag"
	"fmt"
	"os"
	"test_monitor/client"
)

func main() {
	var (
		version = flag.Bool("version", false, "Show version information")
		noColor = flag.Bool("no-color", false, "Disable color output")
	)

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nConnect to a server by typing: connect http://localhost:4000\n")
	}

	flag.Parse()

	if *version {
		fmt.Println("Test_Monitor CLI Client v1.0.0")
		os.Exit(0)
	}

	cli := client.NewClient()

	if *noColor {
		fmt.Println("Warning: no-color flag is set but not yet implemented")
	}

	fmt.Println("\nInteractive CLI Client")
	fmt.Println("Type 'help' to see available commands")
	fmt.Println("Type 'connect http://localhost:4000' to connect to a server")

	// Chạy client ở chế độ tương tác
	cli.Run()
}
