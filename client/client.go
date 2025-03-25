package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"test_monitor/models"
	"time"

	"github.com/abiosoft/ishell"
)

type NodeType int

const (
	UE  NodeType = 0
	Gnb NodeType = 1
)

func (n NodeType) String() string {
	return [...]string{"ue", "gnb"}[n]
}

type Shell struct {
	Nodes []Node
	Ip    string
	Port  int
	Shell *ishell.Shell
}

type Node struct {
	Type        NodeType
	ActiveNodes []string
	Shell       *ishell.Shell
}

type NodesListResponse struct {
	Nodes []string `json:"nodes"`
	Error string   `json:"error"`
}

type ConnectResponse struct {
	Status  string            `json:"status"`
	Objects map[string]string `json:"objects"`
	Error   string            `json:"error"`
}

type CommandResponse struct {
	Response string `json:"response"`
	Error    string `json:"error"`
}

type Command struct {
	Name  string `json:"name"`
	Help  string `json:"help"`
	Usage string `json:"defaultUsage"`
}

type CommandsListResponse struct {
	Commands []Command `json:"commands"`
	Error    string    `json:"error"`
}

func (s *Shell) newShell(id NodeType) {
	s.Nodes = []Node{
		{
			Type:  id,
			Shell: ishell.New(),
		},
	}

	// Setup basic commands for this shell
	s.setupMainShell()
}

func (s *Shell) setupMainShell() (objTypes []string) {
	shell := s.Nodes[0].Shell
	objTypes = []string{} // Initialize the return value

	// Basic commands available in all shells
	shell.AddCmd(&ishell.Cmd{
		Name: "clear",
		Help: "clear the screen",
		Func: func(c *ishell.Context) {
			c.Println("Clearing screen...")
			c.Println()
			c.ClearScreen()
		},
	})

	shell.AddCmd(&ishell.Cmd{
		Name: "exit",
		Help: "Exit the client",
		Func: func(c *ishell.Context) {
			c.Println("Exiting...")
			c.Stop()
		},
	})

	shell.AddCmd(&ishell.Cmd{
		Name: "help",
		Help: "display help",
		Func: func(c *ishell.Context) {
			c.Println("Commands:")
			c.Println("  clear      clear the screen")
			c.Println("  dump       List UEs or gNBs (usage: dump <ue|gnb>)")
			c.Println("  exit       Exit the client")
			c.Println("  help       display help")
			c.Println("  select     Select a node to interact with (usage: select <node-name>)")
		},
	})

	// Add dump command
	shell.AddCmd(&ishell.Cmd{
		Name: "dump",
		Help: "List UEs or gNBs (usage: dump <ue|gnb|amf>)",
		Func: func(c *ishell.Context) {
			if len(c.Args) == 0 {
				c.Println("Usage: dump <ue|gnb|amf>")
				return
			}

			nodeType := strings.ToLower(c.Args[0])
			nodes, err := s.getNodesOfType(nodeType)
			if err != nil {
				c.Println("Error:", err)
				return
			}

			for _, node := range nodes {
				c.Println(node)
			}
		},
	})

	// Add select command
	shell.AddCmd(&ishell.Cmd{
		Name: "select",
		Help: "Select a node to interact with (usage: select <node-name>)",
		Func: func(c *ishell.Context) {
			if len(c.Args) != 1 {
				c.Println("Usage: select <node-name>")
				return
			}

			nodeName := c.Args[0]
			c.Println("Selected node:", nodeName)

			// Try to determine the node type
			var nodeType string

			switch {
			case strings.HasPrefix(nodeName, "UE-"), strings.HasPrefix(nodeName, "imsi-"):
				nodeType = "ue"
			case strings.HasPrefix(nodeName, "GNB-"), strings.HasPrefix(nodeName, "gnb-"):
				nodeType = "gnb"
			case strings.HasPrefix(nodeName, "AMF-"), strings.HasPrefix(nodeName, "amf-"):
				nodeType = "amf"
			default:
				// Try each type
				for _, t := range objTypes {
					exists, err := s.checkNodeExists(t, nodeName)
					if err == nil && exists {
						nodeType = t
						break
					}
				}
			}

			if nodeType == "" {
				c.Printf("Node %s does not exist or its type cannot be determined\n", nodeName)
				return
			}

			// Create and run a shell for this node
			s.handleSelectCommand(shell, nodeType, nodeName)
		},
	})
	return nil
}

func (s *Shell) handleSelectCommand(shell *ishell.Shell, nodeType string, nodeName string) {
	// Check if node exists first
	exists, err := s.checkNodeExists(nodeType, nodeName)
	if err != nil {
		shell.Println("Error:", err)
		return
	}

	if !exists {
		shell.Printf("Node %s of type %s does not exist\n", nodeName, nodeType)
		return
	}

	// Create a sub-shell for this node
	nodeShell := ishell.New()
	nodeShell.SetPrompt(fmt.Sprintf("%s >>> ", nodeName))
	nodeShell.ShowPrompt(true)

	// Setup node commands
	if err := s.setupNodeCommands(nodeShell, nodeType, nodeName); err != nil {
		shell.Println("Error setting up commands:", err)
		return
	}
	// Run the node shell (this will block until exit)
	nodeShell.Run()
}

func (s *Shell) parseCommand(_ string, args []string) map[string]string {
	parsedArgs := make(map[string]string)

	// Process remaining arguments into key-value pairs
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if strings.HasPrefix(arg, "--") {
			// This is a flag
			flag := strings.TrimPrefix(arg, "--")

			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "--") {
				// Flag without value
				parsedArgs[flag] = "true"
			} else {
				// Flag with value
				value := args[i+1]

				// If already have this flag, append the new value
				if existingValue, ok := parsedArgs[flag]; ok {
					parsedArgs[flag] = existingValue + "," + value
				} else {
					parsedArgs[flag] = value
				}
				i++
			}
		}
	}

	return parsedArgs
}

func (s *Shell) setupNodeCommands(nodeShell *ishell.Shell, nodeType string, nodeName string) error {
	commands, err := s.getNodeCommands(nodeType)
	if err != nil {
		return fmt.Errorf("failed to get commands: %v", err)
	}

	shell := nodeShell

	// Add help command specific to this node type
	shell.AddCmd(&ishell.Cmd{
		Name: "help",
		Help: "display help for this node",
		Func: func(c *ishell.Context) {
			c.Println("Available commands for", nodeName, ":")
			for cmdName, cmdHelp := range commands {
				c.Printf("  %-12s %s\n", cmdName, cmdHelp)
			}
			c.Println("\nGeneral commands:")
			c.Println("  clear        clear the screen")
			c.Println("  exit         Exit the client")
			c.Println("  help         display this help")
		},
	})

	// Add node-specific commands
	for cmdName, cmdHelp := range commands {
		name := cmdName // Create a local copy for the closure

		shell.AddCmd(&ishell.Cmd{
			Name: name,
			Help: cmdHelp,
			Func: func(c *ishell.Context) {
				// Build the complete command to send to the server
				command := name
				if len(c.Args) > 0 {
					command += " " + strings.Join(c.Args, " ")
				}

				args := s.parseCommand(command, c.Args)

				// Send response to server
				response, err := s.sendCommand(nodeType, nodeName, command, args)
				if err != nil {
					c.Println("Error:", err)
					return
				}

				c.Println(response)
			},
		})
	}

	// Add clear command
	shell.AddCmd(&ishell.Cmd{
		Name: "clear",
		Help: "Clear the screen",
		Func: func(c *ishell.Context) {
			c.Println("Clearing screen...")
			c.Println()
			c.ClearScreen()
		},
	})

	shell.AddCmd(&ishell.Cmd{
		Name: "exit",
		Help: "Return to main shell",
		Func: func(c *ishell.Context) {
			c.Println("Returning to main shell...")
			c.Stop()
		},
	})
	return nil
}

func Connect(s *Shell, nodeType NodeType, serverIP string, serverPort int) (*ConnectResponse, error) {
	// Initialize the shell
	s.Ip = serverIP
	s.Port = serverPort
	s.newShell(nodeType)

	url := fmt.Sprintf("http://%s:%d/nodes/types", serverIP, serverPort)
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to server at %s:%d: %v", serverIP, serverPort, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned unexpected status code: %d", resp.StatusCode)
	}

	// Read the response to get available types
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	var result struct {
		Types []string `json:"types"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}

	// Create a ConnectResponse with the obtained types
	connectResp := ConnectResponse{
		Status:  "connected",
		Objects: make(map[string]string),
		Error:   "",
	}

	for _, nodeType := range result.Types {
		connectResp.Objects[nodeType] = nodeType
	}

	return &connectResp, nil
}

func (s *Shell) getNodesOfType(nodeType string) ([]string, error) {
	url := fmt.Sprintf("http://%s:%d/nodes/%s", s.Ip, s.Port, nodeType)
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to get nodes: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	var result struct {
		Nodes []string `json:"nodes"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}

	return result.Nodes, nil
}

func (s *Shell) getNodeCommands(nodeType string) (map[string]string, error) {
	url := fmt.Sprintf("http://%s:%d/commands/%s", s.Ip, s.Port, nodeType)
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to get commands: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	var result struct {
		Commands []struct {
			Name         string `json:"name"`
			Help         string `json:"help"`
			DefaultUsage string `json:"defaultUsage"`
		} `json:"commands"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}

	// Convert the commands to a map for easier use
	commandsMap := make(map[string]string)
	for _, cmd := range result.Commands {
		commandsMap[cmd.Name] = cmd.Help
	}

	return commandsMap, nil
}

func (s *Shell) checkNodeExists(nodeType string, nodeName string) (bool, error) {
	url := fmt.Sprintf("http://%s:%d/check/%s/%s", s.Ip, s.Port, nodeType, nodeName)
	resp, err := http.Get(url)
	if err != nil {
		return false, fmt.Errorf("failed to check node: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, fmt.Errorf("failed to read response: %v", err)
	}

	var result struct {
		Exists bool   `json:"exists"`
		Error  string `json:"error"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return false, fmt.Errorf("failed to parse response: %v", err)
	}

	if result.Error != "" {
		return false, fmt.Errorf("server error: %s", result.Error)
	}

	return result.Exists, nil
}

func (s *Shell) sendCommand(nodeType, nodeName, fullCommand string, args map[string]string) (string, error) {
	cmdParts := strings.Fields(fullCommand)
	baseCommand := cmdParts[0]
	// Create the form args
	formArgs := models.FormArgs{
		NodeType:   nodeType,
		NodeName:   nodeName,
		Command:    baseCommand,
		Arguments:  make(map[string]string),
		RawCommand: fullCommand,
	}

	for key, value := range args {
		// Special case handling for known arguments
		if key == "amf" {
			formArgs.Arguments["amf-name"] = value
		} else if key == "smf" {
			formArgs.Arguments["smf-name"] = value
		} else if key == "imsi" {
			formArgs.Arguments["imsi"] = value
		} else {
			formArgs.Arguments[key] = value
		}
	}

	// Convert to JSON
	jsonData, err := json.Marshal(formArgs)
	if err != nil {
		return "", fmt.Errorf("failed to marshal command: %v", err)
	}

	// Send to server
	url := fmt.Sprintf("http://%s:%d/execute", s.Ip, s.Port)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to send command: %v", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %v", err)
	}

	var result struct {
		Result string `json:"result"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("failed to parse response: %v", err)
	}

	return result.Result, nil
}

func main() {
	args := os.Args

	// Debug output
	fmt.Println("Arguments:", args)

	if len(args) < 3 {
		fmt.Println("Usage: go run client/client.go -p <server-ip>:<server-port>")
		os.Exit(1)
	}

	// Tìm vị trí của cờ -p
	var serverAddr string
	for i := 1; i < len(args)-1; i++ {
		if args[i] == "-p" {
			serverAddr = args[i+1]
			break
		}
	}

	if serverAddr == "" {
		fmt.Println("Usage: go run client/client.go -p <server-ip>:<server-port>")
		fmt.Println("Could not find -p flag with server address")
		os.Exit(1)
	}

	// Parse server address
	parts := strings.Split(serverAddr, ":")
	if len(parts) != 2 {
		fmt.Println("Invalid server address format. Expected format: <ip>:<port>")
		os.Exit(1)
	}

	serverIP := parts[0]
	var serverPort int
	_, err := fmt.Sscanf(parts[1], "%d", &serverPort)
	if err != nil {
		fmt.Printf("Invalid port number: %s\n", parts[1])
		os.Exit(1)
	}

	// Create shell instance
	shell := &Shell{
		Ip:   serverIP,
		Port: serverPort,
	}

	// Connect to server
	fmt.Printf("Attempting to connect to server at %s:%d\n", serverIP, serverPort)
	connectResp, err := Connect(shell, UE, serverIP, serverPort)
	if err != nil {
		fmt.Printf("Failed to connect: %v\n", err)
		return
	}

	// Extract available object types
	var objTypes []string
	for k := range connectResp.Objects {
		objTypes = append(objTypes, k)
	}

	fmt.Printf("Connected to server at %s:%d\n", serverIP, serverPort)
	fmt.Printf("Available object types: %s\n", strings.Join(objTypes, ", "))

	// Configure shell display
	shell.Nodes[0].Shell.SetPrompt(">>> ")
	shell.Nodes[0].Shell.ShowPrompt(true)

	// Run the shell
	shell.Nodes[0].Shell.Run()
}
