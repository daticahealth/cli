package releases

import (
	"github.com/Sirupsen/logrus"
	"github.com/daticahealth/cli/commands/services"
	"github.com/daticahealth/cli/config"
	"github.com/daticahealth/cli/lib/auth"
	"github.com/daticahealth/cli/lib/prompts"
	"github.com/daticahealth/cli/models"
	"github.com/jault3/mow.cli"
)

// Cmd for keys
var Cmd = models.Command{
	Name:      "releases",
	ShortHelp: "Manage releases for code services",
	LongHelp: "The <code>releases</code> command allows you to manage your code service releases. " +
		"A release is automatically created each time you perform a git push. " +
		"The release is tagged with the git SHA of the commit. " +
		"Releases are a way of tagging specific points in time of your git history. " +
		"By default, the last three releases will be kept. " +
		"Please contact Support if you require more than the last three releases to be retained. " +
		"You can rollback to a specific release by using the [rollback](#rollback) command. " +
		"The releases command cannot be run directly but has subcommands.",
	CmdFunc: func(settings *models.Settings) func(cmd *cli.Cmd) {
		return func(cmd *cli.Cmd) {
			cmd.CommandLong(ListSubCmd.Name, ListSubCmd.ShortHelp, ListSubCmd.LongHelp, ListSubCmd.CmdFunc(settings))
			cmd.CommandLong(RmSubCmd.Name, RmSubCmd.ShortHelp, RmSubCmd.LongHelp, RmSubCmd.CmdFunc(settings))
			cmd.CommandLong(UpdateSubCmd.Name, UpdateSubCmd.ShortHelp, UpdateSubCmd.LongHelp, UpdateSubCmd.CmdFunc(settings))
		}
	},
}

var ListSubCmd = models.Command{
	Name:      "list",
	ShortHelp: "List all releases for a given code service",
	LongHelp: "<code>releases list</code> lists all of the releases for a given service. " +
		"A release is automatically created each time a git push is performed. " +
		"Here is a sample command\n\n" +
		"<pre>\ndatica -E \"<your_env_name>\" releases list code-1\n</pre>",
	CmdFunc: func(settings *models.Settings) func(cmd *cli.Cmd) {
		return func(cmd *cli.Cmd) {
			serviceName := cmd.StringArg("SERVICE_NAME", "", "The name of the service to list releases for")
			cmd.Action = func() {
				if _, err := auth.New(settings, prompts.New()).Signin(); err != nil {
					logrus.Fatal(err.Error())
				}
				if err := config.CheckRequiredAssociation(settings); err != nil {
					logrus.Fatal(err.Error())
				}
				err := CmdList(*serviceName, New(settings), services.New(settings))
				if err != nil {
					logrus.Fatal(err)
				}
			}
		}
	},
}

var RmSubCmd = models.Command{
	Name:      "rm",
	ShortHelp: "Remove a release from a code service",
	LongHelp: "<code>releases rm</code> removes an existing release. This is useful in the case of a misbehaving code service. " +
		"Removing the release avoids the risk of rolling back to a \"bad\" build. Here is a sample command\n\n" +
		"<pre>\ndatica -E \"<your_env_name>\" releases rm code-1 f93ced037f828dcaabccfc825e6d8d32cc5a1883\n</pre>",
	CmdFunc: func(settings *models.Settings) func(cmd *cli.Cmd) {
		return func(cmd *cli.Cmd) {
			serviceName := cmd.StringArg("SERVICE_NAME", "", "The name of the service to remove a release from")
			releaseName := cmd.StringArg("RELEASE_NAME", "", "The name of the release to remove")
			cmd.Action = func() {
				if _, err := auth.New(settings, prompts.New()).Signin(); err != nil {
					logrus.Fatal(err.Error())
				}
				if err := config.CheckRequiredAssociation(settings); err != nil {
					logrus.Fatal(err.Error())
				}
				err := CmdRm(*serviceName, *releaseName, New(settings), services.New(settings))
				if err != nil {
					logrus.Fatal(err)
				}
			}
		}
	},
}

var UpdateSubCmd = models.Command{
	Name:      "update",
	ShortHelp: "Update a release from a code service",
	LongHelp: "<code>releases update</code> allows you to rename or add notes to an existing release. " +
		"By default, releases are named with the git SHA of the commit used to create the release. " +
		"Renaming them allows you to organize your releases. Here is a sample command\n\n" +
		"<pre>\ndatica -E \"<your_env_name>\" releases update code-1 f93ced037f828dcaabccfc825e6d8d32cc5a1883 --notes \"This is a stable build\"\n</pre>",
	CmdFunc: func(settings *models.Settings) func(cmd *cli.Cmd) {
		return func(cmd *cli.Cmd) {
			serviceName := cmd.StringArg("SERVICE_NAME", "", "The name of the service to update a release for")
			releaseName := cmd.StringArg("RELEASE_NAME", "", "The name of the release to update")
			notes := cmd.StringOpt("n notes", "", "The new notes to save on the release.")
			cmd.Action = func() {
				if _, err := auth.New(settings, prompts.New()).Signin(); err != nil {
					logrus.Fatal(err.Error())
				}
				if err := config.CheckRequiredAssociation(settings); err != nil {
					logrus.Fatal(err.Error())
				}
				err := CmdUpdate(*serviceName, *releaseName, *notes, New(settings), services.New(settings))
				if err != nil {
					logrus.Fatal(err)
				}
			}
			cmd.Spec = "SERVICE_NAME RELEASE_NAME [--notes]"
		}
	},
}

type IReleases interface {
	List(svcID string) (*[]models.Release, error)
	Retrieve(releaseName, svcID string) (*models.Release, error)
	Rm(releaseName, svcID string) error
	Update(releaseName, svcID, notes string) error
}

type SReleases struct {
	Settings *models.Settings
}

func New(settings *models.Settings) IReleases {
	return &SReleases{Settings: settings}
}
