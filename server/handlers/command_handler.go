package handlers

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"test_monitor/models"

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

// CommandStore manages command definitions and executions
type CommandStore struct {
	eApi EmulatorApi
	uApi UeApi
	gApi GnbApi

	// Command definitions
	emuCmd *cli.Command
	ueCmd  *cli.Command
	gnbCmd *cli.Command

	// Cache of command info by node type
	commandCache map[string][]models.CommandInfo
}

// NewCommandStore creates a new command store
func NewCommandStore(eApi EmulatorApi, uApi UeApi, gApi GnbApi) *CommandStore {
	store := &CommandStore{
		eApi:         eApi,
		uApi:         uApi,
		gApi:         gApi,
		commandCache: make(map[string][]models.CommandInfo),
	}

	store.initCommands()

	return store
}

// initCommands initializes all command definitions
func (s *CommandStore) initCommands() {
	// Initialize emulator commands
	s.emuCmd = &cli.Command{
		Name:        "emulator",
		Usage:       "Emulator management commands",
		Description: "Commands to manage and interact with the emulator",
		Commands: []*cli.Command{
			{
				Name:  "list-ue",
				Usage: "List all UEs",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					rspCh := ctx.Value("rsp").(chan string)
					ues := s.eApi.ListUes()
					rspCh <- strings.Join(ues, "\n")
					return nil
				},
			},
			{
				Name:  "list-gnb",
				Usage: "List all GnBs",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					rspCh := ctx.Value("rsp").(chan string)
					gnbs := s.eApi.ListGnbs()
					rspCh <- strings.Join(gnbs, "\n")
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
					success := s.eApi.AddUe(supi, register)
					if success {
						rspCh <- fmt.Sprintf("UE %s added successfully to emulator", supi)
					} else {
						rspCh <- fmt.Sprintf("Failed to add UE %s to emulator", supi)
					}
					return nil
				},
			},
		},
	}

	// Initialize UE commands
	s.ueCmd = &cli.Command{
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
					nodeName, hasNode := GetNodeName(ctx)
					isEmergency := cmd.Bool("emergency")
					success := s.uApi.Register(isEmergency)
					if success {
						if hasNode {
							rspCh <- fmt.Sprintf("UE %s registered successfully", nodeName)
						} else {
							rspCh <- "UE registered successfully"
						}
					} else {
						if hasNode {
							rspCh <- fmt.Sprintf("Failed to register UE %s", nodeName)
						} else {
							rspCh <- "Failed to register UE"
						}
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
					nodeName, hasNode := GetNodeName(ctx)
					deregType := uint8(cmd.Int("type"))
					success := s.uApi.Deregister(deregType)
					if success {
						if hasNode {
							rspCh <- fmt.Sprintf("UE %s deregistered successfully", nodeName)
						} else {
							rspCh <- "UE deregistered successfully"
						}
					} else {
						if hasNode {
							rspCh <- fmt.Sprintf("Failed to deregister UE %s", nodeName)
						} else {
							rspCh <- "Failed to deregister UE"
						}
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
					nodeName, hasNode := GetNodeName(ctx)
					slice := cmd.String("slice")
					dn := cmd.String("dn")
					sessionType := uint8(cmd.Int("type"))
					success := s.uApi.CreateSession(slice, dn, sessionType)
					if success {
						if hasNode {
							rspCh <- fmt.Sprintf("Session created successfully for UE %s", nodeName)
						} else {
							rspCh <- "Session created successfully"
						}
					} else {
						if hasNode {
							rspCh <- fmt.Sprintf("Failed to create session for UE %s", nodeName)
						} else {
							rspCh <- "Failed to create session"
						}
					}
					return nil
				},
			},
		},
	}

	// Initialize GNB commands
	s.gnbCmd = &cli.Command{
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
					nodeName, hasNode := GetNodeName(ctx)
					args := cmd.Args().Slice()
					if len(args) < 1 {
						rspCh <- "Error: UE ID is required"
						return nil
					}
					ueId := args[0]
					success := s.gApi.ReleaseUe(ueId)
					if success {
						if hasNode {
							rspCh <- fmt.Sprintf("UE %s released successfully from gNB %s", ueId, nodeName)
						} else {
							rspCh <- fmt.Sprintf("UE %s released successfully", ueId)
						}
					} else {
						if hasNode {
							rspCh <- fmt.Sprintf("Failed to release UE %s from gNB %s", ueId, nodeName)
						} else {
							rspCh <- fmt.Sprintf("Failed to release UE %s", ueId)
						}
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
					nodeName, hasNode := GetNodeName(ctx)
					args := cmd.Args().Slice()
					if len(args) < 1 {
						rspCh <- "Error: UE ID is required"
						return nil
					}
					ueId := args[0]
					sessionId := uint8(cmd.Int("id"))
					success := s.gApi.ReleaseSession(ueId, sessionId)
					if success {
						if hasNode {
							rspCh <- fmt.Sprintf("Session %d for UE %s released successfully from gNB %s",
								sessionId, ueId, nodeName)
						} else {
							rspCh <- fmt.Sprintf("Session %d for UE %s released successfully",
								sessionId, ueId)
						}
					} else {
						if hasNode {
							rspCh <- fmt.Sprintf("Failed to release session %d for UE %s from gNB %s",
								sessionId, ueId, nodeName)
						} else {
							rspCh <- fmt.Sprintf("Failed to release session %d for UE %s",
								sessionId, ueId)
						}
					}
					return nil
				},
			},
		},
	}

	// Build and cache CommandInfo objects
	s.commandCache["emulator"] = s.convertCommandInfos(s.emuCmd.Commands)
	s.commandCache["ue"] = s.convertCommandInfos(s.ueCmd.Commands)
	s.commandCache["gnb"] = s.convertCommandInfos(s.gnbCmd.Commands)
}

// convertCommandInfos converts CLI commands to CommandInfo objects
func (s *CommandStore) convertCommandInfos(commands []*cli.Command) []models.CommandInfo {
	result := make([]models.CommandInfo, 0, len(commands))

	for _, cmd := range commands {
		info := models.CommandInfo{
			Name:        cmd.Name,
			Usage:       cmd.Usage,
			Description: cmd.Description,
			ArgsUsage:   cmd.ArgsUsage,
		}

		// Process flags
		for _, flag := range cmd.Flags {
			flagInfo := models.FlagInfo{
				Name:  strings.Join(flag.Names(), ", "),
				Usage: flagUsage(flag),
			}

			// Set default value text based on flag type
			switch f := flag.(type) {
			case *cli.StringFlag:
				flagInfo.DefaultText = f.Value
			case *cli.BoolFlag:
				if f.Value {
					flagInfo.DefaultText = "true"
				} else {
					flagInfo.DefaultText = "false"
				}
			case *cli.IntFlag:
				if f.Value != 0 {
					flagInfo.DefaultText = fmt.Sprintf("%d", f.Value)
				}
			}

			info.Flags = append(info.Flags, flagInfo)
		}

		result = append(result, info)
	}

	return result
}

// flagUsage gets the usage text from a flag
func flagUsage(flag cli.Flag) string {
	switch f := flag.(type) {
	case *cli.StringFlag:
		return f.Usage
	case *cli.BoolFlag:
		return f.Usage
	case *cli.IntFlag:
		return f.Usage
	default:
		return ""
	}
}

// GetCommandsForNodeType returns command infos for a node type
func (s *CommandStore) GetCommandsForNodeType(nodeType string) []models.CommandInfo {
	if commands, ok := s.commandCache[nodeType]; ok {
		return commands
	}
	return []models.CommandInfo{}
}

// GetObjectsOfType returns objects of a specific type
func (s *CommandStore) GetObjectsOfType(objectType string) ([]string, error) {
	switch objectType {
	case "ue":
		return s.eApi.ListUes(), nil
	case "gnb":
		return s.eApi.ListGnbs(), nil
	case "emulator":
		return []string{"emulator"}, nil
	default:
		return nil, errors.New("invalid object type")
	}
}

// ExecuteCommand executes a command request
func (s *CommandStore) ExecuteCommand(req models.CommandRequest) (models.CommandResponse, error) {
	// Check for help flag
	hasHelpFlag := false
	for _, arg := range req.Args {
		if arg == "--help" || arg == "-h" {
			hasHelpFlag = true
			break
		}
	}

	if hasHelpFlag {
		// Generate help text directly
		helpText := s.GenerateCommandHelp(req.NodeType, req.CommandPath)
		return models.CommandResponse{
			Response: helpText,
		}, nil
	}

	// Create response channel
	rspCh := make(chan string, 1)
	ctx := context.WithValue(context.Background(), "rsp", rspCh)
	ctx = context.WithValue(ctx, "nodename", req.NodeName)

	// Process command args
	var cmdArgs []string
	if req.RawCommand != "" {
		cmdArgs = strings.Fields(req.RawCommand)
	} else {
		cmdArgs = []string{req.CommandPath}
		cmdArgs = append(cmdArgs, req.Args...)
	}

	// Execute appropriate command
	var err error
	switch req.NodeType {
	case "emulator":
		err = s.emuCmd.Run(ctx, append([]string{"emulator"}, cmdArgs...))
	case "ue":
		err = s.ueCmd.Run(ctx, append([]string{"ue"}, cmdArgs...))
	case "gnb":
		err = s.gnbCmd.Run(ctx, append([]string{"gnb"}, cmdArgs...))
	default:
		return models.CommandResponse{}, errors.New("invalid node type")
	}

	if err != nil {
		return models.CommandResponse{}, err
	}

	// Get response from channel
	response := <-rspCh

	return models.CommandResponse{
		Response: response,
	}, nil
}

// GenerateCommandHelp generates help text for a command
func (s *CommandStore) GenerateCommandHelp(nodeType, commandName string) string {
	var cmd *cli.Command

	// Find the command
	switch nodeType {
	case "emulator":
		for _, c := range s.emuCmd.Commands {
			if c.Name == commandName {
				cmd = c
				break
			}
		}
	case "ue":
		for _, c := range s.ueCmd.Commands {
			if c.Name == commandName {
				cmd = c
				break
			}
		}
	case "gnb":
		for _, c := range s.gnbCmd.Commands {
			if c.Name == commandName {
				cmd = c
				break
			}
		}
	}

	if cmd == nil {
		return "No help available for this command"
	}

	var sb strings.Builder

	sb.WriteString(cmd.Name)
	if cmd.ArgsUsage != "" {
		sb.WriteString(" ")
		sb.WriteString(cmd.ArgsUsage)
	} else {
		sb.WriteString(" [command [command options]]")
	}
	sb.WriteString("\n\n")

	if cmd.Description != "" {
		sb.WriteString(cmd.Description)
		sb.WriteString("\n\n")
	}

	if len(cmd.Flags) > 0 {
		sb.WriteString("Options:\n")
		for _, flag := range cmd.Flags {
			names := strings.Join(flag.Names(), ", ")
			usage := flagUsage(flag)

			sb.WriteString(fmt.Sprintf("   --%s:  %s\n", names, usage))
		}
	}

	return sb.String()
}

func GetNodeName(ctx context.Context) (string, bool) {
	val := ctx.Value("nodename")
	if nodename, ok := val.(string); ok {
		return nodename, true
	}
	return "", false
}
