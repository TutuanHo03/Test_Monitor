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

	DeregisterUe(imsi string, cause uint8) bool
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
					imsi := ""
					cause := uint8(cmd.Int("cause"))

					// Get arguments
					args := cmd.Args().Slice()
					if len(args) > 0 {
						imsi = args[0]
					}

					// Check for custom args
					if customArgs, ok := ctx.Value("custom_args").(map[string]string); ok {
						// Check for positional arg for IMSI
						if val, exists := customArgs["arg1"]; exists {
							imsi = val
						}

						// Check for cause flag
						if val, exists := customArgs["cause"]; exists {
							var causeVal uint8
							n, err := fmt.Sscanf(val, "%d", &causeVal)
							if err == nil && n == 1 {
								cause = causeVal
							}
						}
					}

					if imsi == "" {
						rspCh <- "Error: IMSI is required"
						return nil
					}

					fmt.Printf("Deregistering UE with IMSI: %s and cause: %d\n", imsi, cause)
					success := h.aApi.DeregisterUe(imsi, cause)
					if success {
						rspCh <- fmt.Sprintf("UE %s deregistered successfully", imsi)
					} else {
						rspCh <- fmt.Sprintf("Failed to deregister UE %s", imsi)
					}
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

	var err error

	// Process command
	if req.RawCommand != "" {
		// Process raw command string with custom parser
		cmdName, customArgs := parseCommandArgs(req.RawCommand)

		if len(customArgs) > 0 {
			ctx = context.WithValue(ctx, "custom_args", customArgs)
		}

		fmt.Printf("DEBUG: Parsed AMF raw command '%s' into command '%s' with args: %v\n",
			req.RawCommand, cmdName, customArgs)

		switch req.NodeType {
		case "amf":
			// Set up standard CLI args
			cliArgs := []string{"amf", cmdName}
			err = h.amfCmd.Run(ctx, cliArgs)
		default:
			return models.CommandResponse{}, errors.New("invalid node type")
		}
	} else {
		// Standard command processing
		cmdArgs := []string{req.CommandPath}
		cmdArgs = append(cmdArgs, req.Args...)

		switch req.NodeType {
		case "amf":
			err = h.amfCmd.Run(ctx, append([]string{"amf"}, cmdArgs...))
		default:
			return models.CommandResponse{}, errors.New("invalid node type")
		}
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
