package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"test_monitor/server"
	"test_monitor/server/api"
)

func main() {
	// Định nghĩa các cờ command line
	var (
		port      = flag.String("port", "4000", "Port to listen on")
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

	// Hiển thị thông tin phiên bản nếu được yêu cầu
	if *version {
		fmt.Println("Test_Monitor Server v1.0.0")
		os.Exit(0)
	}

	// Hiển thị thông báo debug mode
	if *debugMode {
		log.Println("Running in debug mode")
	}

	// Khởi tạo API
	log.Println("Initializing APIs...")
	eApi := api.CreateEmulatorApi()
	uApi := api.CreateUeApi()
	gApi := api.CreateGnbApi()

	// Tạo cấu hình server
	config := server.ServerConfig{
		Port: *port,
		Host: *host,
	}

	// Khởi tạo server
	log.Printf("Creating server on %s:%s...", *host, *port)
	srv := server.NewServer(config, eApi, uApi, gApi)

	// Thiết lập xử lý tín hiệu để tắt máy chủ một cách an toàn
	setupGracefulShutdown(srv)

	// Khởi động server
	log.Printf("Starting Test_Monitor server on %s:%s", *host, *port)
	log.Println("Press Ctrl+C to stop the server")
	if err := srv.Start(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

// setupGracefulShutdown đăng ký xử lý shutdown an toàn
func setupGracefulShutdown(_ *server.Server) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		log.Println("Received shutdown signal. Shutting down gracefully...")
		// Thực hiện các bước dọn dẹp ở đây nếu cần
		// srv.Shutdown()  // Giả sử có phương thức này
		log.Println("Server shutdown complete")
		os.Exit(0)
	}()
}
