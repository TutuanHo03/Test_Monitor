package main

import (
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
	cmdHandler := handlers.NewCommandStore(eApi, uApi, gApi)
	ctxHandler := handlers.NewContextHandler(cmdHandler)

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

	// API routes
	r.GET("/api/context", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "Context API ready",
		})
	})

	r.GET("/api/context/path/:path", ctxHandler.GetContextByPath)
	r.GET("/api/context/commands/:path", ctxHandler.GetContextCommands)
	r.GET("/api/context/node/:type", func(c *gin.Context) {
		nodeType := c.Param("type")
		objects, err := cmdHandler.GetObjectsOfType(nodeType)
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, gin.H{
			"type":    nodeType,
			"objects": objects,
		})
	})
	r.GET("/api/context/node/:type/:name/commands", ctxHandler.GetNodeCommands)

	r.POST("/api/context/navigate", ctxHandler.NavigateContext)
	r.POST("/api/exec", ctxHandler.ExecuteCommand)

	log.Println("Starting server on :4000")
	r.Run(":4000")
}
