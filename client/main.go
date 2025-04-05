package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"test_monitor/models"

	"github.com/abiosoft/ishell"
)

var (
	serverURL    string
	contextStack []models.ClientContext
)

func main() {
	flag.Parse()

	shell := ishell.New()

	contextStack = []models.ClientContext{
		{
			Type:     "root",
			Name:     "root",
			Commands: []string{"help", "clear", "exit", "connect"},
		},
	}

	setupCommands(shell, "root")

	shell.Println("Interactive CLI Client")
	shell.SetPrompt(">>> ")

	shell.Run()
}

// setupCommands sets up the commands for the shell based on the context
// để tránh đệ quy
func setupCommands(shell *ishell.Shell, contextType string) {
	for _, cmd := range []string{"help", "clear", "exit", "back", "disconnect", "use", "select", "connect"} {
		shell.DeleteCmd(cmd)
	}

	// Add some basic commands
	shell.AddCmd(&ishell.Cmd{
		Name: "help",
		Help: "display help",
		Func: displayHelp(shell),
	})

	shell.AddCmd(&ishell.Cmd{
		Name: "clear",
		Help: "clear the screen",
		Func: func(c *ishell.Context) {
			c.ClearScreen()
		},
	})

	shell.AddCmd(&ishell.Cmd{
		Name: "exit",
		Help: "exit the program",
		Func: func(c *ishell.Context) {
			c.Println("Goodbye!")
			os.Exit(0)
		},
	})

	// Add commands based on context type
	switch contextType {
	case "root":
		// Add connect command for root context
		shell.AddCmd(&ishell.Cmd{
			Name:     "connect",
			Help:     "Connect to a server [connect http://localhost:4000]",
			LongHelp: "Connect to a server using URL. Example: connect http://localhost:4000",
			Func: func(c *ishell.Context) {
				if len(c.Args) < 1 {
					c.Println("Usage: connect <server-url>")
					return
				}
				url := c.Args[0]
				connectToServer(shell, url)
			},
		})

	case "server":
		// Add command disconnect and back for server context
		shell.AddCmd(&ishell.Cmd{
			Name: "back",
			Help: "Go back to previous context",
			Func: func(c *ishell.Context) {
				navigateContext(shell, "back", nil)
			},
		})

		shell.AddCmd(&ishell.Cmd{
			Name: "disconnect",
			Help: "Disconnect from server",
			Func: func(c *ishell.Context) {
				navigateContext(shell, "disconnect", nil)
			},
		})

		// Add command use for server context
		shell.AddCmd(&ishell.Cmd{
			Name: "use",
			Help: "Select a context to use [use emulator | ue | gnb]",
			Func: func(c *ishell.Context) {
				if len(c.Args) < 1 {
					c.Println("Usage: use <context-type>")
					c.Println("Context types: emulator, ue, gnb")
					return
				}
				navigateContext(shell, "use", c.Args)
			},
		})

	case "context_set":
		// Add command back and disconnect for context_set
		shell.AddCmd(&ishell.Cmd{
			Name: "back",
			Help: "Go back to previous context",
			Func: func(c *ishell.Context) {
				navigateContext(shell, "back", nil)
			},
		})

		shell.AddCmd(&ishell.Cmd{
			Name: "disconnect",
			Help: "Disconnect from server",
			Func: func(c *ishell.Context) {
				navigateContext(shell, "disconnect", nil)
			},
		})

		// Add command select for context_set
		shell.AddCmd(&ishell.Cmd{
			Name: "select",
			Help: "Select a node to interact with [select <node-name>]",
			Func: func(c *ishell.Context) {
				if len(c.Args) < 1 {
					c.Println("Usage: select <node-name>")
					return
				}
				navigateContext(shell, "select", c.Args)
			},
		})

	case "node":
		// Add command back and disconnect for node context
		shell.AddCmd(&ishell.Cmd{
			Name: "back",
			Help: "Go back to previous context",
			Func: func(c *ishell.Context) {
				navigateContext(shell, "back", nil)
			},
		})

		shell.AddCmd(&ishell.Cmd{
			Name: "disconnect",
			Help: "Disconnect from server",
			Func: func(c *ishell.Context) {
				navigateContext(shell, "disconnect", nil)
			},
		})
	}
}

// connectToServer handles server connection
func connectToServer(shell *ishell.Shell, url string) {
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		url = "http://" + url
	}

	// Validate server URL
	serverURL = url

	// Check server connection
	resp, err := http.Get(url + "/api/context")
	if err != nil {
		shell.Printf("Failed to connect to server: %v\n", err)
		serverURL = "" // Reset if failing
		return
	}
	defer resp.Body.Close()

	// Navigate to new context
	navigateContext(shell, "connect", []string{url})
}

// displayHelp generates help text for the current context
func displayHelp(_ *ishell.Shell) func(*ishell.Context) {
	return func(c *ishell.Context) {
		currentContext := getCurrentContext()

		switch currentContext.Type {
		case "root":
			c.Println("Commands:")
			c.Println("  clear        clear the screen")
			c.Println("  connect      Connect to a server [connect http://localhost:4000]")
			c.Println("  exit         exit the program")
			c.Println("  help         display help")

		case "server":
			c.Println("Commands:")
			c.Println("  back                Go back to previous context")
			c.Println("  clear               Clear the screen")
			c.Println("  disconnect          Disconnect server")
			c.Println("  exit                Exit the client")
			c.Println("  help                Display help")
			c.Println("  use                 Select a context to use [use emulator | ue | gnb]")

		case "context_set":
			c.Println("Available commands :")
			c.Println("  select              Select a node to interact with [select <node-name>]")
			c.Println("")
			c.Println("General commands:")
			c.Println("  back                Go back to previous context")
			c.Println("  clear               Clear the screen")
			c.Println("  disconnect          Disconnect server")
			c.Println("  exit                Exit the client")
			c.Println("  help                Display this help")

		case "node":
			c.Printf("Available commands for %s :\n", currentContext.Name)

			// Get command info from server
			commands := requestCommands(currentContext.NodeType, currentContext.Name)
			if len(commands) > 0 {
				for _, cmd := range commands {
					c.Printf("  %-16s %s\n", cmd.Name, cmd.Usage)
				}
			} else {
				// Fallback if no command info available
				for _, cmd := range currentContext.Commands {
					if cmd != "help" && cmd != "clear" && cmd != "exit" && cmd != "back" && cmd != "disconnect" {
						c.Printf("  %-16s\n", cmd)
					}
				}
			}

			c.Println("")
			c.Println("General commands:")
			c.Println("  back                Go back to previous context")
			c.Println("  clear               Clear the screen")
			c.Println("  disconnect          Disconnect server")
			c.Println("  exit                Exit the client")
			c.Println("  help                Display this help")
		}
	}
}

// navigateContext handles navigation between contexts
func navigateContext(shell *ishell.Shell, command string, args []string) {
	currentContext := getCurrentContext()

	// Prepare request with detailed context info
	req := models.NavigationRequest{
		CurrentContext: currentContext.Name,
		Command:        command,
		Args:           args,
		ServerURL:      serverURL,
		NodeType:       currentContext.NodeType,
	}

	if command == "connect" && (serverURL == "" || len(contextStack) <= 1) {
		if len(args) < 1 {
			shell.Println("URL is required for connect command")
			return
		}

		serverURL = args[0]
		if !strings.HasPrefix(serverURL, "http://") && !strings.HasPrefix(serverURL, "https://") {
			serverURL = "http://" + serverURL
		}
	}

	// Endpoint cho request
	endpoint := serverURL + "/api/context/navigate"

	// send request
	jsonData, err := json.Marshal(req)
	if err != nil {
		shell.Printf("Error preparing request: %v\n", err)
		return
	}

	resp, err := http.Post(endpoint, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		shell.Printf("Error communicating with server: %v\n", err)
		return
	}
	defer resp.Body.Close()

	// Process response
	var response models.NavigationResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		shell.Printf("Error parsing response: %v\n", err)
		return
	}

	if response.Error != "" {
		shell.Printf("Error: %s\n", response.Error)
		return
	}

	// Update context stack
	if command == "back" || command == "disconnect" {
		if len(contextStack) > 1 {
			// Remove current context from stack
			contextStack = contextStack[:len(contextStack)-1]

			// Xử lý riêng cho disconnect
			if command == "disconnect" {
				// Reset root context
				contextStack = contextStack[:1]
				serverURL = ""

				setupCommands(shell, "root")
				shell.SetPrompt(">>> ")
				shell.Println("Disconnected from server")
				return
			}

			currentContext = getCurrentContext()
			setupCommands(shell, currentContext.Type)
		} else {
			shell.Println("Already at root context")
			return
		}
	} else {
		response.Context.ServerURL = serverURL

		if command == "use" && len(args) > 0 {
			response.Context.NodeType = args[0]
		}

		// Add new context to stack
		contextStack = append(contextStack, response.Context)

		setupCommands(shell, response.Context.Type)
	}

	// Update prompt
	if response.Prompt != "" {
		shell.SetPrompt(response.Prompt)
	} else {
		currentContext = getCurrentContext()
		if currentContext.Type == "root" || currentContext.Type == "server" {
			shell.SetPrompt(">>> ")
		} else {
			shell.SetPrompt(currentContext.Name + " >>> ")
		}
	}

	if response.Message != "" {
		shell.Println(response.Message)
	}

	// Setup node commands if applicable
	if command == "select" || (command == "use" && args[0] == "emulator") {
		setupNodeCommands(shell, response.Context, response.Commands)
	}
}

// setupNodeCommands sets up commands for a specific node
func setupNodeCommands(shell *ishell.Shell, context models.ClientContext, commands []models.CommandInfo) {
	if len(commands) == 0 {
		commands = requestCommands(context.NodeType, context.Name)
	}

	for _, cmdInfo := range commands {
		info := cmdInfo

		shell.AddCmd(&ishell.Cmd{
			Name:     info.Name,
			Help:     info.Usage,
			LongHelp: generateLongHelp(info),
			Func: func(c *ishell.Context) {
				result, err := execCmd(context.NodeType, context.Name, info.Name, c.Args)
				if err != nil {
					c.Printf("Error: %v\n", err)
					return
				}
				c.Println(result)
			},
		})
	}
}

// requestCommands fetches command definitions from the server
func requestCommands(nodeType, nodeName string) []models.CommandInfo {
	if serverURL == "" {
		return nil
	}

	url := fmt.Sprintf("%s/api/context/node/%s/%s/commands", serverURL, nodeType, nodeName)

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
func execCmd(nodeType, nodeName, cmdName string, args []string) (string, error) {
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

			return sendCmd(cmdReq)
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

	return sendCmd(cmdReq)
}

// sendCmd sends a command request to the server
func sendCmd(cmdReq models.CommandRequest) (string, error) {
	if serverURL == "" {
		return "", fmt.Errorf("not connected to a server")
	}

	jsonData, err := json.Marshal(cmdReq)
	if err != nil {
		return "", fmt.Errorf("failed to marshal command request: %v", err)
	}

	resp, err := http.Post(serverURL+"/api/exec", "application/json", bytes.NewBuffer(jsonData))
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
func generateLongHelp(cmd models.CommandInfo) string {
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
func getCurrentContext() models.ClientContext {
	if len(contextStack) > 0 {
		return contextStack[len(contextStack)-1]
	}
	return models.ClientContext{Type: "root", Name: "root"}
}
