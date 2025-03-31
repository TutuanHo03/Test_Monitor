package main

import (
	"fmt"
	"log"
	"test_monitor/server/api"
	"test_monitor/server/handlers"

	"github.com/gin-gonic/gin"
)

func main() {
	eApi := api.CreateEmulatorApi()
	uApi := api.CreateUeApi()
	gApi := api.CreateGnbApi()

	r := gin.Default()

	// Initialize handlers
	cmdHandler := handlers.NewCommandHandler(eApi, uApi, gApi)
	ctxHandler := handlers.NewContextHandler(eApi)

	// CORS middleware
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	// API routes emulator
	emulatorGroup := r.Group("/api/emulator")
	{
		emulatorGroup.GET("/commands", cmdHandler.GetEmulatorCommands)
		emulatorGroup.GET("/ues", cmdHandler.ListUes)
		emulatorGroup.GET("/gnbs", cmdHandler.ListGnbs)
	}

	// API routes UE context
	ueGroup := r.Group("/api/ue/:ueId")
	{
		ueGroup.GET("/commands", cmdHandler.GetUeCommands)
	}

	// API routes GNB context
	gnbGroup := r.Group("/api/gnb/:gnbId")
	{
		gnbGroup.GET("/commands", cmdHandler.GetGnbCommands)
	}

	// API to process command overall
	r.POST("/api/exec", cmdHandler.ExecuteCommand)

	// API context handling
	r.GET("/api/context", ctxHandler.GetAvailableContexts)
	r.GET("/api/context/:type", ctxHandler.GetContextByType)

	fmt.Println("Server started on :4000")
	if err := r.Run(":4000"); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
