package update

import (
	"github.com/Sirupsen/logrus"
	"github.com/daticahealth/cli/models"
	"github.com/jault3/mow.cli"
)

// Cmd is the contract between the user and the CLI. This specifies the command
// name, arguments, and required/optional arguments and flags for the command.
var Cmd = models.Command{
	Name:      "update",
	ShortHelp: "Checks for available updates and updates the CLI if a new update is available",
	LongHelp: "`update` is a shortcut to update your CLI instantly. " +
		"If a newer version of the CLI is available, it will be downloaded and installed automatically. " +
		"This is used when you want to apply an update before the CLI automatically applies it on its own. " +
		"Here is a sample command\n\n" +
		"```\ndatica update\n```",
	CmdFunc: func(settings *models.Settings) func(cmd *cli.Cmd) {
		return func(cmd *cli.Cmd) {
			cmd.Action = func() {
				err := CmdUpdate(New(settings))
				if err != nil {
					logrus.Fatal(err.Error())
				}
			}
		}
	},
}

// IUpdate
type IUpdate interface {
	Check() (bool, error)
	Update() error
	UpdateEnvironments()
	UpdatePods()
}

// SUpdate is a concrete implementation of IUpdate
type SUpdate struct {
	Settings *models.Settings
}

// New returns an instance of IUpdate
func New(settings *models.Settings) IUpdate {
	return &SUpdate{
		Settings: settings,
	}
}
