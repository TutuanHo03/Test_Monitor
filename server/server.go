package server

import (
	"fmt"
	"log"

	"github.com/TutuanHo03/remote-control/server/handlers"

	"github.com/gin-gonic/gin"
)

type ServerConfig struct {
	Port    string
	Host    string
	AmfPort string
}

type Server struct {
	mssimRouter   *gin.Engine
	amfRouter     *gin.Engine
	config        ServerConfig
	cmdHandler    *handlers.CommandStore
	nfHandler     *handlers.NFHandler
	ctxHandler    *handlers.ContextHandler
	amfCtxHandler *handlers.AmfContextHandler
}

func NewServer(config ServerConfig, eApi handlers.EmulatorApi, uApi handlers.UeApi, gApi handlers.GnbApi, aApi handlers.AmfApi) *Server {
	if config.Port == "" {
		config.Port = "4000"
	}
	if config.Host == "" {
		config.Host = "0.0.0.0"
	}
	if config.AmfPort == "" {
		config.AmfPort = "6000" // Default AMF port
	}

	r := gin.Default()
	amfR := gin.Default()
	cmdHandler := handlers.NewCommandStore(eApi, uApi, gApi)
	nfHandler := handlers.NewNFHandler(aApi)
	ctxHandler := handlers.NewContextHandler(cmdHandler)
	amfCtxHandler := handlers.NewAmfContextHandler(nfHandler)
	server := &Server{
		mssimRouter:   r,
		amfRouter:     amfR,
		config:        config,
		cmdHandler:    cmdHandler,
		nfHandler:     nfHandler,
		ctxHandler:    ctxHandler,
		amfCtxHandler: amfCtxHandler,
	}

	server.setupRoutes()
	server.setupAmfRoutes()
	return server
}

func (s *Server) setupRoutes() {
	// CORS middleware
	s.mssimRouter.Use(func(c *gin.Context) {
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
	s.mssimRouter.GET("/api/context", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "Context API ready",
		})
	})

	s.mssimRouter.GET("/api/context/path/:path", s.ctxHandler.GetContextByPath)
	s.mssimRouter.GET("/api/context/commands/:path", s.ctxHandler.GetContextCommands)
	s.mssimRouter.GET("/api/context/node/:type", func(c *gin.Context) {
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
	s.mssimRouter.GET("/api/context/node/:type/:name/commands", s.ctxHandler.GetNodeCommands)

	s.mssimRouter.POST("/api/context/navigate", s.ctxHandler.NavigateContext)
	s.mssimRouter.POST("/api/exec", s.ctxHandler.ExecuteCommand)
}

func (s *Server) setupAmfRoutes() {
	// CORS middleware for AMF
	s.amfRouter.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	// API status
	s.amfRouter.GET("/api/status", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "AMF API ready",
			"service": "AMF",
		})
	})

	// AMF context navigation
	s.amfRouter.POST("/api/context/navigate", s.amfCtxHandler.ConnectToAmf)

	// Command execution for AMF
	s.amfRouter.POST("/api/exec", s.amfCtxHandler.ExecuteCommand)

}

func (s *Server) Start() error {
	// Start the MSsim server in a goroutine
	go func() {
		address := fmt.Sprintf("%s:%s", s.config.Host, s.config.Port)
		log.Printf("Starting MSsim server on %s", address)
		if err := s.mssimRouter.Run(address); err != nil {
			log.Printf("Failed to start MSsim server: %v", err)
		}
	}()

	// Start the AMF server
	amfAddress := fmt.Sprintf("%s:%s", s.config.Host, s.config.AmfPort)
	log.Printf("Starting AMF server on %s", amfAddress)
	return s.amfRouter.Run(amfAddress)
}

func (s *Server) Shutdown() {
	log.Println("Cleaning up resources...")
}
