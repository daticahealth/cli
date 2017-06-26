package init

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
	Name:      "init",
	ShortHelp: "Get started using the Datica platform",
	LongHelp: "The `init` command walks you through setting up the CLI to use with the Datica platform. " +
		"The `init` command requires you to have an environment already setup. ",
	CmdFunc: func(settings *models.Settings) func(cmd *cli.Cmd) {
		return func(cmd *cli.Cmd) {
			cmd.Action = func() {
				p := prompts.New()
				if _, err := auth.New(settings, p).Signin(); err != nil {
					logrus.Fatal(err.Error())
				}
				err := CmdInit(settings, p)
				if err != nil {
					logrus.Fatal(err.Error())
				}
			}
		}
	},
}
