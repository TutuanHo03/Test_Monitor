package main

import (
	"fmt"
	"github.com/abiosoft/ishell"
	//	"strings"
	"test_monitor/emulator"
	"test_monitor/models"
)

var contextCmds []*ishell.Cmd
var connectCmd, disconnectCmd *ishell.Cmd

func main() {
	//run server
	emulator.InitCmds()

	shell := ishell.New()
	shell.Println("Interactive shell")

	shell.AddCmd(&ishell.Cmd{
		Name: "greet",
		Help: "greet someone",
		Func: func(c *ishell.Context) {
			if len(c.Args) > 0 {
				c.Println("Hello", c.Args[0])
			} else {
				c.Println("Hello there")
			}
		},
	})
	disconnectCmd = &ishell.Cmd{
		Name: "disconnect",
		Help: "Disconnect server",
		Func: func(c *ishell.Context) {
			for _, c := range contextCmds {
				shell.DeleteCmd(c.Name)
			}
			contextCmds = []*ishell.Cmd{}
			shell.DeleteCmd("disconnect")
			shell.AddCmd(connectCmd)
		},
	}

	connectCmd = &ishell.Cmd{
		Name: "connect",
		Help: "Connect to server",
		Func: func(c *ishell.Context) {
			if len(c.Args) == 0 {
				c.Printf("Missing server address\n")
				return
			}
			server := c.Args[0]
			var err error
			if contextCmds, err = getCommands(server); err != nil {
				c.Printf("Fail to connect to server: %+v\n", err)
				return
			}
			c.Printf("Connected to server: %s, type help to see commands\n", server)
			for _, c := range contextCmds {
				shell.AddCmd(c)
			}
			shell.AddCmd(disconnectCmd)
			shell.DeleteCmd("connect")
		},
	}
	shell.AddCmd(connectCmd)

	shell.Run()
}

//retrive commands from server
func requestCommands(server string) (cmds []models.Command, err error) {
	//TODO: connect to server to get command list
	cmds = emulator.GetCommands()
	return
}

func getCommands(server string) (cmds []*ishell.Cmd, err error) {
	var jsonCmds []models.Command
	if jsonCmds, err = requestCommands(server); err != nil {
		err = fmt.Errorf("Fail to get command list: %+v", err)
		return
	}
	for _, cmd := range jsonCmds {
		cmds = append(cmds, &ishell.Cmd{
			Name: cmd.Name,
			Help: cmd.Help,
			Func: execCmd,
		})
	}
	return
}

//send command to server
func execCmd(c *ishell.Context) {
	args := append(c.Args, c.Cmd.Name)
	if rsp, err := sendCmd(args); err != nil {
		c.Printf("Fail to send command to server: %+v\n", err)
	} else {
		c.Printf("Server response: %s\n", rsp)
	}
}

func sendCmd(args []string) (rsp string, err error) {
	//rsp = fmt.Sprintf("command \"%s\" is received", strings.Join(args, " "))
	rsp, err = emulator.RunCmd(args)
	return
}
