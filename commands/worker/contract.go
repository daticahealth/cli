package worker

import (
	"github.com/Sirupsen/logrus"
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
	Name:      "worker",
	ShortHelp: "Manage a service's workers",
	LongHelp:  "The `worker` command allows to deploy, list, remove, and scale the workers in a code service.",
	CmdFunc: func(settings *models.Settings) func(cmd *cli.Cmd) {
		return func(cmd *cli.Cmd) {
			cmd.CommandLong(DeploySubCmd.Name, DeploySubCmd.ShortHelp, DeploySubCmd.LongHelp, DeploySubCmd.CmdFunc(settings))
			cmd.CommandLong(ListSubCmd.Name, ListSubCmd.ShortHelp, ListSubCmd.LongHelp, ListSubCmd.CmdFunc(settings))
			cmd.CommandLong(RmSubCmd.Name, RmSubCmd.ShortHelp, RmSubCmd.LongHelp, RmSubCmd.CmdFunc(settings))
			cmd.CommandLong(ScaleSubCmd.Name, ScaleSubCmd.ShortHelp, ScaleSubCmd.LongHelp, ScaleSubCmd.CmdFunc(settings))
		}
	},
}

var DeploySubCmd = models.Command{
	Name:      "deploy",
	ShortHelp: "Deploy new workers for a given service",
	LongHelp: "`worker deploy` allows you to start a background process asynchronously. The TARGET must be specified in your Procfile. " +
		"Once the worker is started, any output can be found in your logging Dashboard or using the [logs](#logs) command. " +
		"Here is a sample command\n\n" +
		"```\ndatica -E \"<your_env_name>\" worker deploy code-1 mailer\n```",
	CmdFunc: func(settings *models.Settings) func(cmd *cli.Cmd) {
		return func(subCmd *cli.Cmd) {
			serviceName := subCmd.StringArg("SERVICE_NAME", "", "The name of the service to use to deploy a worker")
			target := subCmd.StringArg("TARGET", "", "The name of the Procfile target to invoke as a worker")
			subCmd.Action = func() {
				if _, err := auth.New(settings, prompts.New()).Signin(); err != nil {
					logrus.Fatal(err.Error())
				}
				if err := config.CheckRequiredAssociation(settings); err != nil {
					logrus.Fatal(err.Error())
				}
				err := CmdDeploy(*serviceName, *target, New(settings), services.New(settings), jobs.New(settings))
				if err != nil {
					logrus.Fatal(err.Error())
				}
			}
			subCmd.Spec = "SERVICE_NAME TARGET"
		}
	},
}

var ListSubCmd = models.Command{
	Name:      "list",
	ShortHelp: "Lists all workers for a given service",
	LongHelp: "`worker list` lists all workers and their scale for a given code service along with the number of currently running instances of each worker target. Here is a sample command\n\n" +
		"```\ndatica -E \"<your_env_name>\" worker list code-1\n```",
	CmdFunc: func(settings *models.Settings) func(cmd *cli.Cmd) {
		return func(subCmd *cli.Cmd) {
			serviceName := subCmd.StringArg("SERVICE_NAME", "", "The name of the service to list workers for")
			subCmd.Action = func() {
				if _, err := auth.New(settings, prompts.New()).Signin(); err != nil {
					logrus.Fatal(err.Error())
				}
				if err := config.CheckRequiredAssociation(settings); err != nil {
					logrus.Fatal(err.Error())
				}
				err := CmdList(*serviceName, New(settings), services.New(settings), jobs.New(settings))
				if err != nil {
					logrus.Fatal(err.Error())
				}
			}
			subCmd.Spec = "SERVICE_NAME"
		}
	},
}

var RmSubCmd = models.Command{
	Name:      "rm",
	ShortHelp: "Remove all workers for a given service and target",
	LongHelp: "`worker rm` removes a worker by the given TARGET and stops all currently running instances of that TARGET. Here is a sample command\n\n" +
		"```\ndatica -E \"<your_env_name>\" worker rm code-1 mailer\n```",
	CmdFunc: func(settings *models.Settings) func(cmd *cli.Cmd) {
		return func(subCmd *cli.Cmd) {
			serviceName := subCmd.StringArg("SERVICE_NAME", "", "The name of the service running the workers")
			target := subCmd.StringArg("TARGET", "", "The worker target to remove")
			subCmd.Action = func() {
				if _, err := auth.New(settings, prompts.New()).Signin(); err != nil {
					logrus.Fatal(err.Error())
				}
				if err := config.CheckRequiredAssociation(settings); err != nil {
					logrus.Fatal(err.Error())
				}
				err := CmdRm(*serviceName, *target, New(settings), services.New(settings), prompts.New(), jobs.New(settings))
				if err != nil {
					logrus.Fatalln(err.Error())
				}
			}
			subCmd.Spec = "SERVICE_NAME TARGET"
		}
	},
}

var ScaleSubCmd = models.Command{
	Name:      "scale",
	ShortHelp: "Scale existing workers up or down for a given service and target",
	LongHelp: "`worker scale` allows you to scale up or down a given worker TARGET. " +
		"Scaling up will launch new instances of the worker TARGET while scaling down will immediately stop running instances of the worker TARGET if applicable. Here are some sample commands\n\n" +
		"```\ndatica -E \"<your_env_name>\" worker scale code-1 mailer 1\n" +
		"datica -E \"<your_env_name>\" worker scale code-1 mailer -- -2\n```",
	CmdFunc: func(settings *models.Settings) func(cmd *cli.Cmd) {
		return func(subCmd *cli.Cmd) {
			serviceName := subCmd.StringArg("SERVICE_NAME", "", "The name of the service running the workers")
			target := subCmd.StringArg("TARGET", "", "The worker target to scale up or down")
			scale := subCmd.StringArg("SCALE", "", "The new scale (or change in scale) for the given worker target. This can be a single value (e.g. 2) representing the final number of workers that should be running. Or this can be a change represented by a plus or minus sign followed by the value (e.g. +2 or -1). When using a change in value, be sure to insert the \"--\" operator to signal the end of options. For example, \"datica worker scale code-1 worker -- -1\"")
			subCmd.Action = func() {
				if _, err := auth.New(settings, prompts.New()).Signin(); err != nil {
					logrus.Fatal(err.Error())
				}
				if err := config.CheckRequiredAssociation(settings); err != nil {
					logrus.Fatal(err.Error())
				}
				err := CmdScale(*serviceName, *target, *scale, New(settings), services.New(settings), prompts.New(), jobs.New(settings))
				if err != nil {
					logrus.Fatal(err.Error())
				}
			}
			subCmd.Spec = "SERVICE_NAME TARGET SCALE"
		}
	},
}

// IWorker
type IWorker interface {
	ParseScale(scaleString string) (func(scale, change int) int, int, error)
	Retrieve(svcID string) (*models.Workers, error)
	Update(svcID string, workers *models.Workers) error
}

// SWorker is a concrete implementation of IWorker
type SWorker struct {
	Settings *models.Settings
}

// New returns an instance of IWorker
func New(settings *models.Settings) IWorker {
	return &SWorker{
		Settings: settings,
	}
}
