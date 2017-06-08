package vars

import (
	"github.com/Sirupsen/logrus"
	"github.com/daticahealth/cli/commands/environments"
	"github.com/daticahealth/cli/commands/services"
	"github.com/daticahealth/cli/config"
	"github.com/daticahealth/cli/lib/auth"
	"github.com/daticahealth/cli/lib/prompts"
	"github.com/daticahealth/cli/models"
	"github.com/jault3/mow.cli"
)

// Cmd is the contract between the user and the CLI. This specifies the command
// name, arguments, and required/optional arguments and flags for the command.
var Cmd = models.Command{
	Name:      "vars",
	ShortHelp: "Interaction with environment variables for the associated environment",
	LongHelp:  "The `vars` command allows you to manage environment variables for your code services. The vars command can not be run directly but has sub commands.",
	CmdFunc: func(settings *models.Settings) func(cmd *cli.Cmd) {
		return func(cmd *cli.Cmd) {
			cmd.CommandLong(ListSubCmd.Name, ListSubCmd.ShortHelp, ListSubCmd.LongHelp, ListSubCmd.CmdFunc(settings))
			cmd.CommandLong(SetSubCmd.Name, SetSubCmd.ShortHelp, SetSubCmd.LongHelp, SetSubCmd.CmdFunc(settings))
			cmd.CommandLong(UnsetSubCmd.Name, UnsetSubCmd.ShortHelp, UnsetSubCmd.LongHelp, UnsetSubCmd.CmdFunc(settings))
		}
	},
}

var ListSubCmd = models.Command{
	Name:      "list",
	ShortHelp: "List all environment variables",
	LongHelp: "`vars list` prints out all known environment variables for the given code service. " +
		"You can print out environment variables in JSON or YAML format through the `--json` or `--yaml` flags. " +
		"Here are some sample commands\n\n" +
		"```\ndatica -E \"<your_env_alias>\" vars list code-1\n" +
		"datica -E \"<your_env_alias>\" vars list code-1 --json\n```",
	CmdFunc: func(settings *models.Settings) func(cmd *cli.Cmd) {
		return func(subCmd *cli.Cmd) {
			serviceName := subCmd.StringArg("SERVICE_NAME", "", "The name of the service containing the environment variables.")
			json := subCmd.BoolOpt("json", false, "Output environment variables in JSON format")
			yaml := subCmd.BoolOpt("yaml", false, "Output environment variables in YAML format")
			subCmd.Action = func() {
				if _, err := auth.New(settings, prompts.New()).Signin(); err != nil {
					logrus.Fatal(err.Error())
				}
				if err := config.CheckRequiredAssociation(settings); err != nil {
					envs, errs := environments.New(settings).List()
					if errs != nil && len(errs) > 0 {
						logrus.Debugf("Error listing environments: %+v", errs)
					}
					config.StoreEnvironments(envs, settings)
					if err := config.CheckRequiredAssociation(settings); err != nil {
						logrus.Fatal(err.Error())
					}
				}
				var formatter Formatter
				if *json {
					formatter = &JSONFormatter{}
				} else if *yaml {
					formatter = &YAMLFormatter{}
				} else {
					formatter = &PlainFormatter{}
				}
				err := CmdList(*serviceName, formatter, New(settings), services.New(settings))
				if err != nil {
					logrus.Fatal(err.Error())
				}
			}
			subCmd.Spec = "SERVICE_NAME [--json | --yaml]"
		}
	},
}

var SetSubCmd = models.Command{
	Name:      "set",
	ShortHelp: "Set one or more new environment variables or update the values of existing ones",
	LongHelp: "`vars set` allows you to add new environment variables or update the value of an existing environment variable on the given code service. " +
		"You can set/update 1 or more environment variables at a time with this command by repeating the `-v` option multiple times. " +
		"Once new environment variables are added or values updated, a [redeploy](#redeploy) is required for the given code service to have access to the new values. " +
		"The environment variables must be of the form `<key>=<value>`. Here is a sample command\n\n" +
		"```\ndatica -E \"<your_env_alias>\" vars set code-1 -v AWS_ACCESS_KEY_ID=1234 -v AWS_SECRET_ACCESS_KEY=5678\n```",
	CmdFunc: func(settings *models.Settings) func(cmd *cli.Cmd) {
		return func(subCmd *cli.Cmd) {
			serviceName := subCmd.StringArg("SERVICE_NAME", "", "The name of the service on which the environment variables will be set.")
			variables := subCmd.Strings(cli.StringsOpt{
				Name:      "v variable",
				Value:     []string{},
				Desc:      "The env variable to set or update in the form \"<key>=<value>\"",
				HideValue: true,
			})
			fileName := subCmd.StringOpt("f file", "", "The path to a file to import environment variables from. This file can be in JSON, YAML, or KEY=VALUE format")
			subCmd.Action = func() {
				if _, err := auth.New(settings, prompts.New()).Signin(); err != nil {
					logrus.Fatal(err.Error())
				}
				if err := config.CheckRequiredAssociation(settings); err != nil {
					envs, errs := environments.New(settings).List()
					if errs != nil && len(errs) > 0 {
						logrus.Debugf("Error listing environments: %+v", errs)
					}
					config.StoreEnvironments(envs, settings)
					if err := config.CheckRequiredAssociation(settings); err != nil {
						logrus.Fatal(err.Error())
					}
				}
				err := CmdSet(*serviceName, *variables, *fileName, New(settings), services.New(settings))
				if err != nil {
					logrus.Fatal(err.Error())
				}
			}
			subCmd.Spec = "SERVICE_NAME (-v... | -f)"
		}
	},
}

var UnsetSubCmd = models.Command{
	Name:      "unset",
	ShortHelp: "Unset (delete) an existing environment variable",
	LongHelp: "`vars unset` removes an environment variables from the given code service. " +
		"Only the environment variable name is required to unset. " +
		"Once environment variables are unset, a [redeploy](#redeploy) is required for the given code service to realize the variable was removed. " +
		"Here is a sample command\n\n" +
		"```\ndatica -E \"<your_env_alias>\" vars unset code-1 AWS_ACCESS_KEY_ID\n```",
	CmdFunc: func(settings *models.Settings) func(cmd *cli.Cmd) {
		return func(subCmd *cli.Cmd) {
			serviceName := subCmd.StringArg("SERVICE_NAME", "", "The name of the service on which the environment variables will be unset.")
			variables := subCmd.Strings(cli.StringsOpt{
				Name:      "v variable",
				Value:     []string{},
				Desc:      "The names of environment variables to unset",
				HideValue: true,
			})
			subCmd.Action = func() {
				if _, err := auth.New(settings, prompts.New()).Signin(); err != nil {
					logrus.Fatal(err.Error())
				}
				if err := config.CheckRequiredAssociation(settings); err != nil {
					envs, errs := environments.New(settings).List()
					if errs != nil && len(errs) > 0 {
						logrus.Debugf("Error listing environments: %+v", errs)
					}
					config.StoreEnvironments(envs, settings)
					if err := config.CheckRequiredAssociation(settings); err != nil {
						logrus.Fatal(err.Error())
					}
				}
				err := CmdUnset(*serviceName, *variables, New(settings), services.New(settings))
				if err != nil {
					logrus.Fatal(err.Error())
				}
			}
			subCmd.Spec = "SERVICE_NAME -v..."
		}
	},
}

// IVars
type IVars interface {
	List(svcID string) (map[string]string, error)
	Set(svcID string, envVarsMap map[string]string) error
	Unset(svcID, key string) error
}

// SVars is a concrete implementation of IVars
type SVars struct {
	Settings *models.Settings
}

// New generates a new instance of IVars
func New(settings *models.Settings) IVars {
	return &SVars{
		Settings: settings,
	}
}
