package users

import (
	"github.com/Sirupsen/logrus"
	"github.com/daticahealth/cli/commands/invites"
	"github.com/daticahealth/cli/config"
	"github.com/daticahealth/cli/lib/auth"
	"github.com/daticahealth/cli/lib/prompts"
	"github.com/daticahealth/cli/models"
	"github.com/jault3/mow.cli"
)

// Cmd is the contract between the user and the CLI. This specifies the command
// name, arguments, and required/optional arguments and flags for the command.
var Cmd = models.Command{
	Name:      "users",
	ShortHelp: "Manage users who have access to the given organization",
	LongHelp: "The `users` command allows you to manage who has access to your environment through the organization that owns the environment. " +
		"The users command can not be run directly but has three sub commands.",
	CmdFunc: func(settings *models.Settings) func(cmd *cli.Cmd) {
		return func(cmd *cli.Cmd) {
			cmd.CommandLong(ListSubCmd.Name, ListSubCmd.ShortHelp, ListSubCmd.LongHelp, ListSubCmd.CmdFunc(settings))
			cmd.CommandLong(RmSubCmd.Name, RmSubCmd.ShortHelp, RmSubCmd.LongHelp, RmSubCmd.CmdFunc(settings))
		}
	},
}

var ListSubCmd = models.Command{
	Name:      "list",
	ShortHelp: "List all users who have access to the given organization",
	LongHelp: "`users list` shows every user that belongs to your environment's organization. " +
		"Users who belong to your environment's organization may access to your environment's services and data depending on their role in the organization. " +
		"Here is a sample command\n\n" +
		"```\ndatica -E \"<your_env_alias>\" users list\n```",
	CmdFunc: func(settings *models.Settings) func(cmd *cli.Cmd) {
		return func(subCmd *cli.Cmd) {
			subCmd.Action = func() {
				if _, err := auth.New(settings, prompts.New()).Signin(); err != nil {
					logrus.Fatal(err.Error())
				}
				if err := config.CheckRequiredAssociation(settings); err != nil {
					logrus.Fatal(err.Error())
				}
				err := CmdList(settings.UsersID, New(settings), invites.New(settings))
				if err != nil {
					logrus.Fatal(err.Error())
				}
			}
		}
	},
}

var RmSubCmd = models.Command{
	Name:      "rm",
	ShortHelp: "Revoke access to the given organization for the given user",
	LongHelp: "`users rm` revokes a users access to your environment's organization. " +
		"Revoking a user's access to your environment's organization will revoke their access to your organization's environments. " +
		"Here is a sample command\n\n" +
		"```\ndatica -E \"<your_env_alias>\" users rm user@example.com\n```",
	CmdFunc: func(settings *models.Settings) func(cmd *cli.Cmd) {
		return func(subCmd *cli.Cmd) {
			email := subCmd.StringArg("EMAIL", "", "The email address of the user to revoke access from for the given organization")
			subCmd.Action = func() {
				if _, err := auth.New(settings, prompts.New()).Signin(); err != nil {
					logrus.Fatal(err.Error())
				}
				if err := config.CheckRequiredAssociation(settings); err != nil {
					logrus.Fatal(err.Error())
				}
				err := CmdRm(*email, New(settings))
				if err != nil {
					logrus.Fatal(err.Error())
				}
			}
			subCmd.Spec = "EMAIL"
		}
	},
}

// IUsers
type IUsers interface {
	List() (*[]models.OrgUser, error)
	Rm(usersID string) error
}

// SUsers is a concrete implementation of IUsers
type SUsers struct {
	Settings *models.Settings
}

// New generates a new instance of IUsers
func New(settings *models.Settings) IUsers {
	return &SUsers{
		Settings: settings,
	}
}
