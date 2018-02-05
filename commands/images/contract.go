package images

import (
	"github.com/Sirupsen/logrus"
	"github.com/daticahealth/cli/commands/environments"
	"github.com/daticahealth/cli/commands/images/delegations"
	"github.com/daticahealth/cli/commands/images/tags"
	"github.com/daticahealth/cli/commands/images/targets"
	"github.com/daticahealth/cli/config"
	"github.com/daticahealth/cli/lib/auth"
	"github.com/daticahealth/cli/lib/images"
	"github.com/daticahealth/cli/lib/prompts"
	"github.com/daticahealth/cli/models"
	"github.com/jault3/mow.cli"
)

// Cmd is the contract between the user and the CLI. This specifies the command
// name, arguments, and required/optional arguments and flags for the command.
var Cmd = models.Command{
	Name:      "images",
	ShortHelp: "Operations for working with images",
	LongHelp: "<code>images</code> allows interactions with container images and tags. " +
		"This command cannot be run directly, but has subcommands.",
	CmdFunc: func(settings *models.Settings) func(cmd *cli.Cmd) {
		return func(cmd *cli.Cmd) {
			cmd.CommandLong(listCmd.Name, listCmd.ShortHelp, listCmd.LongHelp, listCmd.CmdFunc(settings))
			cmd.CommandLong(pushCmd.Name, pushCmd.ShortHelp, pushCmd.LongHelp, pushCmd.CmdFunc(settings))
			cmd.CommandLong(pullCmd.Name, pullCmd.ShortHelp, pullCmd.LongHelp, pullCmd.CmdFunc(settings))
			cmd.CommandLong(targets.Cmd.Name, targets.Cmd.ShortHelp, targets.Cmd.LongHelp, targets.Cmd.CmdFunc(settings))
			cmd.CommandLong(tags.Cmd.Name, tags.Cmd.ShortHelp, tags.Cmd.LongHelp, tags.Cmd.CmdFunc(settings))
			cmd.CommandLong(delegations.Cmd.Name, delegations.Cmd.ShortHelp, delegations.Cmd.LongHelp, delegations.Cmd.CmdFunc(settings))
		}
	},
}

var listCmd = models.Command{
	Name:      "list",
	ShortHelp: "List images available for an environment",
	LongHelp: "<code>images list</code> lists available images for an environment. " +
		"These images must be pushed to the registry for the environment and deployed in order to show. Here is a sample command:\n\n" +
		"<pre>\ndatica -E \"<your_env_name>\" images list\n</pre>",
	CmdFunc: func(settings *models.Settings) func(cmd *cli.Cmd) {
		return func(cmd *cli.Cmd) {
			cmd.Action = func() {
				if _, err := auth.New(settings, prompts.New()).Signin(); err != nil {
					logrus.Fatal(err.Error())
				}
				if err := config.CheckRequiredAssociation(settings); err != nil {
					logrus.Fatal(err.Error())
				}
				err := cmdImageList(settings.EnvironmentID, environments.New(settings), images.New(settings))
				if err != nil {
					logrus.Fatalln(err.Error())
				}
			}
		}
	},
}

var pushCmd = models.Command{
	Name:      "push",
	ShortHelp: "Push an image for your environment",
	LongHelp: "<code>images push</code> pushes a new image to the registry for your environment. " +
		"The image will be retagged with the Datica registry and your namespace appended to the front if not provided. " +
		"If no tag is specified, the image will be tagged \"latest\"\n" +
		"Note: Pushed images will not be returned by the `images list` command until they have been deployed. Here is a sample command:\n\n" +
		"<pre>\ndatica -E \"<your_env_name>\" images push <image>:<tag>\n</pre>",
	CmdFunc: func(settings *models.Settings) func(cmd *cli.Cmd) {
		return func(cmd *cli.Cmd) {
			image := cmd.StringArg("TAGGED_IMAGE", "", "The name of the image to push. (e.g. 'my-image:tag')")
			cmd.Action = func() {
				user, err := auth.New(settings, prompts.New()).Signin()
				if err != nil {
					logrus.Fatal(err.Error())
				}
				if err = config.CheckRequiredAssociation(settings); err != nil {
					logrus.Fatal(err.Error())
				}
				err = cmdImagePush(settings.EnvironmentID, *image, user, environments.New(settings), images.New(settings), prompts.New())
				if err != nil {
					logrus.Fatalln(err.Error())
				}
			}
		}
	},
}

var pullCmd = models.Command{
	Name:      "pull",
	ShortHelp: "Pull an image from your environment namespace",
	LongHelp: "<code>images pull</code> pulls an image from the registry for your environment and verifies its content against a signed target. " +
		"The image will be pulled with the Datica registry and your environment namespace appended to the front if not provided. Here is a sample command:\n\n" +
		"<pre>\ndatica -E \"<your_env_name>\" images pull <image>:<tag>\n</pre>",
	CmdFunc: func(settings *models.Settings) func(cmd *cli.Cmd) {
		return func(cmd *cli.Cmd) {
			image := cmd.StringArg("TAGGED_IMAGE", "", "The name of the image to pull. (e.g. 'my-image:tag')")
			cmd.Action = func() {
				user, err := auth.New(settings, prompts.New()).Signin()
				if err != nil {
					logrus.Fatal(err.Error())
				}
				if err = config.CheckRequiredAssociation(settings); err != nil {
					logrus.Fatal(err.Error())
				}
				err = cmdImagePull(settings.EnvironmentID, *image, user, environments.New(settings), images.New(settings))
				if err != nil {
					logrus.Fatalln(err.Error())
				}
			}
		}
	},
}
