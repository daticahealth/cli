package supportids

import (
	"github.com/Sirupsen/logrus"
	"github.com/daticahealth/cli/models"
	"github.com/jault3/mow.cli"
)

// Cmd is the contract between the user and the CLI. This specifies the command
// name, arguments, and required/optional arguments and flags for the command.
var Cmd = models.Command{
	Name:      "support-ids",
	ShortHelp: "Print out various IDs related to your associated environment to be used when contacting Datica support",
	LongHelp: "`support-ids` is helpful when contacting Datica support by submitting a ticket at https://datica.com/support. " +
		"If you are having an issue with a CLI command or anything with your environment, it is helpful to run this command and copy the output into the initial correspondence with a Datica engineer. " +
		"This will help Datica identify the environment faster and help come to resolution faster. Here is a sample command\n\n" +
		"```\ndatica -E \"<your_env_alias>\" support-ids\n```",
	CmdFunc: func(settings *models.Settings) func(cmd *cli.Cmd) {
		return func(cmd *cli.Cmd) {
			cmd.Action = func() {
				err := CmdSupportIDs(New(settings))
				if err != nil {
					logrus.Fatal(err.Error())
				}
			}
		}
	},
}

// ISupportIDs
type ISupportIDs interface {
	SupportIDs() (string, string, string, string, error)
}

// SSupportIDs is a concrete implementation of ISupportIDs
type SSupportIDs struct {
	Settings *models.Settings
}

// New returns an instance of ISupportIDs
func New(settings *models.Settings) ISupportIDs {
	return &SSupportIDs{
		Settings: settings,
	}
}
