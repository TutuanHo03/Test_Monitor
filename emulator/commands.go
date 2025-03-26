package emulator

import (
	"context"
	"github.com/urfave/cli/v3"
	"strings"
	"test_monitor/emulator/api"
	"test_monitor/models"
)

type EmulatorApi interface {
	ListUes() []string
	ListGnbs() []string
	//	AddUe(supi string, triggerRegister bool) bool
}

type UeApi interface {
	Register(isEmergency bool)
	Deregister(deregisterType uint8) bool
	CreateSession(slice string, dnName string, sessionType uint8) bool
}

type GnbApi interface {
	ReleaseUe(ueId string) bool
	ReleaseSession(ueId string, sessionId uint8) bool
}

func printHelper(cmd *cli.Command) string {
	//print a helper string for a given command
	return cmd.Usage
}

func GetCommands() (cmds []models.Command) {
	for _, cmd := range _cmd.Commands {
		cmds = append(cmds, models.Command{
			Name: cmd.Name,
			Help: printHelper(cmd),
		})
	}
	return
}

var _cmd *cli.Command

//should be called when start server
func InitCmds() {
	eApi := api.CreateEmulatorApi()
	_cmd = &cli.Command{
		Name: "emulator",
		Commands: []*cli.Command{
			&cli.Command{
				Name:  "list-gnb",
				Usage: "List all GnBs",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					gnbs := eApi.ListGnbs()
					rsp := strings.Join(gnbs, " ")
					rspCh := ctx.Value("rsp").(chan string)
					rspCh <- rsp
					return nil
				},
			},
			&cli.Command{
				Name:  "list-ue",
				Usage: "List all UEs",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					ues := eApi.ListUes()
					rsp := strings.Join(ues, " ")
					rspCh := ctx.Value("rsp").(chan string)
					rspCh <- rsp
					return nil
				},
			},
		},
	}
}

func RunCmd(args []string) (rsp string, err error) {
	args = append([]string{"emulator"}, args...)
	rspCh := make(chan string, 1)
	ctx := context.WithValue(context.Background(), "rsp", rspCh)
	if err = _cmd.Run(ctx, args); err != nil {
		return
	}
	rsp = <-rspCh
	return
}
