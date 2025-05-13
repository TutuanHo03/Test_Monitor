package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/TutuanHo03/remote-control/server"
	"github.com/TutuanHo03/remote-control/server/api"
)

func main() {
	var (
		port      = flag.String("port", "4000", "Port to listen on for MSsim server")
		amfPort   = flag.String("amf-port", "6000", "Port to listen on for AMF server")
		host      = flag.String("host", "0.0.0.0", "Host to bind to")
		version   = flag.Bool("version", false, "Show version information")
		debugMode = flag.Bool("debug", false, "Enable debug mode")
	)

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	if *version {
		fmt.Println("Test_Monitor Server v1.0.0")
		os.Exit(0)
	}

	if *debugMode {
		log.Println("Running in debug mode")
	}

	log.Println("Initializing APIs...")
	eApi := api.CreateEmulatorApi()
	uApi := api.CreateUeApi()
	gApi := api.CreateGnbApi()
	aApi := api.CreateAmfApi()

	config := server.ServerConfig{
		Port:    *port,
		Host:    *host,
		AmfPort: *amfPort,
	}

	log.Printf("Creating server on %s:%s (MSsim) and %s:%s (AMF)...", *host, *port, *host, *amfPort)
	srv := server.NewServer(config, eApi, uApi, gApi, aApi)

	setupGracefulShutdown(srv)

	log.Printf("Starting servers:")
	log.Printf("- MSsim server on %s:%s", *host, *port)
	log.Printf("- AMF server on %s:%s", *host, *amfPort)
	log.Println("Press Ctrl+C to stop the servers")
	if err := srv.Start(); err != nil {
		log.Fatalf("Failed to start servers: %v", err)
	}
}

func setupGracefulShutdown(srv *server.Server) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		log.Println("Received shutdown signal. Shutting down gracefully...")
		srv.Shutdown()
		log.Println("Server shutdown complete")
		os.Exit(0)
	}()
}
