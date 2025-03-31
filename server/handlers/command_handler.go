package handlers

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/urfave/cli/v3"
)

// Interfaces  API
type EmulatorApi interface {
	ListUes() []string
	ListGnbs() []string
	AddUe(supi string, triggerRegister bool) bool
}

type UeApi interface {
	Register(isEmergency bool) bool
	Deregister(deregisterType uint8) bool
	CreateSession(slice string, dnName string, sessionType uint8) bool
}

type GnbApi interface {
	ReleaseUe(ueId string) bool
	ReleaseSession(ueId string, sessionId uint8) bool
}

// CommandHandler - Process commands for emulator, UE, and gNB
type CommandHandler struct {
	eApi   EmulatorApi
	uApi   UeApi
	gApi   GnbApi
	emuCmd *cli.Command
	ueCmd  *cli.Command
	gnbCmd *cli.Command
}

// CommandInfo - Basic Info Command to pass through API
type CommandInfo struct {
	Name        string        `json:"name"`
	Usage       string        `json:"usage"`
	Description string        `json:"description"`
	ArgsUsage   string        `json:"argsUsage"`
	Flags       []FlagInfo    `json:"flags"`
	Subcommands []CommandInfo `json:"subcommands,omitempty"`
}

// FlagInfo - Thông tin flag đơn giản để truyền qua API
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

func NewCommandHandler(eApi EmulatorApi, uApi UeApi, gApi GnbApi) *CommandHandler {
	h := &CommandHandler{
		eApi: eApi,
		uApi: uApi,
		gApi: gApi,
	}

	// Khởi tạo command cho emulator
	h.initEmulatorCommands()

	// Khởi tạo command cho UE
	h.initUeCommands()

	// Khởi tạo command cho GNB
	h.initGnbCommands()

	return h
}

func (h *CommandHandler) initEmulatorCommands() {
	h.emuCmd = &cli.Command{
		Name:        "emulator",
		Usage:       "Emulator management commands",
		Description: "Commands to manage and interact with the emulator",
		Commands: []*cli.Command{
			{
				Name:  "list-ue",
				Usage: "List all UEs",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					rspCh := ctx.Value("rsp").(chan string)
					ues := h.eApi.ListUes()
					rspCh <- strings.Join(ues, " ")
					return nil
				},
			},
			{
				Name:  "list-gnb",
				Usage: "List all GnBs",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					rspCh := ctx.Value("rsp").(chan string)
					gnbs := h.eApi.ListGnbs()
					rspCh <- strings.Join(gnbs, " ")
					return nil
				},
			},
			{
				Name:        "add-ue",
				Usage:       "Add a new UE with SUPI",
				ArgsUsage:   "<supi>",
				Description: "Add a new UE to the emulator with the specified SUPI",
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:  "register",
						Usage: "Trigger registration after adding",
					},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					rspCh := ctx.Value("rsp").(chan string)
					args := cmd.Args().Slice()
					if len(args) < 1 {
						rspCh <- "Error: SUPI is required"
						return nil
					}
					supi := args[0]
					register := cmd.Bool("register")
					success := h.eApi.AddUe(supi, register)
					if success {
						rspCh <- "UE added successfully"
					} else {
						rspCh <- "Failed to add UE"
					}
					return nil
				},
			},
		},
	}
}

func (h *CommandHandler) initUeCommands() {
	h.ueCmd = &cli.Command{
		Name:        "ue",
		Usage:       "UE management commands",
		Description: "Commands to manage and interact with UEs",
		Commands: []*cli.Command{
			{
				Name:        "register",
				Usage:       "Register UE to the network",
				Description: "Register the UE to the network with optional emergency services",
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:  "emergency",
						Usage: "Register for emergency services",
					},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					rspCh := ctx.Value("rsp").(chan string)
					isEmergency := cmd.Bool("emergency")
					success := h.uApi.Register(isEmergency)
					if success {
						rspCh <- "UE registered successfully"
					} else {
						rspCh <- "Failed to register UE"
					}
					return nil
				},
			},
			{
				Name:        "deregister",
				Usage:       "Deregister UE from the network",
				Description: "Deregister the UE from the network with specified type",
				Flags: []cli.Flag{
					&cli.IntFlag{
						Name:  "type",
						Usage: "Deregistration type (0-3)",
						Value: 0,
					},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					rspCh := ctx.Value("rsp").(chan string)
					deregType := uint8(cmd.Int("type"))
					success := h.uApi.Deregister(deregType)
					if success {
						rspCh <- "UE deregistered successfully"
					} else {
						rspCh <- "Failed to deregister UE"
					}
					return nil
				},
			},
			{
				Name:        "create-session",
				Usage:       "Create a new session",
				Description: "Create a new PDU session with specified parameters",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "slice",
						Usage: "Network slice",
						Value: "default",
					},
					&cli.StringFlag{
						Name:  "dn",
						Usage: "Data Network name",
						Value: "internet",
					},
					&cli.IntFlag{
						Name:  "type",
						Usage: "Session type (0-3)",
						Value: 0,
					},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					rspCh := ctx.Value("rsp").(chan string)
					slice := cmd.String("slice")
					dn := cmd.String("dn")
					sessionType := uint8(cmd.Int("type"))
					success := h.uApi.CreateSession(slice, dn, sessionType)
					if success {
						rspCh <- "Session created successfully"
					} else {
						rspCh <- "Failed to create session"
					}
					return nil
				},
			},
		},
	}
}

func (h *CommandHandler) initGnbCommands() {
	h.gnbCmd = &cli.Command{
		Name:        "gnb",
		Usage:       "gNB management commands",
		Description: "Commands to manage and interact with gNBs",
		Commands: []*cli.Command{
			{
				Name:        "release-ue",
				Usage:       "Release a UE from the gNB",
				ArgsUsage:   "<ue-id>",
				Description: "Release a UE connection from the gNB",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					rspCh := ctx.Value("rsp").(chan string)
					args := cmd.Args().Slice()
					if len(args) < 1 {
						rspCh <- "Error: UE ID is required"
						return nil
					}
					ueId := args[0]
					success := h.gApi.ReleaseUe(ueId)
					if success {
						rspCh <- "UE released successfully"
					} else {
						rspCh <- "Failed to release UE"
					}
					return nil
				},
			},
			{
				Name:        "release-session",
				Usage:       "Release a session",
				ArgsUsage:   "<ue-id>",
				Description: "Release a PDU session for the specified UE",
				Flags: []cli.Flag{
					&cli.IntFlag{
						Name:  "id",
						Usage: "Session ID",
						Value: 1,
					},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					rspCh := ctx.Value("rsp").(chan string)
					args := cmd.Args().Slice()
					if len(args) < 1 {
						rspCh <- "Error: UE ID is required"
						return nil
					}
					ueId := args[0]
					sessionId := uint8(cmd.Int("id"))
					success := h.gApi.ReleaseSession(ueId, sessionId)
					if success {
						rspCh <- "Session released successfully"
					} else {
						rspCh <- "Failed to release session"
					}
					return nil
				},
			},
		},
	}
}

func convertCommandToInfo(cmd *cli.Command) CommandInfo {
	var flags []FlagInfo
	for _, flag := range cmd.Flags {
		names := flag.Names()
		usage := ""
		defaultText := ""
		required := false

		// Kiểm tra nếu flag có phương thức GetRequired()
		if f, ok := flag.(interface{ GetRequired() bool }); ok {
			required = f.GetRequired()
		}

		// Xử lý từng loại flag
		switch f := flag.(type) {
		case *cli.StringFlag:
			usage = f.Usage
			defaultText = f.Value
		case *cli.BoolFlag:
			usage = f.Usage
			if f.Value {
				defaultText = "true"
			}
		case *cli.IntFlag:
			usage = f.Usage
			if f.Value != 0 {
				defaultText = fmt.Sprintf("%d", f.Value)
			}
		default:
			if usageField := reflect.ValueOf(flag).Elem().FieldByName("Usage"); usageField.IsValid() {
				usage = usageField.String()
			}
		}

		flags = append(flags, FlagInfo{
			Name:        strings.Join(names, ", "),
			Usage:       usage,
			DefaultText: defaultText,
			Required:    required,
		})
	}

	var subcommands []CommandInfo
	for _, subcmd := range cmd.Commands {
		subcommands = append(subcommands, convertCommandToInfo(subcmd))
	}

	return CommandInfo{
		Name:        cmd.Name,
		Usage:       cmd.Usage,
		Description: cmd.Description,
		ArgsUsage:   cmd.ArgsUsage,
		Flags:       flags,
		Subcommands: subcommands,
	}
}

func (h *CommandHandler) GetEmulatorCommands(c *gin.Context) {
	commands := convertCommandToInfo(h.emuCmd)
	c.JSON(http.StatusOK, gin.H{
		"commands": commands,
	})
}

func (h *CommandHandler) GetUeCommands(c *gin.Context) {
	ueId := c.Param("ueId")
	if ueId == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "UE ID is required",
		})
		return
	}

	commands := convertCommandToInfo(h.ueCmd)
	c.JSON(http.StatusOK, gin.H{
		"commands": commands,
		"ueId":     ueId,
	})
}

func (h *CommandHandler) GetGnbCommands(c *gin.Context) {
	gnbId := c.Param("gnbId")
	if gnbId == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "GNB ID is required",
		})
		return
	}

	commands := convertCommandToInfo(h.gnbCmd)
	c.JSON(http.StatusOK, gin.H{
		"commands": commands,
		"gnbId":    gnbId,
	})
}

func (h *CommandHandler) ListUes(c *gin.Context) {
	ues := h.eApi.ListUes()
	c.JSON(http.StatusOK, gin.H{
		"ues": ues,
	})
}

func (h *CommandHandler) ListGnbs(c *gin.Context) {
	gnbs := h.eApi.ListGnbs()
	c.JSON(http.StatusOK, gin.H{
		"gnbs": gnbs,
	})
}

// Process commands
func (h *CommandHandler) ExecuteCommand(c *gin.Context) {
	var req CommandRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	// Tạo context và response channel
	rspCh := make(chan string, 1)
	ctx := context.WithValue(context.Background(), "rsp", rspCh)

	// Handle command args and flags
	var cmdArgs []string

	// Parse raw command or use args + flags đã chỉ định
	if req.RawCommand != "" {
		cmdArgs = strings.Fields(req.RawCommand)
	} else {
		// Use command path
		cmdArgs = []string{req.CommandPath}

		// Add arguments
		if len(req.Args) > 0 {
			cmdArgs = append(cmdArgs, req.Args...)
		}

		// Add flags
		for name, value := range req.Flags {
			if value != "" {
				cmdArgs = append(cmdArgs, "--"+name, value)
			} else {
				cmdArgs = append(cmdArgs, "--"+name)
			}
		}
	}

	// Execute commands by the type of node
	var err error

	switch req.NodeType {
	case "emulator":
		err = h.emuCmd.Run(ctx, append([]string{"emulator"}, cmdArgs...))
	case "ue":
		err = h.ueCmd.Run(ctx, append([]string{"ue"}, cmdArgs...))
	case "gnb":
		err = h.gnbCmd.Run(ctx, append([]string{"gnb"}, cmdArgs...))
	default:
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid node type",
		})
		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	//Read the response from channel
	response := <-rspCh
	c.JSON(http.StatusOK, gin.H{
		"response": response,
	})
}
