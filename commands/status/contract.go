package status

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
	Name:      "status",
	ShortHelp: "Get quick readout of the current status of your associated environment and all of its services",
	LongHelp: "`status` will give a quick readout of your environment's health. " +
		"This includes your environment name, environment ID, and for each service the name, size, build status, deploy status, and service ID. " +
		"Here is a sample command\n\n" +
		"```\ndatica -E \"<your_env_alias>\" status\ndatica -E \"<your_env_alias>\" status --historical\n```",
	CmdFunc: func(settings *models.Settings) func(cmd *cli.Cmd) {
		return func(cmd *cli.Cmd) {
			historical := cmd.BoolOpt("historical", false, "If this option is specified, a complete history of jobs will be reported")
			cmd.Action = func() {
				if _, err := auth.New(settings, prompts.New()).Signin(); err != nil {
					logrus.Fatal(err.Error())
				}
				if err := config.CheckRequiredAssociation(settings); err != nil {
					logrus.Fatal(err.Error())
				}
				err := CmdStatus(settings.EnvironmentID, New(settings, jobs.New(settings)), environments.New(settings), services.New(settings), *historical)
				if err != nil {
					logrus.Fatal(err.Error())
				}
			}
			cmd.Spec = "[--historical]"
		}
	},
}

// IStatus
type IStatus interface {
	Status(env *models.Environment, services *[]models.Service, historical bool) error
}

// SStatus is a concrete implementation of IStatus
type SStatus struct {
	Settings *models.Settings
	Jobs     jobs.IJobs
}

// New returns an instance of IStatus
func New(settings *models.Settings, ij jobs.IJobs) IStatus {
	return &SStatus{
		Settings: settings,
		Jobs:     ij,
	}
}
