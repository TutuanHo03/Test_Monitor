// server/handlers/amf_context_handler.go
package handlers

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/TutuanHo03/remote-control/models"

	"github.com/gin-gonic/gin"
)

// AmfContextHandler - Handles AMF context navigation and command execution
type AmfContextHandler struct {
	nfHandler    *NFHandler
	rootCommands []models.CommandInfo
}

// NewAmfContextHandler creates a new AMF context handler
func NewAmfContextHandler(nfHandler *NFHandler) *AmfContextHandler {
	handler := &AmfContextHandler{
		nfHandler: nfHandler,
	}

	// Initialize the root commands for AMF context
	handler.initializeCommands()

	return handler
}

// initializeCommands - Initialize commands for AMF context
func (h *AmfContextHandler) initializeCommands() {
	// Basic commands
	h.rootCommands = []models.CommandInfo{
		{
			Name:        "clear",
			Usage:       "Clear the screen",
			Description: "Clear the terminal screen",
		},
		{
			Name:        "disconnect",
			Usage:       "Disconnect from AMF",
			Description: "Disconnect from the AMF server and return to root context",
		},
		{
			Name:        "exit",
			Usage:       "Exit the client",
			Description: "Exit the client application",
		},
		{
			Name:        "help",
			Usage:       "Display help",
			Description: "Show a list of all available AMF commands",
		},
	}

	// Add AMF-specific commands from the NFHandler
	amfCommands := h.nfHandler.commandCache["amf"]
	h.rootCommands = append(h.rootCommands, amfCommands...)
}

// ConnectToAmf handles the initial connection to AMF
func (h *AmfContextHandler) ConnectToAmf(c *gin.Context) {
	var req models.NavigationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.NavigationResponse{
			Error: "Invalid request format: " + err.Error(),
		})
		return
	}

	// Construct client context for AMF connection
	clientContext := models.ClientContext{
		Type:     "amf",
		Name:     "amf",
		NodeType: "amf",
		Commands: h.getCommandNames(),
	}

	// Prepare response
	c.JSON(http.StatusOK, models.NavigationResponse{
		Context:  clientContext,
		Prompt:   ">>> ",
		Message:  "Connected to AMF: http://localhost:6000, type help to see commands",
		Commands: h.rootCommands,
	})
}

// DisconnectFromAmf handles disconnection from AMF
func (h *AmfContextHandler) DisconnectFromAmf(c *gin.Context) {
	var req models.NavigationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.NavigationResponse{
			Error: "Invalid request format: " + err.Error(),
		})
		return
	}

	// Return to root context
	rootContext := models.ClientContext{
		Type:     "root",
		Name:     "root",
		Commands: []string{"clear", "connect", "exit", "help"},
	}

	// Basic commands for root
	rootCommands := []models.CommandInfo{
		{
			Name:        "clear",
			Usage:       "clear the screen",
			Description: "Clear the terminal screen",
		},
		{
			Name:        "connect",
			Usage:       "Connect to a MSsim [connect http://localhost:4000], Connect to AMF [connect http://localhost:6000]",
			Description: "Connect to a server instance",
			ArgsUsage:   "<server-url>",
		},
		{
			Name:        "exit",
			Usage:       "exit the program",
			Description: "Exit the client application",
		},
		{
			Name:        "help",
			Usage:       "display help",
			Description: "Show a list of all available commands",
		},
	}

	c.JSON(http.StatusOK, models.NavigationResponse{
		Context:  rootContext,
		Prompt:   ">>> ",
		Message:  "Disconnect AMF successfully.",
		Commands: rootCommands,
	})
}

// ExecuteCommand handles command execution in AMF context
func (h *AmfContextHandler) ExecuteCommand(c *gin.Context) {
	var req models.CommandRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.CommandResponse{
			Error: "Invalid request format: " + err.Error(),
		})
		return
	}
	// Check if the command is a navigation command
	switch req.CommandText {
	case "help":
		// Generate help text for AMF context
		helpText := h.generateHelpText()
		c.JSON(http.StatusOK, models.CommandResponse{
			Response: helpText,
		})
		return
	case "disconnect":
		// Handle disconnect via NavigationRequest
		navReq := models.NavigationRequest{
			Command: "disconnect",
		}
		c.Request.Body = nil // Reset the body
		c.Set("navRequest", navReq)
		h.DisconnectFromAmf(c)
		return
	case "clear", "exit":
		// These are handled by the client
		c.JSON(http.StatusOK, models.CommandResponse{
			Response: "",
		})
		return
	}

	// Set node type to AMF for all commands
	req.NodeType = "amf"

	// Execute the command using NFHandler
	response, err := h.nfHandler.ExecuteCommand(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.CommandResponse{
			Error: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, response)
}

// GenerateNavigationResponse creates a navigation response for AMF
func (h *AmfContextHandler) GenerateNavigationResponse(message string) models.NavigationResponse {
	clientContext := models.ClientContext{
		Type:     "amf",
		Name:     "amf",
		NodeType: "amf",
		Commands: h.getCommandNames(),
	}

	return models.NavigationResponse{
		Context:  clientContext,
		Prompt:   ">>> ",
		Message:  message,
		Commands: h.rootCommands,
	}
}

// Helper functions

// getCommandNames extracts just the command names from CommandInfo
func (h *AmfContextHandler) getCommandNames() []string {
	cmdNames := make([]string, 0, len(h.rootCommands))
	for _, cmd := range h.rootCommands {
		cmdNames = append(cmdNames, cmd.Name)
	}
	return cmdNames
}

// generateHelpText creates a formatted help text for AMF context
func (h *AmfContextHandler) generateHelpText() string {
	var sb strings.Builder
	sb.WriteString("\nCommands:\n")

	// First show navigation commands
	for _, cmd := range h.rootCommands[:4] { // First 4 are navigation commands
		sb.WriteString(fmt.Sprintf("  %-20s %s\n", cmd.Name, cmd.Usage))
	}

	// Then show AMF-specific commands
	for _, cmd := range h.rootCommands[4:] {
		sb.WriteString(fmt.Sprintf("  %-20s %s\n", cmd.Name, cmd.Usage))
	}

	return sb.String()
}
