package common

import (
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

var commands []*cli.Command

// Commander --
type Commander interface {
	Execute(c *cli.Context)
}

// RegisterCommand --
func RegisterCommand(command *cli.Command) {
	logrus.Debugln("Registering", command.Name, "command...")
	commands = append(commands, command)
}

// GetCommands --
func GetCommands() []*cli.Command {
	return commands
}

var subcommands = make(map[string][]*cli.Command, 0)

func RegisterSubcommand(group string, command *cli.Command) {
	logrus.Debugln("Registering", command.Name, "command...")
	subcommands[group] = append(subcommands[group], command)
}

func GetSubcommands(group string) []*cli.Command {
	return subcommands[group]
}

func GetCommand(name string) *cli.Command {
	for _, c := range commands {
		if c.Name == name {
			return c
		}
	}

	return nil
}
