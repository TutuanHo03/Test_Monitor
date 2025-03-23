package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"test_monitor/models"

	"github.com/abiosoft/ishell"
	"github.com/gin-gonic/gin"
)

type NodeType int

const (
	UE NodeType = iota
	Gnb
	Amf
)

func (n NodeType) String() string {
	return [...]string{"ue", "gnb", "amf"}[n]
}

type Shell struct {
	Nodes []Node
	Ip    string
	Port  int
	Shell *ishell.Shell
}

type Node struct {
	Type        NodeType
	Name        string
	ActiveNodes []string
	Command     []models.Command
	Shell       *ishell.Shell
}

// Config represents the structure of the command.json file
type Config struct {
	UE  NodeConfig `json:"ue"`
	Gnb NodeConfig `json:"gnb"`
	Amf NodeConfig `json:"amf"`
}

type NodeConfig struct {
	Commands []models.Command `json:"commands"`
}

// CommandWithFn extends the Command with a function field
type CommandWithFn struct {
	models.Command
	Fn func([]models.Subcommand)
}

func NewServer(ip string, port int) *Shell {
	return &Shell{
		Ip:    ip,
		Port:  port,
		Shell: ishell.New(),
	}
}

func (s *Shell) LoadConfig(configPath string) error {
	configFile, err := os.Open(configPath)
	if err != nil {
		return fmt.Errorf("error opening config file: %v", err)
	}
	defer configFile.Close()

	configData, err := io.ReadAll(configFile)
	if err != nil {
		return fmt.Errorf("error reading config file: %v", err)
	}

	var config Config
	if err := json.Unmarshal(configData, &config); err != nil {
		return fmt.Errorf("error parsing config file: %v", err)
	}

	// Create nodes from config
	s.Nodes = append(s.Nodes, Node{
		Type:    UE,
		Command: config.UE.Commands,
		Shell:   ishell.New(),
	})

	s.Nodes = append(s.Nodes, Node{
		Type:    Gnb,
		Command: config.Gnb.Commands,
		Shell:   ishell.New(),
	})

	s.Nodes = append(s.Nodes, Node{
		Type:    Amf,
		Command: config.Amf.Commands,
		Shell:   ishell.New(),
	})

	return nil
}

func (s *Shell) SetupShellUE(fns []func(any)) {
	for _, ue := range s.Nodes {
		if ue.Type == UE {
			for i := range ue.Command {
				//Setup Command vs args
				form := models.FormArgs{}
				fns[i](form)
			}
		}
	}
}

func (s *Shell) SetupShellGnb(fns []func(any)) {
	for _, node := range s.Nodes {
		if node.Type == Gnb {
			for i := range node.Command {
				// Setup Command vs args
				form := models.FormArgs{}
				fns[i](form)
			}
		}
	}
}

func (s *Shell) SetupShellAmf(fns []func(any)) {
	for _, node := range s.Nodes {
		if node.Type == Amf {
			for i := range node.Command {
				// Setup Command vs args
				form := models.FormArgs{}
				fns[i](form)
			}
		}
	}
}

func (s *Shell) SetupServer() {
	r := gin.Default()

	// API to get available node types
	r.GET("/nodes/types", func(c *gin.Context) {
		types := []string{"ue", "gnb", "amf"}
		c.JSON(http.StatusOK, gin.H{
			"types": types,
		})
	})

	// API to get available nodes of a specific type
	r.GET("/nodes/:type", func(c *gin.Context) {
		nodeType := c.Param("type")
		var nodes []string

		for _, node := range s.Nodes {
			if strings.EqualFold(node.Type.String(), nodeType) {
				nodes = append(nodes, node.Name)
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"nodes": nodes,
		})
	})

	r.GET("/commands/:nodeType", func(c *gin.Context) {
		nodeType := c.Param("nodeType")
		var commands []models.Command

		for _, node := range s.Nodes {
			if strings.EqualFold(node.Type.String(), nodeType) {
				// Convert Command to the format expected by the client
				for _, cmd := range node.Command {
					commands = append(commands, models.Command{
						Name:  cmd.Name,
						Help:  cmd.Help,
						Usage: cmd.Usage,
					})
				}
				break
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"commands": commands,
		})
	})

	// NEW: API to check if a node exists
	r.GET("/check/:nodeType/:nodeName", func(c *gin.Context) {
		nodeType := c.Param("nodeType")
		nodeName := c.Param("nodeName")

		exists := false
		for _, node := range s.Nodes {
			if strings.EqualFold(node.Type.String(), nodeType) && node.Name == nodeName {
				exists = true
				break
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"exists": exists,
		})
	})

	// API to execute commands
	r.POST("/execute", func(c *gin.Context) {
		var formArgs models.FormArgs
		if err := c.ShouldBindJSON(&formArgs); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Process command
		result := s.processCommand(formArgs)
		c.JSON(http.StatusOK, gin.H{
			"result": result,
		})
	})

	// Start the server
	serverAddr := fmt.Sprintf("%s:%d", s.Ip, s.Port)
	go func() {
		if err := r.Run(serverAddr); err != nil {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()
}

func (s *Shell) processCommand(args models.FormArgs) string {
	// Find the appropriate node
	var targetNode *Node
	for i, node := range s.Nodes {
		if strings.EqualFold(node.Type.String(), args.NodeType) &&
			(args.NodeName == "" || node.Name == args.NodeName) {
			targetNode = &s.Nodes[i]
			break
		}
	}

	if targetNode == nil {
		return fmt.Sprintf("Error: Node of type %s not found", args.NodeType)
	}

	// Find the command
	var targetCmd *models.Command
	for i, cmd := range targetNode.Command {
		if cmd.Name == args.Command {
			targetCmd = &targetNode.Command[i]
			break
		}
	}

	if targetCmd == nil {
		return fmt.Sprintf("Error: Command %s not found for node type %s", args.Command, args.NodeType)
	}

	// Find the subcommand
	var targetSubcmd *models.Subcommand
	for i, subcmd := range targetCmd.Subcommands {
		if subcmd.Name == args.Subcommand {
			targetSubcmd = &targetCmd.Subcommands[i]
			break
		}
	}

	if targetSubcmd == nil {
		return fmt.Sprintf("Error: Subcommand %s not found for command %s", args.Subcommand, args.Command)
	}

	// Process response template
	response := targetSubcmd.Response
	response = strings.ReplaceAll(response, "${nodeName}", targetNode.Name)

	// Process and handle arguments
	for key, value := range args.Arguments {
		for i, arg := range targetSubcmd.Arguments {
			if arg.Name == key {
				// Save the value in the argument
				targetSubcmd.Arguments[i].Value = value

				// Replace placeholder in response
				if strings.Contains(value, ",") {
					// Format the comma-separated list nicely
					values := strings.Split(value, ",")
					formattedValue := strings.Join(values, ", ")
					response = strings.ReplaceAll(response, "${"+key+"}", formattedValue)
				} else {
					response = strings.ReplaceAll(response, "${"+key+"}", value)
				}
			}
		}
	}

	return response
}

func main() {
	server := NewServer("0.0.0.0", 4000)

	// Load commands from config
	err := server.LoadConfig("config/command.json")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Add some example nodes
	server.Nodes[0].Name = "imsi-001"
	server.Nodes[0].ActiveNodes = []string{"AMF-01", "AMF-02"}

	server.Nodes[1].Name = "GNB-001"
	server.Nodes[1].ActiveNodes = []string{"AMF-01"}

	server.Nodes[2].Name = "AMF-01"
	server.Nodes[2].ActiveNodes = []string{"UE-001", "GNB-001"}

	// Setup the server
	server.SetupServer()

	fmt.Println("Server started at 0.0.0.0:4000")
	fmt.Println("Available object types: ue, gnb, amf")

	// Keep the main goroutine alive
	select {}
}
