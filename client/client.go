package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/TutuanHo03/remote-control/models"

	"github.com/abiosoft/ishell"
)

// Client represents the CLI client interface
type Client struct {
	shell        *ishell.Shell
	serverURL    string
	contextStack []models.ClientContext
	isAmfMode    bool // Track if we're in AMF mode
}

// NewClient creates and initializes a new CLI client
func NewClient() *Client {
	client := &Client{
		shell: ishell.New(),
		contextStack: []models.ClientContext{
			{
				Type:     "root",
				Name:     "root",
				Commands: []string{"help", "clear", "exit", "connect"},
			},
		},
		isAmfMode: false,
	}

	client.setupCommands("root")
	return client
}

// Run starts the interactive shell
func (c *Client) Run() {
	c.shell.Println("Interactive CLI Client")
	c.shell.SetPrompt(">>> ")
	c.shell.Run()
}

func (c *Client) ConnectWithHostAndPort(host string, port string) {
	if host == "" {
		host = "localhost"
	}
	url := fmt.Sprintf("http://%s:%s", host, port)
	c.ConnectToServer(url)
}

func (c *Client) ConnectWithPort(port string) {
	c.ConnectWithHostAndPort("localhost", port)
}

// setupCommands sets up the commands for the shell based on the context
func (c *Client) setupCommands(contextType string) {
	// Clear existing commands to avoid duplicates
	for _, cmd := range []string{"help", "clear", "exit", "back", "disconnect", "use", "select", "connect"} {
		c.shell.DeleteCmd(cmd)
	}

	// Add basic commands
	c.shell.AddCmd(&ishell.Cmd{
		Name: "help",
		Help: "display help",
		Func: c.displayHelp(),
	})

	c.shell.AddCmd(&ishell.Cmd{
		Name: "clear",
		Help: "clear the screen",
		Func: func(ctx *ishell.Context) {
			ctx.ClearScreen()
		},
	})

	c.shell.AddCmd(&ishell.Cmd{
		Name: "exit",
		Help: "exit the program",
		Func: func(ctx *ishell.Context) {
			ctx.Println("Goodbye!")
			os.Exit(0)
		},
	})

	// Check if we're in AMF mode
	if c.isAmfMode {
		c.setupAmfCommands()
		return
	}

	// Context-specific commands for non-AMF mode
	switch contextType {
	case "root":
		c.shell.AddCmd(&ishell.Cmd{
			Name:     "connect",
			Help:     "Connect to a MSsim [connect http://localhost:4000], Connect to AMF [connect http://localhost:6000]",
			LongHelp: "Connect to a server using URL. Example: connect http://localhost:4000 or connect http://localhost:6000 for AMF",
			Func: func(ctx *ishell.Context) {
				if len(ctx.Args) < 1 {
					ctx.Println("Usage: connect <server-url>")
					return
				}
				url := ctx.Args[0]
				c.ConnectToServer(url)
			},
		})

	case "server":
		c.shell.AddCmd(&ishell.Cmd{
			Name: "back",
			Help: "Go back to previous context",
			Func: func(ctx *ishell.Context) {
				c.navigateContext("back", nil)
			},
		})

		c.shell.AddCmd(&ishell.Cmd{
			Name: "disconnect",
			Help: "Disconnect from server",
			Func: func(ctx *ishell.Context) {
				c.navigateContext("disconnect", nil)
			},
		})

		c.shell.AddCmd(&ishell.Cmd{
			Name: "use",
			Help: "Select a context to use [use emulator | ue | gnb]",
			Func: func(ctx *ishell.Context) {
				if len(ctx.Args) < 1 {
					ctx.Println("Usage: use <context-type>")
					ctx.Println("Context types: emulator, ue, gnb")
					return
				}
				c.navigateContext("use", ctx.Args)
			},
		})

	case "context_set":
		c.shell.AddCmd(&ishell.Cmd{
			Name: "back",
			Help: "Go back to previous context",
			Func: func(ctx *ishell.Context) {
				c.navigateContext("back", nil)
			},
		})

		c.shell.AddCmd(&ishell.Cmd{
			Name: "disconnect",
			Help: "Disconnect from server",
			Func: func(ctx *ishell.Context) {
				c.navigateContext("disconnect", nil)
			},
		})

		c.shell.AddCmd(&ishell.Cmd{
			Name: "select",
			Help: "Select a node to interact with [select <node-name>]",
			Func: func(ctx *ishell.Context) {
				if len(ctx.Args) < 1 {
					ctx.Println("Usage: select <node-name>")
					return
				}
				c.navigateContext("select", ctx.Args)
			},
		})

	case "node":
		c.shell.AddCmd(&ishell.Cmd{
			Name: "back",
			Help: "Go back to previous context",
			Func: func(ctx *ishell.Context) {
				c.navigateContext("back", nil)
			},
		})

		c.shell.AddCmd(&ishell.Cmd{
			Name: "disconnect",
			Help: "Disconnect from server",
			Func: func(ctx *ishell.Context) {
				c.navigateContext("disconnect", nil)
			},
		})
	}
}

// setupAmfCommands sets up the AMF-specific commands
func (c *Client) setupAmfCommands() {
	// Add AMF disconnect command
	c.shell.AddCmd(&ishell.Cmd{
		Name: "disconnect",
		Help: "Disconnect AMF",
		Func: func(ctx *ishell.Context) {
			c.disconnectAmf()
		},
	})

	// Add AMF-specific commands
	amfCommands := c.requestAmfCommands()
	for _, cmdInfo := range amfCommands {
		info := cmdInfo
		c.shell.AddCmd(&ishell.Cmd{
			Name:     info.Name,
			Help:     info.Usage,
			LongHelp: c.generateLongHelp(info),
			Func: func(ctx *ishell.Context) {
				result, err := c.execAmfCmd(info.Name, ctx.Args)
				if err != nil {
					ctx.Printf("Error: %v\n", err)
					return
				}
				ctx.Println(result)
			},
		})
	}
}

// requestAmfCommands fetches AMF command definitions from the server
func (c *Client) requestAmfCommands() []models.CommandInfo {
	if c.serverURL == "" {
		return nil
	}

	resp, err := http.Get(c.serverURL + "/api/status")
	if err != nil || resp.StatusCode != http.StatusOK {
		// If status check fails, try a few default commands
		return c.getDefaultAmfCommands()
	}

	// Get the actual commands
	jsonData, err := json.Marshal(models.NavigationRequest{
		Command: "help",
	})
	if err != nil {
		return c.getDefaultAmfCommands()
	}

	resp, err = http.Post(c.serverURL+"/api/context/navigate", "application/json", bytes.NewBuffer(jsonData))
	if err != nil || resp.StatusCode != http.StatusOK {
		return c.getDefaultAmfCommands()
	}
	defer resp.Body.Close()

	var response models.NavigationResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return c.getDefaultAmfCommands()
	}

	return response.Commands
}

// getDefaultAmfCommands returns default AMF commands if we can't get them from the server
func (c *Client) getDefaultAmfCommands() []models.CommandInfo {
	return []models.CommandInfo{
		{Name: "list-ue", Usage: "List UE Context"},
		{Name: "register-ue", Usage: "Register a UE to the core network", ArgsUsage: "<imsi>"},
		{Name: "deregister-ue", Usage: "Deregister a UE from the core network", ArgsUsage: "<imsi>"},
		{Name: "status", Usage: "Get AMF service status"},
		{Name: "config", Usage: "Get AMF configuration"},
		{Name: "send-n1n2-message", Usage: "Send N1/N2 message to a UE", ArgsUsage: "<ue-id> <message-type> <content>"},
		{Name: "list-n1n2-subscriptions", Usage: "List N1/N2 message subscriptions", ArgsUsage: "<ue-id>"},
		{Name: "initiate-handover", Usage: "Initiate handover for a UE", ArgsUsage: "<ue-id> <target-gnb>"},
		{Name: "handover-history", Usage: "Show handover history for a UE", ArgsUsage: "<ue-id>"},
		{Name: "nf-subscriptions", Usage: "List NF subscriptions"},
		{Name: "sbi-endpoints", Usage: "List SBI endpoints"},
	}
}

// connectToServer handles server connection
func (c *Client) ConnectToServer(url string) {
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		url = "http://" + url
	}

	c.serverURL = url

	// Check if this is an AMF connection (port 6000)
	isAmfConnection := strings.Contains(url, ":6000")

	if isAmfConnection {
		c.connectToAmf(url)
	} else {
		// Regular MSsim connection
		resp, err := http.Get(url + "/api/context")
		if err != nil {
			c.shell.Printf("Failed to connect to server: %v\n", err)
			c.serverURL = "" // Reset if failing
			return
		}
		defer resp.Body.Close()

		c.navigateContext("connect", []string{url})
	}
}

// connectToAmf handles AMF connection
func (c *Client) connectToAmf(url string) {
	// Check if AMF server is reachable
	resp, err := http.Get(url + "/api/status")
	if err != nil {
		c.shell.Printf("Failed to connect to AMF: %v\n", err)
		c.serverURL = "" // Reset if failing
		return
	}
	defer resp.Body.Close()

	// Set AMF mode
	c.isAmfMode = true

	// Send connection request to AMF
	jsonData, err := json.Marshal(models.NavigationRequest{
		Command: "connect",
	})
	if err != nil {
		c.shell.Printf("Error preparing request: %v\n", err)
		return
	}

	resp, err = http.Post(url+"/api/context/navigate", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		c.shell.Printf("Error communicating with AMF server: %v\n", err)
		c.isAmfMode = false
		return
	}
	defer resp.Body.Close()

	// Process response
	var response models.NavigationResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		c.shell.Printf("Error parsing response: %v\n", err)
		c.isAmfMode = false
		return
	}

	// Update context
	amfContext := models.ClientContext{
		Type:      "amf",
		Name:      "amf",
		NodeType:  "amf",
		Commands:  getCommandNames(response.Commands),
		ServerURL: url,
	}
	c.contextStack = []models.ClientContext{
		c.contextStack[0], // Keep root
		amfContext,        // Add AMF
	}

	// Update prompt
	c.shell.SetPrompt(">>> ")

	// Display connection message
	if response.Message != "" {
		c.shell.Println(response.Message)
	} else {
		c.shell.Println("Connected to AMF: " + url + ", type help to see commands")
	}

	// Setup AMF commands
	c.setupAmfCommands()
}

// disconnectAmf handles disconnection from AMF
func (c *Client) disconnectAmf() {
	if !c.isAmfMode || c.serverURL == "" {
		return
	}

	// Send disconnect request
	jsonData, err := json.Marshal(models.NavigationRequest{
		Command: "disconnect",
	})
	if err != nil {
		c.shell.Printf("Error preparing request: %v\n", err)
		return
	}

	resp, err := http.Post(c.serverURL+"/api/context/navigate", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		c.shell.Printf("Error communicating with AMF server: %v\n", err)
		// Continue with local disconnect even if server request fails
	} else {
		defer resp.Body.Close()

		// Process response
		var response models.NavigationResponse
		if err := json.NewDecoder(resp.Body).Decode(&response); err == nil && response.Message != "" {
			c.shell.Println(response.Message)
		} else {
			c.shell.Println("Disconnect AMF successfully.")
		}
	}

	// Reset to root context
	c.contextStack = c.contextStack[:1] // Keep only root
	c.serverURL = ""
	c.isAmfMode = false

	// Reset commands
	c.setupCommands("root")
}

// execAmfCmd executes an AMF-specific command
func (c *Client) execAmfCmd(cmdName string, args []string) (string, error) {
	if !c.isAmfMode || c.serverURL == "" {
		return "", fmt.Errorf("not connected to AMF")
	}

	// Create command request
	cmdReq := models.CommandRequest{
		NodeType:    "amf",
		NodeName:    "amf",
		CommandText: cmdName,
		CommandPath: cmdName,
		Args:        args,
	}

	// Send the command
	jsonData, err := json.Marshal(cmdReq)
	if err != nil {
		return "", fmt.Errorf("failed to marshal command request: %v", err)
	}

	resp, err := http.Post(c.serverURL+"/api/exec", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to send command: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %v", err)
	}

	var response models.CommandResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return "", fmt.Errorf("failed to parse response: %v\nresponse body: %s", err, string(body))
	}

	if response.Error != "" {
		return "", fmt.Errorf("server error: %s", response.Error)
	}

	return response.Response, nil
}

// displayHelp generates help text for the current context
func (c *Client) displayHelp() func(*ishell.Context) {
	return func(ctx *ishell.Context) {
		if c.isAmfMode {
			// AMF-specific help
			ctx.Println("\nCommands:")
			ctx.Println("  clear                Clear the screen")
			ctx.Println("  disconnect           Disconnect AMF")
			ctx.Println("  exit                 Exit the client")
			ctx.Println("  help                 Display help")

			// Get AMF commands
			amfCommands := c.requestAmfCommands()
			for _, cmd := range amfCommands {
				if cmd.Name != "clear" && cmd.Name != "disconnect" && cmd.Name != "exit" && cmd.Name != "help" {
					ctx.Printf("  %-20s %s\n", cmd.Name, cmd.Usage)
				}
			}
			return
		}

		// Standard help for other contexts
		currentContext := c.getCurrentContext()

		switch currentContext.Type {
		case "root":
			ctx.Println("Commands:")
			ctx.Println("  clear        clear the screen")
			ctx.Println("  connect      Connect to a MSsim [connect http://localhost:4000], Connect to AMF [connect http://localhost:6000]")
			ctx.Println("  exit         exit the program")
			ctx.Println("  help         display help")

		case "server":
			ctx.Println("Commands:")
			ctx.Println("  back                Go back to previous context")
			ctx.Println("  clear               Clear the screen")
			ctx.Println("  disconnect          Disconnect server")
			ctx.Println("  exit                Exit the client")
			ctx.Println("  help                Display help")
			ctx.Println("  use                 Select a context to use [use emulator | ue | gnb]")

		case "context_set":
			ctx.Println("Available commands :")
			ctx.Println("  select              Select a node to interact with [select <node-name>]")
			ctx.Println("")
			ctx.Println("General commands:")
			ctx.Println("  back                Go back to previous context")
			ctx.Println("  clear               Clear the screen")
			ctx.Println("  disconnect          Disconnect server")
			ctx.Println("  exit                Exit the client")
			ctx.Println("  help                Display this help")

		case "node":
			ctx.Printf("Available commands for %s :\n", currentContext.Name)
			commands := c.requestCommands(currentContext.NodeType, currentContext.Name)
			if len(commands) > 0 {
				for _, cmd := range commands {
					ctx.Printf("  %-16s %s\n", cmd.Name, cmd.Usage)
				}
			} else {
				// Fallback if no command info available
				for _, cmd := range currentContext.Commands {
					if cmd != "help" && cmd != "clear" && cmd != "exit" && cmd != "back" && cmd != "disconnect" {
						ctx.Printf("  %-16s\n", cmd)
					}
				}
			}

			ctx.Println("")
			ctx.Println("General commands:")
			ctx.Println("  back                Go back to previous context")
			ctx.Println("  clear               Clear the screen")
			ctx.Println("  disconnect          Disconnect server")
			ctx.Println("  exit                Exit the client")
			ctx.Println("  help                Display this help")
		}
	}
}

// navigateContext handles navigation between contexts (for non-AMF mode)
func (c *Client) navigateContext(command string, args []string) {
	currentContext := c.getCurrentContext()

	req := models.NavigationRequest{
		CurrentContext: currentContext.Name,
		Command:        command,
		Args:           args,
		ServerURL:      c.serverURL,
		NodeType:       currentContext.NodeType,
	}

	if command == "connect" && (c.serverURL == "" || len(c.contextStack) <= 1) {
		if len(args) < 1 {
			c.shell.Println("URL is required for connect command")
			return
		}

		c.serverURL = args[0]
		if !strings.HasPrefix(c.serverURL, "http://") && !strings.HasPrefix(c.serverURL, "https://") {
			c.serverURL = "http://" + c.serverURL
		}
	}

	endpoint := c.serverURL + "/api/context/navigate"

	// send request
	jsonData, err := json.Marshal(req)
	if err != nil {
		c.shell.Printf("Error preparing request: %v\n", err)
		return
	}

	resp, err := http.Post(endpoint, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		c.shell.Printf("Error communicating with server: %v\n", err)
		return
	}
	defer resp.Body.Close()

	// Process response
	var response models.NavigationResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		c.shell.Printf("Error parsing response: %v\n", err)
		return
	}

	if response.Error != "" {
		c.shell.Printf("Error: %s\n", response.Error)
		return
	}

	// Update context stack
	if command == "back" || command == "disconnect" {
		if len(c.contextStack) > 1 {
			c.contextStack = c.contextStack[:len(c.contextStack)-1]

			// Special handling for disconnect
			if command == "disconnect" {
				// Reset to root context
				c.contextStack = c.contextStack[:1]
				c.serverURL = ""

				c.setupCommands("root")
				c.shell.SetPrompt(">>> ")
				c.shell.Println("Disconnected from server")
				return
			}

			currentContext = c.getCurrentContext()
			c.setupCommands(currentContext.Type)
		} else {
			c.shell.Println("Already at root context")
			return
		}
	} else {
		response.Context.ServerURL = c.serverURL

		if command == "use" && len(args) > 0 {
			response.Context.NodeType = args[0]
		}

		// Add new context to stack
		c.contextStack = append(c.contextStack, response.Context)

		c.setupCommands(response.Context.Type)
	}

	// Update prompt
	if response.Prompt != "" {
		c.shell.SetPrompt(response.Prompt)
	} else {
		currentContext = c.getCurrentContext()
		if currentContext.Type == "root" || currentContext.Type == "server" {
			c.shell.SetPrompt(">>> ")
		} else {
			c.shell.SetPrompt(currentContext.Name + " >>> ")
		}
	}

	if response.Message != "" {
		c.shell.Println(response.Message)
	}

	// Setup node commands if applicable
	if command == "select" || (command == "use" && args[0] == "emulator") {
		c.setupNodeCommands(response.Context, response.Commands)
	}
}

// setupNodeCommands sets up commands for a specific node
func (c *Client) setupNodeCommands(context models.ClientContext, commands []models.CommandInfo) {
	if len(commands) == 0 {
		commands = c.requestCommands(context.NodeType, context.Name)
	}

	for _, cmdInfo := range commands {
		info := cmdInfo

		c.shell.AddCmd(&ishell.Cmd{
			Name:     info.Name,
			Help:     info.Usage,
			LongHelp: c.generateLongHelp(info),
			Func: func(ctx *ishell.Context) {
				result, err := c.execCmd(context.NodeType, context.Name, info.Name, ctx.Args)
				if err != nil {
					ctx.Printf("Error: %v\n", err)
					return
				}
				ctx.Println(result)
			},
		})
	}
}

// requestCommands fetches command definitions from the server
func (c *Client) requestCommands(nodeType, nodeName string) []models.CommandInfo {
	if c.serverURL == "" {
		return nil
	}

	url := fmt.Sprintf("%s/api/context/node/%s/%s/commands", c.serverURL, nodeType, nodeName)

	resp, err := http.Get(url)
	if err != nil || resp.StatusCode != http.StatusOK {
		return nil
	}
	defer resp.Body.Close()

	var commands []models.CommandInfo
	if err := json.NewDecoder(resp.Body).Decode(&commands); err != nil {
		return nil
	}

	return commands
}

// execCmd executes a command on the server
func (c *Client) execCmd(nodeType, nodeName, cmdName string, args []string) (string, error) {
	// Separate args and flags
	cmdArgs := []string{}
	cmdFlags := []string{}

	for _, arg := range args {
		if strings.HasPrefix(arg, "--") || strings.HasPrefix(arg, "-") {
			cmdFlags = append(cmdFlags, arg)
		} else {
			cmdArgs = append(cmdArgs, arg)
		}
	}

	// Check for help flag
	for _, flag := range cmdFlags {
		if flag == "--help" || flag == "-h" {
			cmdReq := models.CommandRequest{
				NodeType:    nodeType,
				NodeName:    nodeName,
				CommandPath: cmdName,
				Args:        []string{"--help"},
			}

			return c.sendCmd(cmdReq)
		}
	}

	// Execute normal command with all args and flags
	allArgs := append(cmdArgs, cmdFlags...)
	cmdReq := models.CommandRequest{
		NodeType:    nodeType,
		NodeName:    nodeName,
		CommandPath: cmdName,
		Args:        allArgs,
	}

	return c.sendCmd(cmdReq)
}

// sendCmd sends a command request to the server
func (c *Client) sendCmd(cmdReq models.CommandRequest) (string, error) {
	if c.serverURL == "" {
		return "", fmt.Errorf("not connected to a server")
	}

	jsonData, err := json.Marshal(cmdReq)
	if err != nil {
		return "", fmt.Errorf("failed to marshal command request: %v", err)
	}

	resp, err := http.Post(c.serverURL+"/api/exec", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to send command: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %v", err)
	}

	var response models.CommandResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return "", fmt.Errorf("failed to parse response: %v\nresponse body: %s", err, string(body))
	}

	if response.Error != "" {
		return "", fmt.Errorf("server error: %s", response.Error)
	}

	return response.Response, nil
}

// generateLongHelp creates detailed help for a command
func (c *Client) generateLongHelp(cmd models.CommandInfo) string {
	var sb strings.Builder
	sb.WriteString("\n")
	sb.WriteString(cmd.Name)

	if len(cmd.ArgsUsage) > 0 {
		sb.WriteString(" ")
		sb.WriteString(cmd.ArgsUsage)
	} else {
		sb.WriteString(" [command [command options]]")
	}
	sb.WriteString("\n")

	if len(cmd.Flags) > 0 {
		for _, flag := range cmd.Flags {
			sb.WriteString("   --")
			sb.WriteString(flag.Name)
			sb.WriteString(":  ")
			sb.WriteString(flag.Usage)
			if flag.DefaultText != "" {
				sb.WriteString(" (default: ")
				sb.WriteString(flag.DefaultText)
				sb.WriteString(")")
			}
			sb.WriteString("\n")
		}
	}

	return sb.String()
}

// getCurrentContext returns the current context
func (c *Client) getCurrentContext() models.ClientContext {
	if len(c.contextStack) > 0 {
		return c.contextStack[len(c.contextStack)-1]
	}
	return models.ClientContext{Type: "root", Name: "root"}
}

// Helper function to get command names from CommandInfo slice
func getCommandNames(cmds []models.CommandInfo) []string {
	names := make([]string, len(cmds))
	for i, cmd := range cmds {
		names[i] = cmd.Name
	}
	return names
}
