package maintenance

import (
	"github.com/Sirupsen/logrus"
	"github.com/daticahealth/cli/commands/services"
	"github.com/daticahealth/cli/config"
	"github.com/daticahealth/cli/lib/auth"
	"github.com/daticahealth/cli/lib/prompts"
	"github.com/daticahealth/cli/models"
	"github.com/jault3/mow.cli"
)

// Cmd is the contract between the user and the CLI. This specifies the command
// name, arguments, and required/optional arguments and flags for the command.
var Cmd = models.Command{
	Name:      "maintenance",
	ShortHelp: "Manage maintenance mode for code services",
	LongHelp: "Maintenance mode can be enabled or disabled for code services " +
		"on demand. This redirects all traffic to a default maintenance page.",
	CmdFunc: func(settings *models.Settings) func(cmd *cli.Cmd) {
		return func(cmd *cli.Cmd) {
			cmd.CommandLong(DisableSubCmd.Name, DisableSubCmd.ShortHelp, DisableSubCmd.LongHelp, DisableSubCmd.CmdFunc(settings))
			cmd.CommandLong(EnableSubCmd.Name, EnableSubCmd.ShortHelp, EnableSubCmd.LongHelp, EnableSubCmd.CmdFunc(settings))
			cmd.CommandLong(ShowSubCmd.Name, ShowSubCmd.ShortHelp, ShowSubCmd.LongHelp, ShowSubCmd.CmdFunc(settings))
		}
	},
}

var DisableSubCmd = models.Command{
	Name:      "disable",
	ShortHelp: "Disable maintenance mode for a code service",
	LongHelp: "`maintenance disable` turns off maintenance mode for a given code service. " +
		"Here is a sample command\n\n" +
		"```\ndatica -E \"<your_env_alias>\" maintenance disable code-1\n```",
	CmdFunc: func(settings *models.Settings) func(cmd *cli.Cmd) {
		return func(subCmd *cli.Cmd) {
			serviceName := subCmd.StringArg("SERVICE_NAME", "", "The name of the service to disable maintenance mode for")
			subCmd.Action = func() {
				if _, err := auth.New(settings, prompts.New()).Signin(); err != nil {
					logrus.Fatal(err.Error())
				}
				if err := config.CheckRequiredAssociation(settings); err != nil {
					logrus.Fatal(err.Error())
				}
				err := CmdDisable(*serviceName, New(settings), services.New(settings))
				if err != nil {
					logrus.Fatal(err.Error())
				}
			}
			subCmd.Spec = "SERVICE_NAME"
		}
	},
}

var EnableSubCmd = models.Command{
	Name:      "enable",
	ShortHelp: "Enable maintenance mode for a code service",
	LongHelp: "`maintenance enable` turns on maintenance mode for a given code service. " +
		"Maintenance mode redirects all traffic for the given code service to a default HTTP maintenance page. " +
		"If you would like to customize this maintenance page, please contact Datica support. " +
		"Here is a sample command\n\n" +
		"```\ndatica -E \"<your_env_alias>\" maintenance enable code-1\n```",
	CmdFunc: func(settings *models.Settings) func(cmd *cli.Cmd) {
		return func(subCmd *cli.Cmd) {
			serviceName := subCmd.StringArg("SERVICE_NAME", "", "The name of the code service to enable maintenance mode for")
			subCmd.Action = func() {
				if _, err := auth.New(settings, prompts.New()).Signin(); err != nil {
					logrus.Fatal(err.Error())
				}
				if err := config.CheckRequiredAssociation(settings); err != nil {
					logrus.Fatal(err.Error())
				}
				err := CmdEnable(*serviceName, New(settings), services.New(settings))
				if err != nil {
					logrus.Fatal(err.Error())
				}
			}
			subCmd.Spec = "SERVICE_NAME"
		}
	},
}

var ShowSubCmd = models.Command{
	Name:      "show",
	ShortHelp: "Show the status of maintenance mode for a code service",
	LongHelp: "`maintenance show` displays whether or not maintenance mode is enabled " +
		"for a code service or all code services. " +
		"Here are some sample commands\n\n" +
		"```\ndatica -E \"<your_env_alias>\" maintenance show\n" +
		"datica -E \"<your_env_alias>\" maintenance show code-1\n```",
	CmdFunc: func(settings *models.Settings) func(cmd *cli.Cmd) {
		return func(subCmd *cli.Cmd) {
			serviceName := subCmd.StringArg("SERVICE_NAME", "", "The name of the code service to show the status of maintenance mode")
			subCmd.Action = func() {
				if _, err := auth.New(settings, prompts.New()).Signin(); err != nil {
					logrus.Fatal(err.Error())
				}
				if err := config.CheckRequiredAssociation(settings); err != nil {
					logrus.Fatal(err.Error())
				}
				err := CmdShow(*serviceName, settings.EnvironmentID, settings.Pod, New(settings), services.New(settings))
				if err != nil {
					logrus.Fatal(err.Error())
				}
			}
			subCmd.Spec = "[SERVICE_NAME]"
		}
	},
}

// IMaintenance
type IMaintenance interface {
	Enable(svcProxyID, upstreamID string) error
	Disable(svcProxyID, upstreamID string) error
	List(svcProxyID string) (*[]models.Maintenance, error)
}

// SMaintenance is a concrete implementation of IMaintenance
type SMaintenance struct {
	Settings *models.Settings
}

// New returns an instance of IMetrics
func New(settings *models.Settings) IMaintenance {
	return &SMaintenance{
		Settings: settings,
	}
}
