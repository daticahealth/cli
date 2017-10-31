package redeploy

import (
	"github.com/Sirupsen/logrus"
	"github.com/daticahealth/cli/commands/environments"
	"github.com/daticahealth/cli/commands/services"
	"github.com/daticahealth/cli/config"
	"github.com/daticahealth/cli/lib/auth"
	"github.com/daticahealth/cli/lib/jobs"
	"github.com/daticahealth/cli/lib/prompts"
	"github.com/daticahealth/cli/models"
	"github.com/jault3/mow.cli"
)

// Cmd is the contract between the user and the CLI. This specifies the command
// name, arguments, and required/optional arguments and flags for the command.
var Cmd = models.Command{
	Name:      "redeploy",
	ShortHelp: "Redeploy a service without having to do a git push. This will cause downtime for all redeploys (see the resources page for more details).",
	LongHelp: "`redeploy` deploys an identical copy of the given service. " +
		"For code services, this avoids having to perform a code push. You skip the git push and the build. " +
		"For service proxies, new instances replace the old ones. " +
		"All other service types cannot be redeployed with this command. " +
		"For service proxy redeploys, there will be approximately 5 minutes of downtime. " +
		"For code service redeploys, there will be approximately 30 seconds of downtime. " +
		"Here is a sample command\n\n" +
		"```\ndatica -E \"<your_env_name>\" redeploy app01\n```",
	CmdFunc: func(settings *models.Settings) func(cmd *cli.Cmd) {
		return func(cmd *cli.Cmd) {
			serviceName := cmd.StringArg("SERVICE_NAME", "", "The name of the service to redeploy (e.g. 'app01')")
			cmd.Action = func() {
				if _, err := auth.New(settings, prompts.New()).Signin(); err != nil {
					logrus.Fatal(err.Error())
				}
				if err := config.CheckRequiredAssociation(settings); err != nil {
					logrus.Fatal(err.Error())
				}
				err := CmdRedeploy(settings.EnvironmentID, *serviceName, jobs.New(settings), services.New(settings), environments.New(settings))
				if err != nil {
					logrus.Fatal(err.Error())
				}
			}
			cmd.Spec = "SERVICE_NAME"
		}
	},
}
