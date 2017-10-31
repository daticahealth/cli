package ssl

import (
	"fmt"

	"github.com/Sirupsen/logrus"
	"github.com/daticahealth/cli/models"
	"github.com/jault3/mow.cli"
)

// IncompleteChainError represents an error thrown when an SSL certificate is
// signed by an invalid CA or has an incomplete certificate chain.
type IncompleteChainError struct {
	Message string
	Err     error
}

func (i *IncompleteChainError) Error() string {
	return fmt.Sprintf("%s: %s", i.Message, i.Err.Error())
}

// HostnameMismatchError represents an error thrown when the hostname listed in
// an SSL certificate is different from the user input.
type HostnameMismatchError struct {
	Message string
	Err     error
}

func (h *HostnameMismatchError) Error() string {
	return fmt.Sprintf("%s: %s", h.Message, h.Err.Error())
}

// Cmd is the contract between the user and the CLI. This specifies the command
// name, arguments, and required/optional arguments and flags for the command.
var Cmd = models.Command{
	Name:      "ssl",
	ShortHelp: "Perform operations on local certificates to verify their validity",
	LongHelp:  "The `ssl` command offers access to subcommands that deal with SSL certificates. You cannot run the SSL command directly but must call a subcommand.",
	CmdFunc: func(settings *models.Settings) func(cmd *cli.Cmd) {
		return func(cmd *cli.Cmd) {
			cmd.CommandLong(ResolveSubCmd.Name, ResolveSubCmd.ShortHelp, ResolveSubCmd.LongHelp, ResolveSubCmd.CmdFunc(settings))
			cmd.CommandLong(VerifySubCmd.Name, VerifySubCmd.ShortHelp, VerifySubCmd.LongHelp, VerifySubCmd.CmdFunc(settings))
		}
	},
}

var ResolveSubCmd = models.Command{
	Name:      "resolve",
	ShortHelp: "Verify that an SSL certificate is signed by a valid CA and attempt to resolve any incomplete certificate chains that are found",
	LongHelp: "`ssl resolve` is a tool that will attempt to fix invalid SSL certificates chains. " +
		"A well formatted SSL certificate will include your certificate, intermediate certificates, and root certificates. " +
		"It should follow this format\n\n" +
		"```\n-----BEGIN CERTIFICATE-----\n" +
		"<Your SSL certificate here>\n" +
		"-----END CERTIFICATE-----\n" +
		"-----BEGIN CERTIFICATE-----\n" +
		"<One or more intermediate certificates here>\n" +
		"-----END CERTIFICATE-----\n" +
		"-----BEGIN CERTIFICATE-----\n" +
		"<Root CA here>\n" +
		"-----END CERTIFICATE-----\n```\n\n" +
		"If your certificate only includes your own certificate, such as the following format shows\n\n" +
		"```\n-----BEGIN CERTIFICATE-----\n" +
		"<Your SSL certificate here>\n" +
		"-----END CERTIFICATE-----\n```\n\n" +
		"then the SSL resolve command will attempt to resolve this by downloading public intermediate certificates and root certificates. " +
		"A general rule of thumb is, if your certificate passes the `ssl resolve` check, it will almost always work on the Datica platform. " +
		"You can specify where to save the updated chain or omit the `OUTPUT` argument to print it to STDOUT.\n\n" +
		"Please note you all certificates and private keys should be in PEM format. " +
		"You cannot use self signed certificates with this command as they cannot be resolved as they are not signed by a valid CA. " +
		"Here are some sample commands\n\n" +
		"```\ndatica ssl resolve ~/mysites_cert.pem ~/mysites_key.key *.mysite.com ~/updated_mysites_cert.pem -f\n" +
		"datica ssl resolve ~/mysites_cert.pem ~/mysites_key.key *.mysite.com\n```",
	CmdFunc: func(settings *models.Settings) func(cmd *cli.Cmd) {
		return func(subCmd *cli.Cmd) {
			chain := subCmd.StringArg("CHAIN", "", "The path to your full certificate chain in PEM format")
			privateKey := subCmd.StringArg("PRIVATE_KEY", "", "The path to your private key in PEM format")
			hostname := subCmd.StringArg("HOSTNAME", "", "The hostname that should match your certificate (e.g. \"*.datica.com\")")
			output := subCmd.StringArg("OUTPUT", "", "The path of a file to save your properly resolved certificate chain (defaults to STDOUT)")
			force := subCmd.BoolOpt("f force", false, "If an output file is specified and already exists, setting force to true will overwrite the existing output file")
			subCmd.Action = func() {
				err := CmdResolve(*chain, *privateKey, *hostname, *output, *force, New(settings))
				if err != nil {
					logrus.Fatal(err.Error())
				}
			}
			subCmd.Spec = "CHAIN PRIVATE_KEY HOSTNAME [OUTPUT] [-f]"
		}
	},
}

var VerifySubCmd = models.Command{
	Name:      "verify",
	ShortHelp: "Verify whether a certificate chain is complete and if it matches the given private key",
	LongHelp: "`ssl verify` will tell you if your SSL certificate and private key are properly formatted for use with Datica's Compliant Cloud. " +
		"Before uploading a certificate to Datica you should verify it creates a full chain and matches the given private key with this command. " +
		"Both your chain and private key should be **unencrypted** and in **PEM** format. " +
		"The private key is the only key in the key file. " +
		"However, for the chain, you should include your SSL certificate, intermediate certificates, and root certificate in the following order and format.\n\n" +
		"```\n-----BEGIN CERTIFICATE-----\n" +
		"<Your SSL certificate here>\n" +
		"-----END CERTIFICATE-----\n" +
		"-----BEGIN CERTIFICATE-----\n" +
		"<One or more intermediate certificates here>\n" +
		"-----END CERTIFICATE-----\n" +
		"-----BEGIN CERTIFICATE-----\n" +
		"<Root CA here>\n" +
		"-----END CERTIFICATE-----\n```\n\n" +
		"This command also requires you to specify the hostname that you are using the SSL certificate for in order to verify that the hostname matches what is in the chain. " +
		"If it is a wildcard certificate, your hostname would be in the following format: `*.datica.com`. " +
		"This command will verify a complete chain can be made from your certificate down through the intermediate certificates all the way to a root certificate that you have given or one found in your system.\n\n" +
		"You can also use this command to verify self-signed certificates match a given private key. " +
		"To do so, add the `-s` option which will skip verifying the certificate to root chain and just tell you if your certificate matches your private key. " +
		"Please note that the empty quotes are required for checking self signed certificates. " +
		"This is the required parameter HOSTNAME which is ignored when checking self signed certificates. " +
		"Here are some sample commands\n\n" +
		"```\ndatica ssl verify ./datica.crt ./datica.key *.datica.com\n" +
		"datica ssl verify ~/self-signed.crt ~/self-signed.key \"\" -s\n```",
	CmdFunc: func(settings *models.Settings) func(cmd *cli.Cmd) {
		return func(subCmd *cli.Cmd) {
			chain := subCmd.StringArg("CHAIN", "", "The path to your full certificate chain in PEM format")
			privateKey := subCmd.StringArg("PRIVATE_KEY", "", "The path to your private key in PEM format")
			hostname := subCmd.StringArg("HOSTNAME", "", "The hostname that should match your certificate (e.g. \"*.datica.com\")")
			selfSigned := subCmd.BoolOpt("s self-signed", false, "Whether or not the certificate is self signed. If set, chain verification is skipped")
			subCmd.Action = func() {
				err := CmdVerify(*chain, *privateKey, *hostname, *selfSigned, New(settings))
				if err != nil {
					logrus.Fatal(err.Error())
				}
			}
			subCmd.Spec = "CHAIN PRIVATE_KEY HOSTNAME [-s]"
		}
	},
}

// ISSL
type ISSL interface {
	Verify(chainPath, privateKeyPath, hostname string, selfSigned bool) error
	Resolve(chainPath string) ([]byte, error)
}

// SSSL is a concrete implementation of ISSL
type SSSL struct {
	Settings *models.Settings
}

// New generates a new instance of ISSL
func New(settings *models.Settings) ISSL {
	return &SSSL{
		Settings: settings,
	}
}
