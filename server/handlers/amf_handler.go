package handlers

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/TutuanHo03/remote-control/models"
	"github.com/urfave/cli/v3"
)

type AmfApi interface {
	// Basic AMF functions
	ListUeContexts() []string
	RegisterUe(imsi string) bool
	DeregisterUe(imsi string, cause uint8) bool
	GetServiceStatus() map[string]string
	GetConfiguration() map[string]interface{}

	// N1/N2 Interface Management
	SendN1N2Message(ueId string, messageType string, content string) bool
	ListN1N2Subscriptions(ueId string) []string

	// Handover Management
	InitiateHandover(ueId string, targetGnb string) bool
	ListHandoverHistory(ueId string) []map[string]string

	// SBI Interface
	GetNfSubscriptions() []string
	GetSbiEndpoints() map[string]string
}

// AmfHandler manages AMF command definitions and executions
type NFHandler struct {
	aApi   AmfApi
	amfCmd *cli.Command

	commandCache map[string][]models.CommandInfo
}

// NewNFHandler creates a new AMF handler
func NewNFHandler(aApi AmfApi) *NFHandler {
	handler := &NFHandler{
		aApi:         aApi,
		commandCache: make(map[string][]models.CommandInfo),
	}

	handler.initCommands()

	return handler
}

// initCommands initializes all AMF command definitions
func (h *NFHandler) initCommands() {
	// Initialize AMF commands
	h.amfCmd = &cli.Command{
		Name:        "amf",
		Usage:       "AMF network function management commands",
		Description: "Commands to manage and interact with the AMF (Access and Mobility Function)",
		Commands: []*cli.Command{
			{
				Name:  "list-ues",
				Usage: "List all UE contexts",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					rspCh := ctx.Value("rsp").(chan string)
					ues := h.aApi.ListUeContexts()
					if len(ues) == 0 {
						rspCh <- "No UE contexts found"
					} else {
						rspCh <- fmt.Sprintf("UE contexts:\n%s", strings.Join(ues, "\n"))
					}
					return nil
				},
			},
			{
				Name:        "register-ue",
				Usage:       "Register a UE with IMSI",
				ArgsUsage:   "<imsi>",
				Description: "Register a UE to the network with the specified IMSI",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					rspCh := ctx.Value("rsp").(chan string)
					args := cmd.Args().Slice()
					if len(args) < 1 {
						rspCh <- "Error: IMSI is required"
						return nil
					}
					imsi := args[0]
					success := h.aApi.RegisterUe(imsi)
					if success {
						rspCh <- fmt.Sprintf("UE %s registered successfully", imsi)
					} else {
						rspCh <- fmt.Sprintf("Failed to register UE %s", imsi)
					}
					return nil
				},
			},
			{
				Name:        "deregister-ue",
				Usage:       "Deregister a UE with IMSI",
				ArgsUsage:   "<imsi>",
				Description: "Deregister a UE from the network with the specified IMSI",
				Flags: []cli.Flag{
					&cli.IntFlag{
						Name:  "cause",
						Usage: "Deregistration cause (0-255)",
						Value: 0,
					},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					rspCh := ctx.Value("rsp").(chan string)
					args := cmd.Args().Slice()
					if len(args) < 1 {
						rspCh <- "Error: IMSI is required"
						return nil
					}
					imsi := args[0]
					cause := uint8(cmd.Int("cause"))
					success := h.aApi.DeregisterUe(imsi, cause)
					if success {
						rspCh <- fmt.Sprintf("UE %s deregistered successfully", imsi)
					} else {
						rspCh <- fmt.Sprintf("Failed to deregister UE %s", imsi)
					}
					return nil
				},
			},
			{
				Name:  "status",
				Usage: "Get AMF service status",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					rspCh := ctx.Value("rsp").(chan string)
					status := h.aApi.GetServiceStatus()
					var sb strings.Builder
					sb.WriteString("AMF Service Status:\n")
					for k, v := range status {
						sb.WriteString(fmt.Sprintf("  %s: %s\n", k, v))
					}
					rspCh <- sb.String()
					return nil
				},
			},
			{
				Name:  "config",
				Usage: "Get AMF configuration",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					rspCh := ctx.Value("rsp").(chan string)
					config := h.aApi.GetConfiguration()

					// Format config output
					var sb strings.Builder
					sb.WriteString("AMF Configuration:\n")

					// Handle plmnId specifically as it's nested
					if plmnId, ok := config["plmnId"].(map[string]string); ok {
						sb.WriteString(fmt.Sprintf("  plmnId: MCC %s, MNC %s\n",
							plmnId["mcc"], plmnId["mnc"]))
						delete(config, "plmnId")
					}

					// Add rest of the config items
					for k, v := range config {
						sb.WriteString(fmt.Sprintf("  %s: %v\n", k, v))
					}

					rspCh <- sb.String()
					return nil
				},
			},
			// N1/N2 Interface Management
			{
				Name:        "send-n1n2-message",
				Usage:       "Send N1/N2 message to a UE",
				ArgsUsage:   "<ue-id> <message-type> <content>",
				Description: "Send an N1/N2 message to a specific UE",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					rspCh := ctx.Value("rsp").(chan string)
					args := cmd.Args().Slice()
					if len(args) < 3 {
						rspCh <- "Error: UE ID, message type and content are required"
						return nil
					}
					ueId := args[0]
					msgType := args[1]
					content := args[2]

					success := h.aApi.SendN1N2Message(ueId, msgType, content)
					if success {
						rspCh <- fmt.Sprintf("Message sent successfully to UE %s", ueId)
					} else {
						rspCh <- fmt.Sprintf("Failed to send message to UE %s", ueId)
					}
					return nil
				},
			},
			{
				Name:        "list-n1n2-subscriptions",
				Usage:       "List N1/N2 message subscriptions for a UE",
				ArgsUsage:   "<ue-id>",
				Description: "List all N1/N2 message subscriptions for a specific UE",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					rspCh := ctx.Value("rsp").(chan string)
					args := cmd.Args().Slice()
					if len(args) < 1 {
						rspCh <- "Error: UE ID is required"
						return nil
					}
					ueId := args[0]

					subs := h.aApi.ListN1N2Subscriptions(ueId)
					if len(subs) == 0 {
						rspCh <- fmt.Sprintf("No N1/N2 subscriptions found for UE %s", ueId)
					} else {
						rspCh <- fmt.Sprintf("N1/N2 subscriptions for UE %s:\n%s", ueId, strings.Join(subs, "\n"))
					}
					return nil
				},
			},

			// Handover Management
			{
				Name:        "initiate-handover",
				Usage:       "Initiate handover for a UE",
				ArgsUsage:   "<ue-id> <target-gnb>",
				Description: "Initiate handover procedure for a UE to a target gNB",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					rspCh := ctx.Value("rsp").(chan string)
					args := cmd.Args().Slice()
					if len(args) < 2 {
						rspCh <- "Error: UE ID and target gNB are required"
						return nil
					}
					ueId := args[0]
					targetGnb := args[1]

					success := h.aApi.InitiateHandover(ueId, targetGnb)
					if success {
						rspCh <- fmt.Sprintf("Handover initiated for UE %s to gNB %s", ueId, targetGnb)
					} else {
						rspCh <- fmt.Sprintf("Failed to initiate handover for UE %s", ueId)
					}
					return nil
				},
			},
			{
				Name:        "handover-history",
				Usage:       "Show handover history for a UE",
				ArgsUsage:   "<ue-id>",
				Description: "Display handover history for a specific UE",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					rspCh := ctx.Value("rsp").(chan string)
					args := cmd.Args().Slice()
					if len(args) < 1 {
						rspCh <- "Error: UE ID is required"
						return nil
					}
					ueId := args[0]

					history := h.aApi.ListHandoverHistory(ueId)
					if len(history) == 0 {
						rspCh <- fmt.Sprintf("No handover history found for UE %s", ueId)
					} else {
						var sb strings.Builder
						sb.WriteString(fmt.Sprintf("Handover history for UE %s:\n", ueId))
						for i, entry := range history {
							sb.WriteString(fmt.Sprintf("%d. Time: %s, Source: %s, Target: %s, Status: %s\n",
								i+1, entry["time"], entry["source"], entry["target"], entry["status"]))
						}
						rspCh <- sb.String()
					}
					return nil
				},
			},

			// SBI Interface
			{
				Name:  "nf-subscriptions",
				Usage: "List NF subscriptions",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					rspCh := ctx.Value("rsp").(chan string)
					subs := h.aApi.GetNfSubscriptions()
					if len(subs) == 0 {
						rspCh <- "No NF subscriptions found"
					} else {
						rspCh <- fmt.Sprintf("NF subscriptions:\n%s", strings.Join(subs, "\n"))
					}
					return nil
				},
			},
			{
				Name:  "sbi-endpoints",
				Usage: "List SBI endpoints",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					rspCh := ctx.Value("rsp").(chan string)
					endpoints := h.aApi.GetSbiEndpoints()

					var sb strings.Builder
					sb.WriteString("SBI Endpoints:\n")
					for name, url := range endpoints {
						sb.WriteString(fmt.Sprintf("  %s: %s\n", name, url))
					}
					rspCh <- sb.String()
					return nil
				},
			},
		},
	}

	h.commandCache["amf"] = h.convertCommandInfos(h.amfCmd.Commands)
}

// convertCommandInfos converts CLI commands to CommandInfo objects
func (h *NFHandler) convertCommandInfos(commands []*cli.Command) []models.CommandInfo {
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

// ExecuteCommand executes a command request
func (h *NFHandler) ExecuteCommand(req models.CommandRequest) (models.CommandResponse, error) {
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
		helpText := h.GenerateCommandHelp(req.NodeType, req.CommandPath)
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
	case "amf":
		err = h.amfCmd.Run(ctx, append([]string{"amf"}, cmdArgs...))
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
func (h *NFHandler) GenerateCommandHelp(nodeType, commandName string) string {
	var cmd *cli.Command

	// Find the command
	switch nodeType {
	case "amf":
		for _, c := range h.amfCmd.Commands {
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
