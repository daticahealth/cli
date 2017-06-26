package logout

import (
	"github.com/Sirupsen/logrus"
	"github.com/daticahealth/cli/lib/auth"
	"github.com/daticahealth/cli/lib/prompts"
	"github.com/daticahealth/cli/models"
	"github.com/jault3/mow.cli"
)

// Cmd is the contract between the user and the CLI. This specifies the command
// name, arguments, and required/optional arguments and flags for the command.
var Cmd = models.Command{
	Name:      "logout",
	ShortHelp: "Clear the stored user information from your local machine",
	LongHelp: "When using the CLI, your email and password are **never** stored in any file on your filesystem. " +
		"However, in order to not type in your email and password each and every command, a session token is stored in the CLI's configuration file and used until it expires. " +
		"`logout` removes this session token from the configuration file. Here is a sample command\n\n" +
		"```\ndatica logout\n```",
	CmdFunc: func(settings *models.Settings) func(cmd *cli.Cmd) {
		return func(cmd *cli.Cmd) {
			cmd.Action = func() {
				err := CmdLogout(New(settings), auth.New(settings, prompts.New()))
				if err != nil {
					logrus.Fatal(err.Error())
				}
			}
		}
	},
}

// ILogout
type ILogout interface {
	Clear() error
}

// SLogout is a concrete implementation of ILogout
type SLogout struct {
	Settings *models.Settings
}

// New returns an instance of ILogout
func New(settings *models.Settings) ILogout {
	return &SLogout{
		Settings: settings,
	}
}
