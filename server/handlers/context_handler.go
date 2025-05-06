package handlers

import (
	"fmt"
	"net/http"
	"remote-control/models"
	"strings"

	"github.com/gin-gonic/gin"
)

// ContextType defines the various types of contexts in the system
type ContextType string

const (
	RootType       ContextType = "root"
	ServerType     ContextType = "server"
	ContextSetType ContextType = "context_set"
	NodeType       ContextType = "node"
)

// Context - Represents a hierarchical context in the CLI
type Context struct {
	Type        ContextType          // Type of context (root, server, context_set, node)
	Name        string               // Name of this context
	Description string               // Description of this context
	Parent      *Context             // Parent context (if any)
	Commands    []models.CommandInfo // Available commands in this context
	Children    map[string]*Context  // Child contexts
	NodeType    string               // Only for NodeType contexts: type of node (ue, gnb, emulator)
}

// ContextHandler - Manages context navigation and command execution
type ContextHandler struct {
	rootContext  *Context            // Root context of the system
	contextMap   map[string]*Context // Map to store all contexts by path
	commandStore *CommandStore       // Reference to command definitions
}

// NewContextHandler creates a new context handler with initialized contexts
func NewContextHandler(commandStore *CommandStore) *ContextHandler {
	handler := &ContextHandler{
		contextMap:   make(map[string]*Context),
		commandStore: commandStore,
	}

	// Initialize the context hierarchy
	handler.initializeContexts()

	return handler
}

// initializeContexts - Initialize the complete context hierarchy
func (h *ContextHandler) initializeContexts() {
	// Root context
	h.rootContext = &Context{
		Type:        RootType,
		Name:        "root",
		Description: "Root context with basic commands",
		Commands:    h.getBasicCommands(true, false, false),
		Children:    make(map[string]*Context),
	}
	h.contextMap["root"] = h.rootContext

	// Server context
	serverContext := &Context{
		Type:        ServerType,
		Name:        "server",
		Description: "Server connection context",
		Parent:      h.rootContext,
		Commands:    h.getBasicCommands(true, true, true),
		Children:    make(map[string]*Context),
	}
	h.rootContext.Children["server"] = serverContext
	h.contextMap["server"] = serverContext

	// Context sets for different node types
	nodeTypes := []string{"ue", "gnb", "emulator"}
	for _, nodeType := range nodeTypes {
		contextSet := &Context{
			Type:        ContextSetType,
			Name:        nodeType,
			Description: strings.ToUpper(nodeType) + " context set",
			Parent:      serverContext,
			Commands:    h.getContextSetCommands(),
			Children:    make(map[string]*Context),
			NodeType:    nodeType,
		}
		serverContext.Children[nodeType] = contextSet
		h.contextMap[nodeType] = contextSet

		// For emulator, create a direct node context automatically
		if nodeType == "emulator" {
			emulatorNode := &Context{
				Type:        NodeType,
				Name:        "emulator",
				Description: "Emulator control context",
				Parent:      contextSet,
				Commands:    h.commandStore.GetCommandsForNodeType("emulator"),
				Children:    make(map[string]*Context),
				NodeType:    "emulator",
			}
			contextSet.Children["emulator"] = emulatorNode
			h.contextMap["emulator:emulator"] = emulatorNode
		}
	}
}

// getBasicCommands returns the standard commands for a context
func (h *ContextHandler) getBasicCommands(includeHelp bool, includeBack bool, includeDisconnect bool) []models.CommandInfo {
	var commands []models.CommandInfo

	if includeHelp {
		commands = append(commands, models.CommandInfo{
			Name:        "help",
			Usage:       "Display available commands",
			Description: "Show a list of all available commands in the current context",
		})
	}

	commands = append(commands, models.CommandInfo{
		Name:        "clear",
		Usage:       "Clear the screen",
		Description: "Clear the terminal screen",
	})

	commands = append(commands, models.CommandInfo{
		Name:        "exit",
		Usage:       "Exit the program",
		Description: "Exit the client application",
	})

	if includeBack {
		commands = append(commands, models.CommandInfo{
			Name:        "back",
			Usage:       "Go back to previous context",
			Description: "Navigate back to the parent context",
		})
	}

	if includeDisconnect {
		commands = append(commands, models.CommandInfo{
			Name:        "disconnect",
			Usage:       "Disconnect from server",
			Description: "Disconnect from the current server and return to root context",
		})
	}

	// Add use command for server context
	if includeDisconnect {
		commands = append(commands, models.CommandInfo{
			Name:        "use",
			Usage:       "Select a context to use [use emulator | ue | gnb]",
			Description: "Navigate to a specific context type",
			ArgsUsage:   "<context-type>",
		})
	}

	return commands
}

// getContextSetCommands returns commands specific to context sets
func (h *ContextHandler) getContextSetCommands() []models.CommandInfo {
	commands := h.getBasicCommands(true, true, false)

	// Add select command for context sets
	commands = append(commands, models.CommandInfo{
		Name:        "select",
		Usage:       "Select a node to interact with [select <node-name>]",
		Description: "Navigate to a specific node in this context set",
		ArgsUsage:   "<node-name>",
	})

	return commands
}

// FindOrCreateNodeContext finds an existing node context or creates one if it doesn't exist
func (h *ContextHandler) FindOrCreateNodeContext(nodeType string, nodeName string) *Context {
	contextKey := nodeType + ":" + nodeName

	// Check if context already exists
	if ctx, exists := h.contextMap[contextKey]; exists {
		return ctx
	}

	// Find parent context
	parentCtx, exists := h.contextMap[nodeType]
	if !exists {
		return nil
	}

	// Create new node context
	nodeContext := &Context{
		Type:        NodeType,
		Name:        nodeName,
		Description: fmt.Sprintf("%s node of type %s", nodeName, nodeType),
		Parent:      parentCtx,
		Commands:    h.commandStore.GetCommandsForNodeType(nodeType),
		Children:    make(map[string]*Context),
		NodeType:    nodeType,
	}

	h.contextMap[contextKey] = nodeContext
	parentCtx.Children[nodeName] = nodeContext

	return nodeContext
}

// NavigateContext handles navigation between contexts
func (h *ContextHandler) NavigateContext(c *gin.Context) {
	var req models.NavigationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.NavigationResponse{
			Error: "Invalid request format: " + err.Error(),
		})
		return
	}

	// Find the current context
	var currentCtx *Context
	var exists bool

	// Process node contexts
	contextKey := req.CurrentContext
	if req.NodeType != "" && req.CurrentContext != "" {
		// Chỉ sử dụng prefix cho node contexts
		if req.NodeType != req.CurrentContext { // Check for the case where they are the same
			contextKey = req.NodeType + ":" + req.CurrentContext
		}
	}

	// FInd context
	if req.CurrentContext != "" && req.CurrentContext != "root" {
		currentCtx, exists = h.contextMap[contextKey]

		if !exists {
			currentCtx, exists = h.contextMap[req.CurrentContext]
		}

		if !exists {
			fmt.Printf("Context not found: %s (tried keys: %s, %s)\n",
				req.CurrentContext, contextKey, req.CurrentContext)
			fmt.Printf("Available contexts: %v\n", h.getContextKeys())
		}
	} else if req.CurrentContext == "root" {
		currentCtx = h.rootContext
		exists = true
	}

	if !exists && req.CurrentContext != "root" {
		c.JSON(http.StatusBadRequest, models.NavigationResponse{
			Error: fmt.Sprintf("Current context not found: %s", req.CurrentContext),
		})
		return
	}

	var newCtx *Context
	var message string
	var cmdInfos []models.CommandInfo

	// Process navigation command
	switch req.Command {
	case "connect":
		if len(req.Args) < 1 {
			c.JSON(http.StatusBadRequest, models.NavigationResponse{
				Error: "URL is required for connect command",
			})
			return
		}
		serverURL := req.Args[0]
		newCtx = h.contextMap["server"]
		message = fmt.Sprintf("Connected to server: %s, type help to see commands", serverURL)

	case "disconnect":
		newCtx = h.rootContext
		message = "Disconnected from server"

	case "back":
		if currentCtx != nil && currentCtx.Parent != nil {
			newCtx = currentCtx.Parent
			if newCtx.Type == ServerType {
				message = "Back to server context"
			} else {
				message = fmt.Sprintf("Back to %s context", newCtx.Name)
			}
		} else {
			c.JSON(http.StatusBadRequest, models.NavigationResponse{
				Error: "Already at root context",
			})
			return
		}

	case "use":
		if len(req.Args) < 1 {
			c.JSON(http.StatusBadRequest, models.NavigationResponse{
				Error: "Context type is required for use command",
			})
			return
		}
		contextType := req.Args[0]

		// Check if context type exists
		childCtx, exists := h.contextMap[contextType]
		if !exists || childCtx.Type != ContextSetType {
			c.JSON(http.StatusBadRequest, models.NavigationResponse{
				Error: "Invalid context type. Use 'emulator', 'ue', or 'gnb'",
			})
			return
		}

		newCtx = childCtx

		// Special handling for emulator
		if contextType == "emulator" {
			emulatorNode := h.FindOrCreateNodeContext("emulator", "emulator")
			newCtx = emulatorNode
			message = "Switched to emulator context"
		} else {
			// For UE/GNB, list available objects
			objects, err := h.commandStore.GetObjectsOfType(contextType)
			if err != nil {
				c.JSON(http.StatusInternalServerError, models.NavigationResponse{
					Error: "Failed to get objects: " + err.Error(),
				})
				return
			}

			message = fmt.Sprintf("Available %s objects:\n", contextType)
			for _, obj := range objects {
				message += fmt.Sprintf("  - %s\n", obj)
			}
		}

	case "select":
		if len(req.Args) < 1 {
			c.JSON(http.StatusBadRequest, models.NavigationResponse{
				Error: "Node name is required for select command",
			})
			return
		}

		nodeName := req.Args[0]
		if currentCtx == nil || currentCtx.Type != ContextSetType {
			c.JSON(http.StatusBadRequest, models.NavigationResponse{
				Error: "Can only select nodes from a context set",
			})
			return
		}

		nodeType := currentCtx.NodeType

		objects, err := h.commandStore.GetObjectsOfType(nodeType)
		if err != nil {
			c.JSON(http.StatusInternalServerError, models.NavigationResponse{
				Error: "Failed to get objects: " + err.Error(),
			})
			return
		}

		nodeExists := false
		for _, obj := range objects {
			if obj == nodeName {
				nodeExists = true
				break
			}
		}

		if !nodeExists {
			c.JSON(http.StatusBadRequest, models.NavigationResponse{
				Error: fmt.Sprintf("Node '%s' not found", nodeName),
			})
			return
		}

		// Find or create node context
		nodeCtx := h.FindOrCreateNodeContext(nodeType, nodeName)
		newCtx = nodeCtx
		message = fmt.Sprintf("Selected node: %s", nodeName)

		cmdInfos = h.commandStore.GetCommandsForNodeType(nodeType)

		// Create client context
		clientContext := h.createClientContext(nodeCtx)

		c.JSON(http.StatusOK, models.NavigationResponse{
			Context:  clientContext,
			Prompt:   nodeName + " >>> ",
			Message:  message,
			Commands: cmdInfos,
		})
		return

	default:
		c.JSON(http.StatusBadRequest, models.NavigationResponse{
			Error: fmt.Sprintf("Unknown navigation command: %s", req.Command),
		})
		return
	}

	// Create client context from server context
	clientContext := h.createClientContext(newCtx)

	// Create appropriate prompt
	var prompt string
	if newCtx.Type == RootType || newCtx.Type == ServerType {
		prompt = ">>> "
	} else {
		prompt = newCtx.Name + " >>> "
	}

	c.JSON(http.StatusOK, models.NavigationResponse{
		Context:  clientContext,
		Prompt:   prompt,
		Message:  message,
		Commands: cmdInfos,
	})
}

// findContext retrieves a context from the context map
func (h *ContextHandler) findContext(path string, nodeType string) (*Context, bool) {
	if nodeType != "" && path != "" && nodeType != path {
		contextKey := nodeType + ":" + path
		if ctx, exists := h.contextMap[contextKey]; exists {
			return ctx, true
		}
	}

	if ctx, exists := h.contextMap[path]; exists {
		return ctx, true
	}
	// If path is "root", return the root context
	if path == "root" {
		return h.rootContext, true
	}

	return nil, false
}

// createClientContext converts a server context to a client context
func (h *ContextHandler) createClientContext(ctx *Context) models.ClientContext {
	if ctx == nil {
		return models.ClientContext{}
	}

	// Create command lists từ CommandInfo
	commandNames := make([]string, len(ctx.Commands))
	for i, cmd := range ctx.Commands {
		commandNames[i] = cmd.Name
	}

	return models.ClientContext{
		Type:     string(ctx.Type),
		Name:     ctx.Name,
		NodeType: ctx.NodeType,
		Commands: commandNames,
	}
}

// GetContextByPath retrieves context information by path
func (h *ContextHandler) GetContextByPath(c *gin.Context) {
	path := c.Param("path")

	ctx, exists := h.contextMap[path]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Context not found",
		})
		return
	}

	clientContext := h.createClientContext(ctx)

	// Get parent path
	var parentPath string
	if ctx.Parent != nil {
		parentPath = ctx.Parent.Name
	}

	// Get children paths
	childrenPaths := make([]string, 0, len(ctx.Children))
	for name := range ctx.Children {
		childrenPaths = append(childrenPaths, name)
	}

	c.JSON(http.StatusOK, gin.H{
		"context":       clientContext,
		"description":   ctx.Description,
		"parentPath":    parentPath,
		"childrenPaths": childrenPaths,
	})
}

// GetContextCommands returns the available commands for a context
func (h *ContextHandler) GetContextCommands(c *gin.Context) {
	contextPath := c.Param("path")

	ctx, exists := h.findContext(contextPath, "")
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Context not found",
		})
		return
	}

	c.JSON(http.StatusOK, ctx.Commands)
}

// GetNodeCommands returns commands for a specific node
func (h *ContextHandler) GetNodeCommands(c *gin.Context) {
	nodeType := c.Param("type")
	nodeName := c.Param("name")

	contextKey := nodeType + ":" + nodeName
	ctx, exists := h.contextMap[contextKey]

	if !exists {
		// Try to get commands without a context
		commands := h.commandStore.GetCommandsForNodeType(nodeType)
		if len(commands) > 0 {
			c.JSON(http.StatusOK, commands)
			return
		}

		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Node context not found",
		})
		return
	}

	c.JSON(http.StatusOK, ctx.Commands)
}

// ExecuteCommand handles command execution requests
func (h *ContextHandler) ExecuteCommand(c *gin.Context) {
	var req models.CommandRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.CommandResponse{
			Error: "Invalid request format: " + err.Error(),
		})
		return
	}

	// Execute the command via command store
	response, err := h.commandStore.ExecuteCommand(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.CommandResponse{
			Error: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, response)
}

// Add helper function to debug
func (h *ContextHandler) getContextKeys() []string {
	keys := make([]string, 0, len(h.contextMap))
	for k := range h.contextMap {
		keys = append(keys, k)
	}
	return keys
}
