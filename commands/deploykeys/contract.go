package deploykeys

import (
	"crypto/rsa"

	"golang.org/x/crypto/ssh"

	"github.com/Sirupsen/logrus"
	"github.com/daticahealth/cli/commands/services"
	"github.com/daticahealth/cli/config"
	"github.com/daticahealth/cli/lib/auth"
	"github.com/daticahealth/cli/lib/prompts"
	"github.com/daticahealth/cli/models"
	"github.com/jault3/mow.cli"
)

var Cmd = models.Command{
	Name:      "deploy-keys",
	ShortHelp: "Tasks for SSH deploy keys",
	LongHelp:  "The `deploy-keys` command gives access to SSH deploy keys for environment services. The deploy-keys command can not be run directly but has sub commands.",
	CmdFunc: func(settings *models.Settings) func(cmd *cli.Cmd) {
		return func(cmd *cli.Cmd) {
			cmd.CommandLong(AddSubCmd.Name, AddSubCmd.ShortHelp, AddSubCmd.LongHelp, AddSubCmd.CmdFunc(settings))
			cmd.CommandLong(ListSubCmd.Name, ListSubCmd.ShortHelp, ListSubCmd.LongHelp, ListSubCmd.CmdFunc(settings))
			cmd.CommandLong(RmSubCmd.Name, RmSubCmd.ShortHelp, RmSubCmd.LongHelp, RmSubCmd.CmdFunc(settings))
		}
	},
}

var AddSubCmd = models.Command{
	Name:      "add",
	ShortHelp: "Add a new deploy key",
	LongHelp: "`deploy-keys add` allows you to upload an SSH public key in OpenSSH format. " +
		"These keys are used for pushing code to your code services but are not required. " +
		"You should be using personal SSH keys with the [keys](#keys) command unless you are pushing code from Continuous Integration or Continuous Deployment scenarios. " +
		"Deploy keys are intended to be shared among an organization. Here are some sample commands\n\n" +
		"```\ndatica -E \"<your_env_alias>\" deploy-keys add app01_public ~/.ssh/app01_rsa.pub app01\n```",
	CmdFunc: func(settings *models.Settings) func(cmd *cli.Cmd) {
		return func(subCmd *cli.Cmd) {
			name := subCmd.StringArg("NAME", "", "The name for the new key, for your own purposes")
			path := subCmd.StringArg("KEY_PATH", "", "Relative path to the SSH key file")
			serviceName := subCmd.StringArg("SERVICE_NAME", "", "The name of the code service to add this deploy key to")
			subCmd.Action = func() {
				if _, err := auth.New(settings, prompts.New()).Signin(); err != nil {
					logrus.Fatal(err.Error())
				}
				if err := config.CheckRequiredAssociation(settings); err != nil {
					logrus.Fatal(err.Error())
				}
				err := CmdAdd(*name, *path, *serviceName, New(settings), services.New(settings))
				if err != nil {
					logrus.Fatal(err.Error())
				}
			}
			subCmd.Spec = "NAME KEY_PATH SERVICE_NAME"
		}
	},
}

var ListSubCmd = models.Command{
	Name:      "list",
	ShortHelp: "List all deploy keys",
	LongHelp: "`deploy-keys list` will list all of your previously uploaded deploy keys by name including the key's fingerprint in SHA256 format. Here is a sample command\n\n" +
		"```\ndatica -E \"<your_env_alias>\" deploy-keys list app01\n```",
	CmdFunc: func(settings *models.Settings) func(cmd *cli.Cmd) {
		return func(subCmd *cli.Cmd) {
			serviceName := subCmd.StringArg("SERVICE_NAME", "", "The name of the code service to list deploy keys")
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
			subCmd.Spec = "SERVICE_NAME"
		}
	},
}

var RmSubCmd = models.Command{
	Name:      "rm",
	ShortHelp: "Remove a deploy key",
	LongHelp: "`deploy-keys rm` will remove a previously created deploy key by name. " +
		"It is a good idea to rotate deploy keys on a set schedule as they are intended to be shared among an organization. " +
		"Here are some sample commands\n\n" +
		"```\ndatica -E \"<your_env_alias>\" deploy-keys rm app01_public app01\n```",
	CmdFunc: func(settings *models.Settings) func(cmd *cli.Cmd) {
		return func(subCmd *cli.Cmd) {
			name := subCmd.StringArg("NAME", "", "The name of the key to remove")
			serviceName := subCmd.StringArg("SERVICE_NAME", "", "The name of the code service to remove this deploy key from")
			subCmd.Action = func() {
				if _, err := auth.New(settings, prompts.New()).Signin(); err != nil {
					logrus.Fatal(err.Error())
				}
				if err := config.CheckRequiredAssociation(settings); err != nil {
					logrus.Fatal(err.Error())
				}
				err := CmdRm(*name, *serviceName, New(settings), services.New(settings))
				if err != nil {
					logrus.Fatal(err.Error())
				}
			}
			subCmd.Spec = "NAME SERVICE_NAME"
		}
	},
}

type IDeployKeys interface {
	Add(name, keyType, key, svcID string) error
	ExtractPublicKey(privKey *rsa.PrivateKey) (ssh.PublicKey, error)
	List(svcID string) (*[]models.DeployKey, error)
	ParsePrivateKey(b []byte) (*rsa.PrivateKey, error)
	ParsePublicKey(b []byte) (ssh.PublicKey, error)
	Rm(name, keyType, svcID string) error
}

type SDeployKeys struct {
	Settings *models.Settings
}

func New(settings *models.Settings) IDeployKeys {
	return &SDeployKeys{
		Settings: settings,
	}
}
