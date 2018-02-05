package delegations

import (
	"github.com/Sirupsen/logrus"
	"github.com/daticahealth/cli/commands/environments"
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
	Name:      "delegations",
	ShortHelp: "Operations for working with delegation keys",
	LongHelp: "<code>delegations</code> allows interactions with delegation keys for collaborators to sign images. " +
		"This command cannot be run directly, but has subcommands.",
	CmdFunc: func(settings *models.Settings) func(cmd *cli.Cmd) {
		return func(cmd *cli.Cmd) {
			cmd.CommandLong(createCmd.Name, createCmd.ShortHelp, createCmd.LongHelp, createCmd.CmdFunc(settings))
			// cmd.CommandLong(listCmd.Name, listCmd.ShortHelp, listCmd.LongHelp, listCmd.CmdFunc(settings))
			// cmd.CommandLong(deleteCmd.Name, deleteCmd.ShortHelp, deleteCmd.LongHelp, deleteCmd.CmdFunc(settings))
		}
	},
}

var createCmd = models.Command{
	Name:      "create",
	ShortHelp: "Create a delegation key and certificate",
	LongHelp: "<code>images delgations create</code> creates a delegation key and certificate to be addded to a trust repository for signing." +
		"The delegation certificate will expire after one year by default. You can create a new cert from the same key by specifying the --key option. Here is a sample command:\n\n" +
		"<pre>\ndatica -E \"<your_env_name>\" images delegations create\n</pre>",
	CmdFunc: func(settings *models.Settings) func(cmd *cli.Cmd) {
		return func(cmd *cli.Cmd) {
			keyPath := cmd.StringOpt("--key", "", "Path to a premade key to use for creating a new public delegation certificate")
			size := cmd.IntOpt("--size", 2048, "The bit length of the key. Must be at least 2048") //TODO: Discuss
			expiration := cmd.IntOpt("--expiration", 365, "The number of days that the delegation certificiate will remain valid")
			importKey := cmd.BoolOpt("-i, --import", false, "Import the delegation key into your local trust directory")
			cmd.Action = func() {
				_, err := auth.New(settings, prompts.New()).Signin()
				if err != nil {
					logrus.Fatal(err.Error())
				}
				if err = config.CheckRequiredAssociation(settings); err != nil {
					logrus.Fatal(err.Error())
				}
				if err = cmdDelegationsCreate(settings.EnvironmentID, *keyPath, *size, *expiration, *importKey, environments.New(settings), images.New(settings), prompts.New()); err != nil {
					logrus.Fatalln(err.Error())
				}
			}
		}
	},
}

var addCmd = models.Command{
	Name:      "add",
	ShortHelp: "Add a delegation certificate to a repository",
	LongHelp: "<code>images delgations add</code> grants signing priveledges to a trust repository for a delgation key using its public certificate." +
		"You must have access to the root and targets keys for the specified repository. Here is a sample command:\n\n" +
		"<pre>\ndatica -E \"<your_env_name>\" images delegations add <image> /path/to/delegation/certificate\n</pre>",
	CmdFunc: func(settings *models.Settings) func(cmd *cli.Cmd) {
		return func(cmd *cli.Cmd) {
			image := cmd.StringArg("IMAGE_NAME", "", "The name of the image to grant signing priveledges for. (e.g. 'my-image')")
			certPath := cmd.StringArg("CERT_PATH", "", "Path to the public certificate for a delegation key. (e.g. './delegation.crt')")
			cmd.Action = func() {
				user, err := auth.New(settings, prompts.New()).Signin()
				if err != nil {
					logrus.Fatal(err.Error())
				}
				if err = config.CheckRequiredAssociation(settings); err != nil {
					logrus.Fatal(err.Error())
				}
				if err = cmdDelegationsAdd(settings.EnvironmentID, *image, *certPath, user, environments.New(settings), images.New(settings), prompts.New()); err != nil {
					logrus.Fatalln(err.Error())
				}
			}
		}
	},
}

// var listCmd = models.Command{
// 	Name:      "list",
// 	ShortHelp: "List delegation keys for a trust repository",
// 	LongHelp: "<code>images delgations list</code> lists delegation keys for a trust repository. Here is a sample command:\n\n" +
// 		"<pre>\ndatica -E \"<your_env_name>\" images delegations list <image>\n</pre>",
// 	CmdFunc: func(settings *models.Settings) func(cmd *cli.Cmd) {
// 		return func(cmd *cli.Cmd) {
// 			image := cmd.StringArg("IMAGE_NAME", "", "The name of the image to list delegation keys for. (e.g. 'my-image')")
// 			cmd.Action = func() {
// 				user, err := auth.New(settings, prompts.New()).Signin()
// 				if err != nil {
// 					logrus.Fatal(err.Error())
// 				}
// 				if err = config.CheckRequiredAssociation(settings); err != nil {
// 					logrus.Fatal(err.Error())
// 				}
// 				// if err = cmdTargetsList(settings.EnvironmentID, *image, user, environments.New(settings), images.New(settings)); err != nil {
// 				// 	logrus.Fatalln(err.Error())
// 				// }
// 			}
// 		}
// 	},
// }

// var deleteCmd = models.Command{
// 	Name:      "rm",
// 	ShortHelp: "Delete a signed target for a given image",
// 	LongHelp: "<code>images targets rm</code> deletes a signed target for a given image. " +
// 		"You environment namespace will be filled in for you if not provided. Here is a sample command:\n\n" +
// 		"<pre>\ndatica -E \"<your_env_name>\" images targets rm <image>:<tag>\n</pre>",
// 	CmdFunc: func(settings *models.Settings) func(cmd *cli.Cmd) {
// 		return func(cmd *cli.Cmd) {
// 			image := cmd.StringArg("TAGGED_IMAGE", "", "The name and tag of the image to delete targets from. (e.g. 'my-image:tag')")
// 			cmd.Action = func() {
// 				user, err := auth.New(settings, prompts.New()).Signin()
// 				if err != nil {
// 					logrus.Fatal(err.Error())
// 				}
// 				if err := config.CheckRequiredAssociation(settings); err != nil {
// 					logrus.Fatal(err.Error())
// 				}
// 				if err := cmdTargetsDelete(settings.EnvironmentID, *image, user, environments.New(settings), images.New(settings), prompts.New()); err != nil {
// 					logrus.Fatalln(err.Error())
// 				}
// 			}
// 		}
// 	},
// }
