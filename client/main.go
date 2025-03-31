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

	"github.com/abiosoft/ishell"
)

// CommandInfo - Định nghĩa cấu trúc thông tin lệnh giống với server
type CommandInfo struct {
	Name        string        `json:"name"`
	Usage       string        `json:"usage"`
	Description string        `json:"description"`
	ArgsUsage   string        `json:"argsUsage"`
	Flags       []FlagInfo    `json:"flags"`
	Subcommands []CommandInfo `json:"subcommands,omitempty"`
}

// FlagInfo - Định nghĩa cấu trúc thông tin flag giống với server
type FlagInfo struct {
	Name        string `json:"name"`
	Usage       string `json:"usage"`
	DefaultText string `json:"defaultText,omitempty"`
	Required    bool   `json:"required"`
}

// CommandRequest - Cấu trúc yêu cầu thực thi lệnh
type CommandRequest struct {
	NodeType    string            `json:"nodeType"`
	NodeName    string            `json:"nodeName"`
	CommandPath string            `json:"commandPath"`
	RawCommand  string            `json:"rawCommand"`
	Args        []string          `json:"args,omitempty"`
	Flags       map[string]string `json:"flags,omitempty"`
}

// Structure of context stack to organize context
type Context struct {
	Type      string // "root", "server", "context_type", "node"
	Name      string // tên của context: emulator, ue, gnb, hoặc tên node cụ thể
	ServerURL string // URL của server đang kết nối
}

var (
	serverURL    string
	contextStack []Context
)

func main() {
	flag.Parse()

	shell := ishell.New()

	// Khởi tạo context stack với root context
	contextStack = []Context{{Type: "root", Name: "root", ServerURL: ""}}

	setupRootCommands(shell)

	shell.Println("Interactive CLI Client")
	shell.SetPrompt(">>> ")

	// Bắt đầu shell
	shell.Run()
}

// setupRootCommands thiết lập các lệnh ở root context
func setupRootCommands(shell *ishell.Shell) {

	shell.AddCmd(&ishell.Cmd{
		Name: "help",
		Help: "display help",
		Func: func(c *ishell.Context) {
			c.Println("Commands:")
			c.Println("  clear        clear the screen")
			c.Println("  connect      Connect to a server [connect http://localhost:4000]")
			c.Println("  exit         exit the program")
			c.Println("  help         display help")
		},
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
			if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
				url = "http://" + url
			}

			resp, err := http.Get(url + "/api/context")
			if err != nil {
				c.Printf("Failed to connect to server: %v\n", err)
				return
			}
			defer resp.Body.Close()

			// Chuyển sang server context
			serverURL = url
			contextStack = append(contextStack, Context{Type: "server", Name: "server", ServerURL: url})
			setupServerCommands(shell)
			c.Printf("Connected to server: %s, type help to see commands\n", url)
		},
	})
}

// setupServerCommands thiết lập các lệnh khi đã kết nối tới server
func setupServerCommands(shell *ishell.Shell) {

	shell.AddCmd(&ishell.Cmd{
		Name: "help",
		Help: "Display help",
		Func: func(c *ishell.Context) {
			c.Println("Commands:")
			c.Println("  back                 Go back to previous context")
			c.Println("  clear                Clear the screen")
			c.Println("  disconnect           Disconnect server")
			c.Println("  exit                 Exit the client")
			c.Println("  help                 Display help")
			c.Println("  list                 List available objects [list ue|gnb]")
			c.Println("  use                  Select a context to use [use emulator|ue|gnb]")
		},
	})

	shell.AddCmd(&ishell.Cmd{
		Name: "clear",
		Help: "Clear the screen",
		Func: func(c *ishell.Context) {
			c.ClearScreen()
		},
	})

	shell.AddCmd(&ishell.Cmd{
		Name: "exit",
		Help: "Exit the client",
		Func: func(c *ishell.Context) {
			c.Println("Goodbye!")
			os.Exit(0)
		},
	})

	shell.AddCmd(&ishell.Cmd{
		Name: "disconnect",
		Help: "Disconnect server",
		Func: func(c *ishell.Context) {
			c.Printf("Disconnected from %s\n", serverURL)
			serverURL = ""
			// Trở về root context
			contextStack = contextStack[:1] // Chỉ giữ lại root context
			setupRootCommands(shell)
			shell.SetPrompt(">>> ")
		},
	})

	shell.AddCmd(&ishell.Cmd{
		Name: "back",
		Help: "Go back to previous context",
		Func: func(c *ishell.Context) {
			if len(contextStack) > 1 {
				// Xóa context hiện tại
				contextStack = contextStack[:len(contextStack)-1]
				previousContext := contextStack[len(contextStack)-1]

				// Setup lại commands dựa trên context trước đó
				if previousContext.Type == "root" {
					setupRootCommands(shell)
					shell.SetPrompt(">>> ")
				} else if previousContext.Type == "server" {
					setupServerCommands(shell)
					shell.SetPrompt(">>> ")
				} else if previousContext.Type == "context_type" {
					setupContextTypeCommands(shell, previousContext.Name)
					shell.SetPrompt(previousContext.Name + " >>> ")
				}
			} else {
				c.Println("Already at root context")
			}
		},
	})

	// Thêm lệnh list
	shell.AddCmd(&ishell.Cmd{
		Name: "list",
		Help: "List available objects [list ue|gnb]",
		Func: func(c *ishell.Context) {
			if len(c.Args) < 1 {
				c.Println("Usage: list <type>")
				c.Println("Types: ue, gnb")
				return
			}

			objType := c.Args[0]
			objects, err := getObjectsByType(objType)
			if err != nil {
				c.Printf("Error: %v\n", err)
				return
			}

			for _, obj := range objects {
				c.Printf("  - %s\n", obj)
			}
		},
	})

	// Thêm lệnh use
	shell.AddCmd(&ishell.Cmd{
		Name: "use",
		Help: "Select a context to use [use emulator | ue | gnb]",
		Func: func(c *ishell.Context) {
			if len(c.Args) < 1 {
				c.Println("Usage: use <context-type>")
				c.Println("Context types: emulator, ue, gnb")
				return
			}

			contextType := c.Args[0]
			if contextType != "emulator" && contextType != "ue" && contextType != "gnb" {
				c.Println("Invalid context type. Use 'emulator', 'ue', or 'gnb'")
				return
			}

			// Nếu chọn context type
			if contextType == "ue" || contextType == "gnb" {
				// Lấy danh sách đối tượng
				objects, err := getObjectsByType(contextType)
				if err != nil {
					c.Printf("Error: %v\n", err)
					return
				}

				c.Printf("Available %s objects:\n", contextType)
				for _, obj := range objects {
					c.Printf("  - %s\n", obj)
				}

				// Thêm context mới vào stack
				contextStack = append(contextStack, Context{Type: "context_type", Name: contextType, ServerURL: serverURL})
				setupContextTypeCommands(shell, contextType)
				shell.SetPrompt(contextType + " >>> ")
			} else {
				// Đối với emulator, chuyển trực tiếp đến node
				c.Println("Switching to emulator context")
				// Lấy commands và thiết lập shell
				setupNodeCommands(shell, "emulator", "emulator")
				shell.SetPrompt("emulator >>> ")
				// Thêm context vào stack
				contextStack = append(contextStack,
					Context{Type: "context_type", Name: "emulator", ServerURL: serverURL},
					Context{Type: "node", Name: "emulator", ServerURL: serverURL})
			}
		},
	})
}

// setupContextTypeCommands thiết lập các lệnh cho context type (ue hoặc gnb)
func setupContextTypeCommands(shell *ishell.Shell, contextType string) {
	shell.AddCmd(&ishell.Cmd{
		Name: "help",
		Help: "display this help",
		Func: func(c *ishell.Context) {
			c.Println("Available commands :")
			c.Println("  select       Select a node to interact with [select <node-name>]")
			c.Println("")
			c.Println("General commands:")
			c.Println("  back         Go back to previous context")
			c.Println("  clear        Clear the screen")
			c.Println("  exit         Exit the client")
			c.Println("  help         Display this help")
		},
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
		Help: "Exit the client",
		Func: func(c *ishell.Context) {
			c.Println("Goodbye!")
			os.Exit(0)
		},
	})

	shell.AddCmd(&ishell.Cmd{
		Name: "back",
		Help: "Go back to previous context",
		Func: func(c *ishell.Context) {
			if len(contextStack) > 1 {
				// Xóa context hiện tại
				contextStack = contextStack[:len(contextStack)-1]
				previousContext := contextStack[len(contextStack)-1]

				// Setup lại commands dựa trên context trước đó
				if previousContext.Type == "server" {
					setupServerCommands(shell)
					shell.SetPrompt(">>> ")
				} else if previousContext.Type == "context_type" {
					setupContextTypeCommands(shell, previousContext.Name)
					shell.SetPrompt(previousContext.Name + " >>> ")
				}
			}
		},
	})

	shell.AddCmd(&ishell.Cmd{
		Name: "select",
		Help: "Select a node to interact with [select <node-name>]",
		Func: func(c *ishell.Context) {
			if len(c.Args) < 1 {
				c.Println("Usage: select <node-name>")
				return
			}

			nodeName := c.Args[0]
			objects, err := getObjectsByType(contextType)
			if err != nil {
				c.Printf("Error: %v\n", err)
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
				c.Printf("Node '%s' not found\n", nodeName)
				return
			}

			// Thiết lập commands cho node
			setupNodeCommands(shell, contextType, nodeName)
			contextStack = append(contextStack, Context{Type: "node", Name: nodeName, ServerURL: serverURL})
			shell.SetPrompt(nodeName + " >>> ")
		},
	})
}

// setupNodeCommands thiết lập các lệnh cho một node cụ thể
func setupNodeCommands(shell *ishell.Shell, nodeType, nodeName string) {
	// Lấy commands từ server
	commands, err := getCommands(nodeType, nodeName)
	if err != nil {
		fmt.Printf("Error getting commands: %v\n", err)
		return
	}

	shell.AddCmd(&ishell.Cmd{
		Name: "help",
		Help: "display this help",
		Func: func(c *ishell.Context) {
			c.Printf("Available commands for %s :\n", nodeName)
			// Hiển thị các lệnh chuyên biệt
			for _, cmd := range commands.Subcommands {
				c.Printf("  %-12s %s\n", cmd.Name, cmd.Usage)
			}
			c.Println("")
			c.Println("General commands:")
			c.Println("  back        Go back to previous context")
			c.Println("  clear        clear the screen")
			c.Println("  exit         Exit the client")
			c.Println("  help         display this help")
		},
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
		Help: "Exit the client",
		Func: func(c *ishell.Context) {
			c.Println("Goodbye!")
			os.Exit(0)
		},
	})

	shell.AddCmd(&ishell.Cmd{
		Name: "back",
		Help: "Go back to previous context",
		Func: func(c *ishell.Context) {
			if len(contextStack) > 1 {
				// Xóa context hiện tại
				contextStack = contextStack[:len(contextStack)-1]
				previousContext := contextStack[len(contextStack)-1]

				// Setup lại commands dựa trên context trước đó
				if previousContext.Type == "context_type" {
					setupContextTypeCommands(shell, previousContext.Name)
					shell.SetPrompt(previousContext.Name + " >>> ")
				} else if previousContext.Type == "server" {
					setupServerCommands(shell)
					shell.SetPrompt(">>> ")
				}
			}
		},
	})

	// Thêm các lệnh từ server
	for _, cmd := range commands.Subcommands {
		cmdInfo := cmd
		shell.AddCmd(&ishell.Cmd{
			Name:     cmdInfo.Name,
			Help:     cmdInfo.Usage,
			LongHelp: generateLongHelp(cmdInfo),
			Func: func(c *ishell.Context) {

				if len(c.Args) > 0 && c.Args[0] == "--help" {
					c.Println(generateLongHelp(cmdInfo))
					return
				}

				result, err := execCmd(nodeType, nodeName, cmdInfo.Name, c.Args)
				if err != nil {
					c.Printf("Error: %v\n", err)
					return
				}
				c.Println(result)
			},
		})
	}
}

func getObjectsByType(objType string) ([]string, error) {
	resp, err := http.Get(fmt.Sprintf("%s/api/context/%s", serverURL, objType))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to server: %v", err)
	}
	defer resp.Body.Close()

	var response struct {
		Type    string   `json:"type"`
		Objects []string `json:"objects"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}

	return response.Objects, nil
}

func getCommands(nodeType, nodeName string) (CommandInfo, error) {
	var url string
	switch nodeType {
	case "emulator":
		url = fmt.Sprintf("%s/api/emulator/commands", serverURL)
	case "ue":
		url = fmt.Sprintf("%s/api/ue/%s/commands", serverURL, nodeName)
	case "gnb":
		url = fmt.Sprintf("%s/api/gnb/%s/commands", serverURL, nodeName)
	default:
		return CommandInfo{}, fmt.Errorf("invalid node type: %s", nodeType)
	}

	resp, err := http.Get(url)
	if err != nil {
		return CommandInfo{}, fmt.Errorf("failed to connect to server: %v", err)
	}
	defer resp.Body.Close()

	var response struct {
		Commands CommandInfo `json:"commands"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return CommandInfo{}, fmt.Errorf("failed to parse response: %v", err)
	}

	return response.Commands, nil
}

func execCmd(nodeType, nodeName, cmdName string, args []string) (string, error) {
	cmdReq := CommandRequest{
		NodeType:    nodeType,
		NodeName:    nodeName,
		CommandPath: cmdName,
		Args:        args,
	}

	return sendCmd(cmdReq)
}

func sendCmd(cmdReq CommandRequest) (string, error) {
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

	var response struct {
		Response string `json:"response"`
		Error    string `json:"error"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return "", fmt.Errorf("failed to parse response: %v", err)
	}

	if response.Error != "" {
		return "", fmt.Errorf("server error: %s", response.Error)
	}

	return response.Response, nil
}

// generateLongHelp tạo help chi tiết cho lệnh
func generateLongHelp(cmd CommandInfo) string {
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
