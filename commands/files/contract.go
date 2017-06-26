package files

import (
	"github.com/Sirupsen/logrus"
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
	Name:      "files",
	ShortHelp: "Tasks for managing service files",
	LongHelp: "The `files` command gives access to service files on your environment's services. " +
		"Service files can include Nginx configs, SSL certificates, and any other file that might be injected into your running service. " +
		"The files command can not be run directly but has sub commands.",
	CmdFunc: func(settings *models.Settings) func(cmd *cli.Cmd) {
		return func(cmd *cli.Cmd) {
			cmd.CommandLong(DownloadSubCmd.Name, DownloadSubCmd.ShortHelp, DownloadSubCmd.LongHelp, DownloadSubCmd.CmdFunc(settings))
			cmd.CommandLong(ListSubCmd.Name, ListSubCmd.ShortHelp, ListSubCmd.LongHelp, ListSubCmd.CmdFunc(settings))
		}
	},
}

var DownloadSubCmd = models.Command{
	Name:      "download",
	ShortHelp: "Download a file to your localhost with the same file permissions as on the remote host or print it to stdout",
	LongHelp: "`files download` allows you to view the contents of a service file and save it to your local machine. " +
		"Most service files are stored on your service_proxy and therefore you should not have to specify the `SERVICE_NAME` argument. " +
		"Simply supply the `FILE_NAME` found from the [files list](#files-list) command and the contents of the file, as well as the permissions string, will be printed to your console. " +
		"You can always store the file locally, applying the same permissions as those on the remote server, by specifying an output file with the `-o` flag. Here is a sample command\n\n" +
		"```\ndatica -E \"<your_env_alias>\" files download /etc/nginx/sites-enabled/mywebsite.com\n```",
	CmdFunc: func(settings *models.Settings) func(cmd *cli.Cmd) {
		return func(subCmd *cli.Cmd) {
			serviceName := subCmd.StringArg("SERVICE_NAME", "service_proxy", "The name of the service to download a file from")
			fileName := subCmd.StringArg("FILE_NAME", "", "The name of the service file from running \"datica files list\"")
			output := subCmd.StringOpt("o output", "", "The downloaded file will be saved to the given location with the same file permissions as it has on the remote host. If those file permissions cannot be applied, a warning will be printed and default 0644 permissions applied. If no output is specified, stdout is used.")
			force := subCmd.BoolOpt("f force", false, "If the specified output file already exists, automatically overwrite it")
			subCmd.Action = func() {
				if _, err := auth.New(settings, prompts.New()).Signin(); err != nil {
					logrus.Fatal(err.Error())
				}
				if err := config.CheckRequiredAssociation(settings); err != nil {
					logrus.Fatal(err.Error())
				}
				err := CmdDownload(*serviceName, *fileName, *output, *force, New(settings), services.New(settings))
				if err != nil {
					logrus.Fatal(err.Error())
				}
			}
			subCmd.Spec = "[SERVICE_NAME] FILE_NAME [-o] [-f]"
		}
	},
}

var ListSubCmd = models.Command{
	Name:      "list",
	ShortHelp: "List all files available for a given service",
	LongHelp: "`files list` prints out a listing of all service files available for download. " +
		"Nearly all service files are stored on the service_proxy and therefore you should not have to specify the `SERVICE_NAME` argument. " +
		"Here is a sample command\n\n" +
		"```\ndatica -E \"<your_env_alias>\" files list\n```",
	CmdFunc: func(settings *models.Settings) func(cmd *cli.Cmd) {
		return func(subCmd *cli.Cmd) {
			svcName := subCmd.StringArg("SERVICE_NAME", "service_proxy", "The name of the service to list files for")
			subCmd.Action = func() {
				if _, err := auth.New(settings, prompts.New()).Signin(); err != nil {
					logrus.Fatal(err.Error())
				}
				if err := config.CheckRequiredAssociation(settings); err != nil {
					logrus.Fatal(err.Error())
				}
				err := CmdList(*svcName, New(settings), services.New(settings))
				if err != nil {
					logrus.Fatal(err.Error())
				}
			}
			subCmd.Spec = "[SERVICE_NAME]"
		}
	},
}

// IFiles
type IFiles interface {
	Create(svcID, filePath, name, mode string) (*models.ServiceFile, error)
	List(svcID string) (*[]models.ServiceFile, error)
	Retrieve(fileName string, svcID string) (*models.ServiceFile, error)
	Save(output string, force bool, file *models.ServiceFile) error
}

// SFiles is a concrete implementation of IFiles
type SFiles struct {
	Settings *models.Settings
}

// New generates a new instance of IFiles
func New(settings *models.Settings) IFiles {
	return &SFiles{
		Settings: settings,
	}
}
