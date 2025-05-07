package server

import (
	"fmt"
	"log"

	"github.com/TutuanHo03/remote-control/server/handlers"

	"github.com/gin-gonic/gin"
)

type ServerConfig struct {
	Port string
	Host string
}

type Server struct {
	router     *gin.Engine
	config     ServerConfig
	cmdHandler *handlers.CommandStore
	ctxHandler *handlers.ContextHandler
}

func NewServer(config ServerConfig, eApi handlers.EmulatorApi, uApi handlers.UeApi, gApi handlers.GnbApi) *Server {
	if config.Port == "" {
		config.Port = "4000"
	}
	if config.Host == "" {
		config.Host = "0.0.0.0"
	}

	r := gin.Default()
	cmdHandler := handlers.NewCommandStore(eApi, uApi, gApi)
	ctxHandler := handlers.NewContextHandler(cmdHandler)

	server := &Server{
		router:     r,
		config:     config,
		cmdHandler: cmdHandler,
		ctxHandler: ctxHandler,
	}

	server.setupRoutes()
	return server
}

func (s *Server) setupRoutes() {
	// CORS middleware
	s.router.Use(func(c *gin.Context) {
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
	s.router.GET("/api/context", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "Context API ready",
		})
	})

	s.router.GET("/api/context/path/:path", s.ctxHandler.GetContextByPath)
	s.router.GET("/api/context/commands/:path", s.ctxHandler.GetContextCommands)
	s.router.GET("/api/context/node/:type", func(c *gin.Context) {
		nodeType := c.Param("type")
		objects, err := s.cmdHandler.GetObjectsOfType(nodeType)
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, gin.H{
			"type":    nodeType,
			"objects": objects,
		})
	})
	s.router.GET("/api/context/node/:type/:name/commands", s.ctxHandler.GetNodeCommands)

	s.router.POST("/api/context/navigate", s.ctxHandler.NavigateContext)
	s.router.POST("/api/exec", s.ctxHandler.ExecuteCommand)
}

func (s *Server) Start() error {
	address := fmt.Sprintf("%s:%s", s.config.Host, s.config.Port)
	return s.router.Run(address)
}

func (s *Server) Shutdown() {
	log.Println("Cleaning up resources...")
}
