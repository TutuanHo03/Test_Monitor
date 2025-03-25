package main

import (
	"fmt"

	"log"
	"net/http"

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
	AllNodes    map[string][]string
	ActiveNodes map[string][]string // Format: {"gnb": ["gnb1", "gnb2"], "amf": ["amf1", "amf2"]}
	Commands    []models.Command
	Shell       *ishell.Shell
}

func NewServer(ip string, port int) *Shell {
	return &Shell{
		Ip:    ip,
		Port:  port,
		Shell: ishell.New(),
		Nodes: []Node{
			{
				Type:        UE,
				Name:        "UE-001",
				AllNodes:    make(map[string][]string),
				ActiveNodes: make(map[string][]string),
				Commands:    []models.Command{},
				Shell:       ishell.New(),
			},
			{
				Type:        Gnb,
				Name:        "GNB-001",
				AllNodes:    make(map[string][]string),
				ActiveNodes: make(map[string][]string),
				Commands:    []models.Command{},
				Shell:       ishell.New(),
			},
		},
	}
}

// CommandHandler defines the signature for command handler functions
type CommandHandler func(map[string]string) string

// SetupShellUE configures UE commands with their handlers
func (s *Shell) SetupShellUE(fn func(map[string]string), args map[string]string) {
	// Find UE node
	var ueNode *Node
	for i, node := range s.Nodes {
		if node.Type == UE {
			ueNode = &s.Nodes[i]
			break
		}
	}

	if ueNode == nil {
		// Create new UE node if not found
		ueNode = &Node{
			Type:        UE,
			Name:        "UE-Default",
			ActiveNodes: make(map[string][]string),
			Commands:    []models.Command{},
			Shell:       ishell.New(),
		}
		s.Nodes = append(s.Nodes, *ueNode)
	}

	// Define handlers
	registerHandler := func(arguments map[string]string) string {
		// Process the arguments
		nodeName := ueNode.Name
		response := ""

		// Handle --help flag
		if _, ok := arguments["help"]; ok {
			return "Usage: register [--amf] [--smf] [--help]\n" +
				"--amf      : Register UE to AMF\n" +
				"--smf      : Register UE to SMF\n" +
				"--help     : Show this help message"
		}

		// Process AMF arguments
		if amfValues, ok := arguments["amf-name"]; ok {
			amfs := strings.Split(amfValues, ",")
			formattedAmfs := strings.Join(amfs, ", ")
			if response != "" {
				response += ", "
			}
			response += fmt.Sprintf("AMF: %s", formattedAmfs)
		}

		// Process SMF arguments
		if smfValues, ok := arguments["smf-name"]; ok {
			smfs := strings.Split(smfValues, ",")
			formattedSmfs := strings.Join(smfs, ", ")
			if response != "" {
				response += ", "
			}
			response += fmt.Sprintf("SMF: %s", formattedSmfs)
		}

		if response == "" {
			return "Error: No valid arguments provided"
		}

		return fmt.Sprintf("Registering UE %s to %s", nodeName, response)
	}

	deregisterHandler := func(arguments map[string]string) string {
		// Process the arguments
		nodeName := ueNode.Name
		response := ""
		forceFlag := false

		// Handle --help flag
		if _, ok := arguments["help"]; ok {
			return "Usage: deregister [--amf] [--smf] [--help] [--force]\n" +
				"--amf      : Deregister UE from AMF\n" +
				"--smf      : Deregister UE from SMF\n" +
				"--force    : Force deregister\n" +
				"--help     : Show this help message"
		}

		// Check if force flag is present
		if _, ok := arguments["force"]; ok {
			forceFlag = true
		}

		// Process AMF arguments
		if amfValues, ok := arguments["amf-name"]; ok {
			amfs := strings.Split(amfValues, ",")
			formattedAmfs := strings.Join(amfs, ", ")
			if response != "" {
				response += ", "
			}
			response += fmt.Sprintf("AMF: %s", formattedAmfs)
		}

		// Process SMF arguments
		if smfValues, ok := arguments["smf-name"]; ok {
			smfs := strings.Split(smfValues, ",")
			formattedSmfs := strings.Join(smfs, ", ")
			if response != "" {
				response += ", "
			}
			response += fmt.Sprintf("SMF: %s", formattedSmfs)
		}

		if response == "" {
			return "Error: No valid arguments provided"
		}

		if forceFlag {
			return fmt.Sprintf("Forcing to deregister UE %s from %s", nodeName, response)
		}
		return fmt.Sprintf("Deregistering UE %s from %s", nodeName, response)
	}

	// Setup register command
	registerCommand := models.Command{
		Name:  "register",
		Help:  "Sign in the UE to Core",
		Usage: "Usage: register [--amf] [--smf] [--help]",
		Func:  registerHandler,
		Arguments: []models.Argument{
			{
				Tag:          "--amf",
				Description:  "AMF name to register with",
				Type:         "string",
				Required:     false,
				AllowMutiple: true,
			},
			{
				Tag:          "--smf",
				Description:  "SMF name to register with",
				Type:         "string",
				Required:     false,
				AllowMutiple: true,
			},
			{
				Tag:          "--help",
				Description:  "Show help information",
				Type:         "flag",
				Required:     false,
				AllowMutiple: false,
			},
		},
	}

	// Setup deregister command
	deregisterCommand := models.Command{
		Name:  "deregister",
		Help:  "Sign out the UE from Core",
		Usage: "Usage: deregister [--amf] [--smf] [--help] [--force]",
		Func:  deregisterHandler,
		Arguments: []models.Argument{
			{
				Tag:          "--amf",
				Description:  "AMF name to deregister from",
				Type:         "string",
				Required:     false,
				AllowMutiple: true,
			},
			{
				Tag:          "--smf",
				Description:  "SMF name to deregister from",
				Type:         "string",
				Required:     false,
				AllowMutiple: true,
			},
			{
				Tag:          "--force",
				Description:  "Force deregistration",
				Type:         "flag",
				Required:     false,
				AllowMutiple: false,
			},
			{
				Tag:          "--help",
				Description:  "Show help information",
				Type:         "flag",
				Required:     false,
				AllowMutiple: false,
			},
		},
	}

	// Set commands on the node
	ueNode.Commands = []models.Command{registerCommand, deregisterCommand}
}

// SetupShellGnb configures gNB commands with their handlers
func (s *Shell) SetupShellGnb(fn func(map[string]string), args map[string]string) {
	// Find gNB node
	var gnbNode *Node
	for i, node := range s.Nodes {
		if node.Type == Gnb {
			gnbNode = &s.Nodes[i]
			break
		}
	}

	if gnbNode == nil {
		// Create new gNB node if not found
		gnbNode = &Node{
			Type:        Gnb,
			Name:        "GNB-Default",
			ActiveNodes: make(map[string][]string),
			Commands:    []models.Command{},
			Shell:       ishell.New(),
		}
		s.Nodes = append(s.Nodes, *gnbNode)
	}

	// Define handlers
	amfInfoHandler := func(arguments map[string]string) string {
		// Process the arguments
		nodeName := gnbNode.Name

		// Handle --help flag
		if _, ok := arguments["help"]; ok {
			return "Usage: amf-info [--status] [--detail] [amf-name]\n" +
				"--status : Show AMF status\n" +
				"--detail : Show AMF detail"
		}

		// Handle status query
		if _, ok := arguments["status"]; ok {
			if amfValues, ok := arguments["amf-name"]; ok {
				amfs := strings.Split(amfValues, ",")
				formattedAmfs := strings.Join(amfs, ", ")
				return fmt.Sprintf("%s Status for gNodeB %s: Connected and operational", formattedAmfs, nodeName)
			}
			return fmt.Sprintf("All AMFs Status for gNodeB %s: Connected and operational", nodeName)
		}

		// Handle detail query
		if _, ok := arguments["detail"]; ok {
			if amfValues, ok := arguments["amf-name"]; ok {
				amfs := strings.Split(amfValues, ",")
				formattedAmfs := strings.Join(amfs, ", ")
				return fmt.Sprintf("%s Detail for gNodeB %s: Capacity=85%%", formattedAmfs, nodeName)
			}
			return fmt.Sprintf("All AMFs Detail for gNodeB %s: Capacity=85%%", nodeName)
		}

		return "Error: No valid arguments provided"
	}

	amfListHandler := func(arguments map[string]string) string {
		// Process the arguments
		nodeName := gnbNode.Name

		// Handle --help flag
		if _, ok := arguments["help"]; ok {
			return "Usage: amf-list [--all] [--active]\n" +
				"--all     : List all AMFs\n" +
				"--active  : List active AMFs"
		}

		// Handle all AMFs listing
		if _, ok := arguments["all"]; ok {
			if allAmfs, ok := gnbNode.AllNodes["amf"]; ok && len(allAmfs) > 0 {
				formattedAmfs := strings.Join(allAmfs, " ")
				return fmt.Sprintf("AMF List for gNodeB %s: %s", nodeName, formattedAmfs)
			}
			return fmt.Sprintf("No AMFs found for gNodeB %s", nodeName)
		}

		// Handle active AMFs listing
		if _, ok := arguments["active"]; ok {
			if activeAmfs, ok := gnbNode.ActiveNodes["amf"]; ok && len(activeAmfs) > 0 {
				formattedAmfs := strings.Join(activeAmfs, " ")
				return fmt.Sprintf("Active AMFs connected to gNodeB %s: %s", nodeName, formattedAmfs)
			}
			return fmt.Sprintf("No active AMFs connected to gNodeB %s", nodeName)
		}

		return "Error: No valid arguments provided"
	}

	// Setup amf-info command
	amfInfoCommand := models.Command{
		Name:  "amf-info",
		Help:  "Show information about AMFs",
		Usage: "Usage: amf-info [--status] [--detail] [amf-name]",
		Func:  amfInfoHandler,
		Arguments: []models.Argument{
			{
				Tag:          "--status",
				Description:  "Show AMF status",
				Type:         "flag",
				Required:     false,
				AllowMutiple: false,
			},
			{
				Tag:          "--detail",
				Description:  "Show AMF details",
				Type:         "flag",
				Required:     false,
				AllowMutiple: false,
			},
			{
				Tag:          "--amf",
				Description:  "AMF name to show info for",
				Type:         "string",
				Required:     false,
				AllowMutiple: true,
			},
			{
				Tag:          "--help",
				Description:  "Show help information",
				Type:         "flag",
				Required:     false,
				AllowMutiple: false,
			},
		},
	}

	// Setup amf-list command
	amfListCommand := models.Command{
		Name:  "amf-list",
		Help:  "List AMFs connected to this gNB",
		Usage: "Usage: amf-list [--all] [--active] [--help]",
		Func:  amfListHandler,
		Arguments: []models.Argument{
			{
				Tag:          "--all",
				Description:  "List all AMFs",
				Type:         "flag",
				Required:     false,
				AllowMutiple: false,
			},
			{
				Tag:          "--active",
				Description:  "List active AMFs only",
				Type:         "flag",
				Required:     false,
				AllowMutiple: false,
			},
			{
				Tag:          "--help",
				Description:  "Show help information",
				Type:         "flag",
				Required:     false,
				AllowMutiple: false,
			},
		},
	}

	// Set commands on the node
	gnbNode.Commands = []models.Command{amfInfoCommand, amfListCommand}
}

func (s *Shell) SetupServer() {
	r := gin.Default()

	// API to get available node types
	r.GET("/nodes/types", func(c *gin.Context) {
		types := []string{"ue", "gnb"}
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

	//API to get available commands for a specific node type
	r.GET("/commands/:nodeType", func(c *gin.Context) {
		nodeType := c.Param("nodeType")
		var commands []map[string]string

		for _, node := range s.Nodes {
			if strings.EqualFold(node.Type.String(), nodeType) {
				// Convert Command to the format expected by the client
				for _, cmd := range node.Commands {
					command := map[string]string{
						"name":  cmd.Name,
						"help":  cmd.Help,
						"usage": cmd.Usage,
					}
					commands = append(commands, command)
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
		// Find the appropriate node
		var targetNode *Node
		for i, node := range s.Nodes {
			if strings.EqualFold(node.Type.String(), formArgs.NodeType) &&
				(formArgs.NodeName == "" || node.Name == formArgs.NodeName) {
				targetNode = &s.Nodes[i]
				break
			}
		}

		if targetNode == nil {
			c.JSON(http.StatusNotFound, gin.H{
				"result": fmt.Sprintf("Error: Node of type %s not found", formArgs.NodeType),
			})
			return
		}

		// Find the command
		var targetCmd *models.Command
		for i, cmd := range targetNode.Commands {
			if cmd.Name == formArgs.Command {
				targetCmd = &targetNode.Commands[i]
				break
			}
		}

		if targetCmd == nil {
			c.JSON(http.StatusNotFound, gin.H{
				"result": fmt.Sprintf("Error: Command %s not found for node type %s", formArgs.Command, formArgs.NodeType),
			})
			return
		}

		// Execute the command handler function
		if targetCmd.Func != nil {
			result := targetCmd.Func(formArgs.Arguments)
			c.JSON(http.StatusOK, gin.H{
				"result": result,
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"result": "Error: Command handler not implemented",
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

func main() {
	server := NewServer("0.0.0.0", 4000)

	// Setup command handlers for each node type
	dummyFunction := func(args map[string]string) {}
	dummyArgs := make(map[string]string)

	server.SetupShellUE(dummyFunction, dummyArgs)
	server.SetupShellGnb(dummyFunction, dummyArgs)

	// Setup example All nodes and Active nodes
	for i, node := range server.Nodes {
		switch node.Type {
		case UE:
			// All nodes that UE knows about
			server.Nodes[i].AllNodes = map[string][]string{
				"amf": {"AMF-01", "AMF-02", "AMF-03", "AMF-04"},
				"gnb": {"GNB-001", "GNB-002", "GNB-003"},
			}
			// Active nodes that UE is connected to
			server.Nodes[i].ActiveNodes = map[string][]string{
				"amf": {"AMF-01", "AMF-02"},
				"gnb": {"GNB-001"},
			}
		case Gnb:
			// All nodes that GNB knows about
			server.Nodes[i].AllNodes = map[string][]string{
				"amf": {"AMF-01", "AMF-02", "AMF-03", "AMF-04", "AMF-05"},
				"ue":  {"UE-001", "UE-002", "UE-003", "UE-004"},
			}
			// Active nodes that GNB is connected to
			server.Nodes[i].ActiveNodes = map[string][]string{
				"amf": {"AMF-01", "AMF-03"},
				"ue":  {"UE-001", "UE-002"},
			}
		}
	}

	// Setup the server
	server.SetupServer()

	fmt.Println("Server started at 0.0.0.0:4000")
	fmt.Println("Available object types: ue, gnb")

	// Keep the main goroutine alive
	select {}
}
