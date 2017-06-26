package certs

import (
	"github.com/Sirupsen/logrus"
	"github.com/daticahealth/cli/commands/services"
	"github.com/daticahealth/cli/commands/ssl"
	"github.com/daticahealth/cli/config"
	"github.com/daticahealth/cli/lib/auth"
	"github.com/daticahealth/cli/lib/prompts"
	"github.com/daticahealth/cli/models"
	"github.com/jault3/mow.cli"
)

// Cmd is the contract between the user and the CLI. This specifies the command
// name, arguments, and required/optional arguments and flags for the command.
var Cmd = models.Command{
	Name:      "certs",
	ShortHelp: "Manage your SSL certificates and domains",
	LongHelp:  "The `certs` command gives access to certificate and private key management for public facing services. The certs command cannot be run directly but has sub commands.",
	CmdFunc: func(settings *models.Settings) func(cmd *cli.Cmd) {
		return func(cmd *cli.Cmd) {
			cmd.CommandLong(CreateSubCmd.Name, CreateSubCmd.ShortHelp, CreateSubCmd.LongHelp, CreateSubCmd.CmdFunc(settings))
			cmd.CommandLong(ListSubCmd.Name, ListSubCmd.ShortHelp, ListSubCmd.LongHelp, ListSubCmd.CmdFunc(settings))
			cmd.CommandLong(RmSubCmd.Name, RmSubCmd.ShortHelp, RmSubCmd.LongHelp, RmSubCmd.CmdFunc(settings))
			cmd.CommandLong(UpdateSubCmd.Name, UpdateSubCmd.ShortHelp, UpdateSubCmd.LongHelp, UpdateSubCmd.CmdFunc(settings))
		}
	},
}

var CreateSubCmd = models.Command{
	Name:      "create",
	ShortHelp: "Create a new domain with an SSL certificate and private key",
	LongHelp: "`certs create` allows you to upload an SSL certificate and private key which can be used to secure your public facing code service. " +
		"Cert creation can be done at any time, even after environment provisioning, but must be done before [creating a site](#sites-create). " +
		"When creating a cert, the CLI will check to ensure the certificate and private key match. If you are using a self signed cert, pass in the `-s` flag and the hostname check will be skipped. " +
		"Datica requires that your certificate include your own certificate, intermediate certificates, and the root certificate in that order. " +
		"If you only include your certificate, the CLI will attempt to resolve this and fetch intermediate and root certificates for you. " +
		"It is advised that you create a full chain before running this command as the `-r` flag is accomplished on a \"best effort\" basis.\n\n" +
		"The `HOSTNAME` for a certificate does not need to match the valid Subject of the actual SSL certificate nor does it need to match the `site` name used in the `sites create` command. " +
		"The `HOSTNAME` is used for organizational purposes only and can be named anything with the exclusion of the following characters: `/`, `&`, `%`. Here is a sample command\n\n" +
		"```\ndatica -E \"<your_env_alias>\" certs create wildcard_mysitecom ~/path/to/cert.pem ~/path/to/priv.key\n```",
	CmdFunc: func(settings *models.Settings) func(cmd *cli.Cmd) {
		return func(subCmd *cli.Cmd) {
			name := subCmd.StringArg("NAME", "", "The name of this SSL certificate plus private key pair")
			pubKeyPath := subCmd.StringArg("PUBLIC_KEY_PATH", "", "The path to a public key file in PEM format")
			privKeyPath := subCmd.StringArg("PRIVATE_KEY_PATH", "", "The path to an unencrypted private key file in PEM format")
			selfSigned := subCmd.BoolOpt("s self-signed", false, "Whether or not the given SSL certificate and private key are self signed")
			resolve := subCmd.BoolOpt("r resolve", true, "Whether or not to attempt to automatically resolve incomplete SSL certificate issues")
			subCmd.Action = func() {
				if _, err := auth.New(settings, prompts.New()).Signin(); err != nil {
					logrus.Fatal(err.Error())
				}
				if err := config.CheckRequiredAssociation(settings); err != nil {
					logrus.Fatal(err.Error())
				}
				err := CmdCreate(*name, *pubKeyPath, *privKeyPath, *selfSigned, *resolve, New(settings), services.New(settings), ssl.New(settings))
				if err != nil {
					logrus.Fatal(err.Error())
				}
			}
			subCmd.Spec = "NAME PUBLIC_KEY_PATH PRIVATE_KEY_PATH [-s] [-r]"
		}
	},
}

var ListSubCmd = models.Command{
	Name:      "list",
	ShortHelp: "List all existing domains that have SSL certificate and private key pairs",
	LongHelp: "`certs list` lists all of the available certs you have created on your environment. " +
		"The displayed names are the names that should be used as the `DOMAIN` parameter in the [sites create](#sites-create) command. Here is a sample command\n\n" +
		"```\ndatica -E \"<your_env_alias>\" certs list\n```",
	CmdFunc: func(settings *models.Settings) func(cmd *cli.Cmd) {
		return func(subCmd *cli.Cmd) {
			subCmd.Action = func() {
				if _, err := auth.New(settings, prompts.New()).Signin(); err != nil {
					logrus.Fatal(err.Error())
				}
				if err := config.CheckRequiredAssociation(settings); err != nil {
					logrus.Fatal(err.Error())
				}
				err := CmdList(New(settings), services.New(settings))
				if err != nil {
					logrus.Fatal(err.Error())
				}
			}
		}
	},
}

var RmSubCmd = models.Command{
	Name:      "rm",
	ShortHelp: "Remove an existing domain and its associated SSL certificate and private key pair",
	LongHelp: "`certs rm` allows you to delete old certificate and private key pairs. Only certs that are not in use by a site can be deleted. Here is a sample command\n\n" +
		"```\ndatica -E \"<your_env_alias>\" certs rm mywebsite.com\n```",
	CmdFunc: func(settings *models.Settings) func(cmd *cli.Cmd) {
		return func(subCmd *cli.Cmd) {
			name := subCmd.StringArg("HOSTNAME", "", "The hostname of the domain and SSL certificate and private key pair")
			subCmd.Action = func() {
				if _, err := auth.New(settings, prompts.New()).Signin(); err != nil {
					logrus.Fatal(err.Error())
				}
				if err := config.CheckRequiredAssociation(settings); err != nil {
					logrus.Fatal(err.Error())
				}
				err := CmdRm(*name, New(settings), services.New(settings))
				if err != nil {
					logrus.Fatal(err.Error())
				}
			}
			subCmd.Spec = "HOSTNAME"
		}
	},
}

var UpdateSubCmd = models.Command{
	Name:      "update",
	ShortHelp: "Update the SSL certificate and private key pair for an existing domain",
	LongHelp: "`certs update` works nearly identical to the [certs create](#certs-create) command. " +
		"All rules regarding self signed certs and certificate resolution from the `certs create` command apply to the `certs update` command. " +
		"This is useful for when your certificates have expired and you need to upload new ones. Update your certs and then redeploy your service_proxy. Here is a sample command\n\n" +
		"```\ndatica -E \"<your_env_alias>\" certs update mywebsite.com ~/path/to/new/cert.pem ~/path/to/new/priv.key\n```",
	CmdFunc: func(settings *models.Settings) func(cmd *cli.Cmd) {
		return func(subCmd *cli.Cmd) {
			name := subCmd.StringArg("NAME", "", "The name of this SSL certificate and private key pair")
			pubKeyPath := subCmd.StringArg("PUBLIC_KEY_PATH", "", "The path to a public key file in PEM format")
			privKeyPath := subCmd.StringArg("PRIVATE_KEY_PATH", "", "The path to an unencrypted private key file in PEM format")
			selfSigned := subCmd.BoolOpt("s self-signed", false, "Whether or not the given SSL certificate and private key are self signed")
			resolve := subCmd.BoolOpt("r resolve", true, "Whether or not to attempt to automatically resolve incomplete SSL certificate issues")
			subCmd.Action = func() {
				if _, err := auth.New(settings, prompts.New()).Signin(); err != nil {
					logrus.Fatal(err.Error())
				}
				if err := config.CheckRequiredAssociation(settings); err != nil {
					logrus.Fatal(err.Error())
				}
				err := CmdUpdate(*name, *pubKeyPath, *privKeyPath, *selfSigned, *resolve, New(settings), services.New(settings), ssl.New(settings))
				if err != nil {
					logrus.Fatal(err.Error())
				}
			}
			subCmd.Spec = "NAME PUBLIC_KEY_PATH PRIVATE_KEY_PATH [-s] [-r]"
		}
	},
}

// ICerts
type ICerts interface {
	Create(hostname, pubKey, privKey, svcID string) error
	Update(hostname, pubKey, privKey, svcID string) error
	List(svcID string) (*[]models.Cert, error)
	Rm(hostname, svcID string) error
}

// SCerts is a concrete implementation of ICerts
type SCerts struct {
	Settings *models.Settings
}

// New returns an instance of ICerts
func New(settings *models.Settings) ICerts {
	return &SCerts{
		Settings: settings,
	}
}
