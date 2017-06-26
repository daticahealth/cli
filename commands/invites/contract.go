package invites

import (
	"github.com/Sirupsen/logrus"
	"github.com/daticahealth/cli/config"
	"github.com/daticahealth/cli/lib/auth"
	"github.com/daticahealth/cli/lib/prompts"
	"github.com/daticahealth/cli/models"
	"github.com/jault3/mow.cli"
)

// Cmd is the contract between the user and the CLI. This specifies the command
// name, arguments, and required/optional arguments and flags for the command.
var Cmd = models.Command{
	Name:      "invites",
	ShortHelp: "Manage invitations for your organizations",
	LongHelp: "The `invites` command gives access to organization invitations. " +
		"Every environment is owned by an organization and users join organizations in order to access individual environments. " +
		"You can invite new users by email and manage pending invites through the CLI. " +
		"You cannot call the `invites` command directly, but must call one of its subcommands.",
	CmdFunc: func(settings *models.Settings) func(cmd *cli.Cmd) {
		return func(cmd *cli.Cmd) {
			cmd.CommandLong(AcceptSubCmd.Name, AcceptSubCmd.ShortHelp, AcceptSubCmd.LongHelp, AcceptSubCmd.CmdFunc(settings))
			cmd.CommandLong(ListSubCmd.Name, ListSubCmd.ShortHelp, ListSubCmd.LongHelp, ListSubCmd.CmdFunc(settings))
			cmd.CommandLong(RmSubCmd.Name, RmSubCmd.ShortHelp, RmSubCmd.LongHelp, RmSubCmd.CmdFunc(settings))
			cmd.CommandLong(SendSubCmd.Name, SendSubCmd.ShortHelp, SendSubCmd.LongHelp, SendSubCmd.CmdFunc(settings))
		}
	},
}

var AcceptSubCmd = models.Command{
	Name:      "accept",
	ShortHelp: "Accept an organization invite",
	LongHelp: "`invites accept` is an alternative form of accepting an invitation sent by email. " +
		"The invitation email you receive will have instructions as well as the invite code to use with this command. " +
		"Here is a sample command\n\n" +
		"```\ndatica -E \"<your_env_alias>\" invites accept 5a206aa8-04f4-4bc1-a017-ede7e6c7dbe2\n```",
	CmdFunc: func(settings *models.Settings) func(cmd *cli.Cmd) {
		return func(subCmd *cli.Cmd) {
			inviteCode := subCmd.StringArg("INVITE_CODE", "", "The invite code that was sent in the invite email")
			subCmd.Action = func() {
				p := prompts.New()
				a := auth.New(settings, p)
				if _, err := a.Signin(); err != nil {
					logrus.Fatal(err.Error())
				}

				err := CmdAccept(*inviteCode, New(settings), a, p)
				if err != nil {
					logrus.Fatal(err.Error())
				}
			}
			subCmd.Spec = "INVITE_CODE"
		}
	},
}

var ListSubCmd = models.Command{
	Name:      "list",
	ShortHelp: "List all pending organization invitations",
	LongHelp: "`invites list` lists all pending invites for the associated environment's organization. " +
		"Any invites that have already been accepted will not appear in this list. " +
		"To manage users who have already accepted invitations or are already granted access to your environment, use the [users](#users) group of commands. " +
		"Here is a sample command\n\n" +
		"```\ndatica -E \"<your_env_alias>\" invites list\n```",
	CmdFunc: func(settings *models.Settings) func(cmd *cli.Cmd) {
		return func(subCmd *cli.Cmd) {
			subCmd.Action = func() {
				if _, err := auth.New(settings, prompts.New()).Signin(); err != nil {
					logrus.Fatal(err.Error())
				}
				if err := config.CheckRequiredAssociation(settings); err != nil {
					logrus.Fatal(err.Error())
				}
				err := CmdList(settings.EnvironmentName, New(settings))
				if err != nil {
					logrus.Fatal(err.Error())
				}
			}
		}
	},
}

var RmSubCmd = models.Command{
	Name:      "rm",
	ShortHelp: "Remove a pending organization invitation",
	LongHelp: "`invites rm` removes a pending invitation found by using the [invites list](#invites-list) command. " +
		"Once an invite has already been accepted, it cannot be removed. " +
		"Removing an invitation is helpful if an email was misspelled or an invitation was sent to an incorrect email address. " +
		"If you want to revoke access to a user who already has been given access to your environment, use the [users rm](#users-rm) command. " +
		"Here is a sample command\n\n" +
		"```\ndatica -E \"<your_env_alias>\" invites rm 78b5d0ed-f71c-47f7-a4c8-6c8c58c29db1\n```",
	CmdFunc: func(settings *models.Settings) func(cmd *cli.Cmd) {
		return func(subCmd *cli.Cmd) {
			inviteID := subCmd.StringArg("INVITE_ID", "", "The ID of an invitation to remove")
			subCmd.Action = func() {
				if _, err := auth.New(settings, prompts.New()).Signin(); err != nil {
					logrus.Fatal(err.Error())
				}
				if err := config.CheckRequiredAssociation(settings); err != nil {
					logrus.Fatal(err.Error())
				}
				err := CmdRm(*inviteID, New(settings))
				if err != nil {
					logrus.Fatal(err.Error())
				}
			}
			subCmd.Spec = "INVITE_ID"
		}
	},
}

var SendSubCmd = models.Command{
	Name:      "send",
	ShortHelp: "Send an invite to a user by email for a given organization",
	LongHelp: "`invites send` invites a new user to your environment's organization. " +
		"The only piece of information required is the email address to send the invitation to. " +
		"The invited user will join the organization with no permissions. You must grant them permission through the dashboard. " +
		"The recipient does **not** need to have a Dashboard account in order to send them an invitation. " +
		"However, they will need to have a Dashboard account to accept the invitation. Here is a sample command\n\n" +
		"```\ndatica -E \"<your_env_alias>\" invites send coworker@datica.com\n```",
	CmdFunc: func(settings *models.Settings) func(cmd *cli.Cmd) {
		return func(subCmd *cli.Cmd) {
			email := subCmd.StringArg("EMAIL", "", "The email of a user to invite to the associated environment. This user does not need to have a Datica account prior to sending the invitation")
			subCmd.Action = func() {
				if _, err := auth.New(settings, prompts.New()).Signin(); err != nil {
					logrus.Fatal(err.Error())
				}
				if err := config.CheckRequiredAssociation(settings); err != nil {
					logrus.Fatal(err.Error())
				}
				err := CmdSend(*email, settings.EnvironmentName, New(settings), prompts.New())
				if err != nil {
					logrus.Fatal(err.Error())
				}
			}
			subCmd.Spec = "EMAIL"
		}
	},
}

// IInvites
type IInvites interface {
	Accept(inviteCode string) (string, error)
	List() (*[]models.Invite, error)
	Rm(inviteID string) error
	Send(email string) error
	ListOrgGroups() (*[]models.Group, error)
}

// SInvites is a concrete implementation of IInvites
type SInvites struct {
	Settings *models.Settings
}

// New generates a new instance of IInvites
func New(settings *models.Settings) IInvites {
	return &SInvites{
		Settings: settings,
	}
}
