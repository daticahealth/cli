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
	LongHelp: "The <code>init</code> command walks you through setting up the CLI to use with the Datica platform. " +
		"The <code>init</code> command requires access to an environment with a code service. ",
	CmdFunc: func(settings *models.Settings) func(cmd *cli.Cmd) {
		return func(cmd *cli.Cmd) {
			service := cmd.StringArg("SERVICE", "", "Service name to use. If not provided, and the environment has more than one, you will be asked for a choice.")
			remoteName := cmd.StringOpt("remote-name", "datica", "Name to use for the git remote.")
			noInput := cmd.BoolOpt("no-input", false, "If set, this command does not prompt for input. Environment and service must be explicitly provided.")
			overwriteRemote := cmd.BoolOpt("overwrite-remote", false, "If set, and the git remote named by --remote-name already exists for this repository, overwrite it without prompting.")
			cmd.Action = func() {
				p := prompts.New()
				if _, err := auth.New(settings, p).Signin(); err != nil {
					logrus.Fatal(err.Error())
				}
				err := CmdInit(*service, *noInput, *remoteName, *overwriteRemote, settings, p)
				if err != nil {
					logrus.Fatal(err.Error())
				}
			}
			cmd.Spec = "[SERVICE] [--remote-name] [--overwrite-remote] [--no-input]"
		}
	},
}
