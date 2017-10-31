package jobs

import (
	"github.com/Sirupsen/logrus"
	"github.com/daticahealth/cli/commands/services"
	"github.com/daticahealth/cli/config"
	"github.com/daticahealth/cli/lib/auth"
	"github.com/daticahealth/cli/lib/prompts"
	"github.com/daticahealth/cli/models"
	"github.com/jault3/mow.cli"
)

var Cmd = models.Command{
	Name:      "jobs",
	ShortHelp: "Perform operations on a service's jobs",
	LongHelp:  "The `jobs` command allows you to manage jobs for your service(s).  The jobs command cannot be run directly but has sub commands.",
	CmdFunc: func(settings *models.Settings) func(cmd *cli.Cmd) {
		return func(cmd *cli.Cmd) {
			cmd.CommandLong(ListSubCmd.Name, ListSubCmd.ShortHelp, ListSubCmd.LongHelp, ListSubCmd.CmdFunc(settings))
			cmd.CommandLong(StartSubCmd.Name, StartSubCmd.ShortHelp, StartSubCmd.LongHelp, StartSubCmd.CmdFunc(settings))
			cmd.CommandLong(StopSubCmd.Name, StopSubCmd.ShortHelp, StopSubCmd.LongHelp, StopSubCmd.CmdFunc(settings))
		}
	},
}

var ListSubCmd = models.Command{
	Name:      "list",
	ShortHelp: "List all jobs for a service",
	LongHelp: "`jobs list` prints out a list of all jobs in your environment and their current status." +
		"Here is a sample command\n\n" +
		"```\ndatica -E \"<your_env_name>\" jobs list <your_service_name>\n```",
	CmdFunc: func(settings *models.Settings) func(cmd *cli.Cmd) {
		return func(subCmd *cli.Cmd) {
			serviceName := subCmd.StringArg("SERVICE_NAME", "", "The name of the service to list jobs for")
			subCmd.Action = func() {
				if _, err := auth.New(settings, prompts.New()).Signin(); err != nil {
					logrus.Fatal(err.Error())
				}
				if err := config.CheckRequiredAssociation(settings); err != nil {
					logrus.Fatal(err.Error())
				}
				err := CmdList(*serviceName, New(settings), services.New(settings))
				if err != nil {
					logrus.Fatal(err.Error())
				}
			}
		}
	},
}

var StartSubCmd = models.Command{
	Name:      "start",
	ShortHelp: "Start a specific job within a service",
	LongHelp: "`jobs start` will start a job that is configured but not currently running within a given service" +
		"This command is useful for granual control of your services and their workers, tasks, etc." +
		"```\ndatica -E \"<your_env_name>\" jobs start <your_service_name> <your_job_id>\n```",
	CmdFunc: func(settings *models.Settings) func(cmd *cli.Cmd) {
		return func(subCmd *cli.Cmd) {
			serviceName := subCmd.StringArg("SERVICE_NAME", "", "The name of the service to list jobs for")
			jobID := subCmd.StringArg("JOB_ID", "", "The job ID for the job in service to be started")
			subCmd.Action = func() {
				if _, err := auth.New(settings, prompts.New()).Signin(); err != nil {
					logrus.Fatal(err.Error())
				}
				if err := config.CheckRequiredAssociation(settings); err != nil {
					logrus.Fatal(err.Error())
				}
				err := CmdStart(*jobID, *serviceName, New(settings), services.New(settings))
				if err != nil {
					logrus.Fatal(err.Error())
				}
			}
		}
	},
}

var StopSubCmd = models.Command{
	Name:      "stop",
	ShortHelp: "Stop a specific job within a service",
	LongHelp: "`jobs stop` will shut down a running job within a given service" +
		"This command is useful for granual control of your services and their workers, tasks, etc." +
		"```\ndatica -E \"<your_env_name>\" jobs stop <your_service_name> <your_job_id>\n```",
	CmdFunc: func(settings *models.Settings) func(cmd *cli.Cmd) {
		return func(subCmd *cli.Cmd) {
			serviceName := subCmd.StringArg("SERVICE_NAME", "", "The name of the service to list jobs for")
			jobID := subCmd.StringArg("JOB_ID", "", "The job ID for the job in service to be stopped")
			force := subCmd.BoolOpt("f force", false, "Allow this command to be executed without prompting to confirm")

			subCmd.Action = func() {
				if _, err := auth.New(settings, prompts.New()).Signin(); err != nil {
					logrus.Fatal(err.Error())
				}
				if err := config.CheckRequiredAssociation(settings); err != nil {
					logrus.Fatal(err.Error())
				}
				err := CmdStop(*jobID, *serviceName, New(settings), services.New(settings), *force, prompts.New())
				if err != nil {
					logrus.Fatal(err.Error())
				}
			}
			subCmd.Spec = "[SERVICE_NAME] [JOB_ID] [-f]"

		}
	},
}

// IJobs
type IJobs interface {
	List(svcID string) (*[]models.Job, error)
	Start(jobID string, svcID string) error
	Stop(jobID string, svcID string) error
}

// SServices is a concrete implementation of IJobs
type SJobs struct {
	Settings *models.Settings
}

// New generates a new instance of IJobs
func New(settings *models.Settings) IJobs {
	return &SJobs{
		Settings: settings,
	}
}
