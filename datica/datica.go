package datica

import (
	"fmt"
	"os"
	"runtime"
	"strconv"
	"time"

	"github.com/daticahealth/cli/commands/certs"
	"github.com/daticahealth/cli/commands/clear"
	"github.com/daticahealth/cli/commands/console"
	"github.com/daticahealth/cli/commands/db"
	"github.com/daticahealth/cli/commands/deploykeys"
	"github.com/daticahealth/cli/commands/domain"
	"github.com/daticahealth/cli/commands/environments"
	"github.com/daticahealth/cli/commands/files"
	"github.com/daticahealth/cli/commands/git"
	initcmd "github.com/daticahealth/cli/commands/init"
	"github.com/daticahealth/cli/commands/invites"
	"github.com/daticahealth/cli/commands/keys"
	"github.com/daticahealth/cli/commands/logout"
	"github.com/daticahealth/cli/commands/logs"
	"github.com/daticahealth/cli/commands/maintenance"
	"github.com/daticahealth/cli/commands/metrics"
	"github.com/daticahealth/cli/commands/rake"
	"github.com/daticahealth/cli/commands/redeploy"
	"github.com/daticahealth/cli/commands/releases"
	"github.com/daticahealth/cli/commands/rollback"
	"github.com/daticahealth/cli/commands/services"
	"github.com/daticahealth/cli/commands/sites"
	"github.com/daticahealth/cli/commands/ssl"
	"github.com/daticahealth/cli/commands/status"
	"github.com/daticahealth/cli/commands/supportids"
	"github.com/daticahealth/cli/commands/update"
	"github.com/daticahealth/cli/commands/users"
	"github.com/daticahealth/cli/commands/vars"
	"github.com/daticahealth/cli/commands/version"
	"github.com/daticahealth/cli/commands/whoami"
	"github.com/daticahealth/cli/commands/worker"

	"github.com/daticahealth/cli/config"
	"github.com/daticahealth/cli/models"

	"github.com/daticahealth/cli/lib/httpclient"
	"github.com/daticahealth/cli/lib/pods"
	"github.com/daticahealth/cli/lib/updater"

	"github.com/Sirupsen/logrus"
	"github.com/jault3/mow.cli"
)

type simpleLogger struct{}

func (s *simpleLogger) Format(entry *logrus.Entry) ([]byte, error) {
	levelString := fmt.Sprintf("[%s] ", entry.Level)
	levelPrefix := ""
	levelSuffix := ""
	if entry.Level == logrus.InfoLevel {
		levelString = ""
	}
	if runtime.GOOS != "windows" {
		if entry.Level == logrus.WarnLevel {
			// [33m = yellow
			levelPrefix = "\033[33m\033[1m"
			levelSuffix = "\033[0m"
		} else if entry.Level == logrus.PanicLevel || entry.Level == logrus.FatalLevel || entry.Level == logrus.ErrorLevel {
			// [31m = red
			levelPrefix = "\033[31m\033[1m"
			levelSuffix = "\033[0m"
		}
	}

	l := fmt.Sprintf("%s%s%s%s\n", levelPrefix, levelString, levelSuffix, entry.Message)
	return []byte(l), nil
}

// Run runs the Datica CLI
func Run() {
	InitLogrus()

	if !config.Beta {
		if updater.AutoUpdater != nil {
			updater.AutoUpdater.BackgroundRun()
		}
	}

	var app = cli.App("datica", fmt.Sprintf("Datica CLI. Version %s", config.VERSION))
	settings := &models.Settings{}
	InitGlobalOpts(app, settings)
	InitCLI(app, settings)

	app.Run(os.Args)
}

func InitGlobalOpts(app *cli.Cli, settings *models.Settings) {
	accountsHost := os.Getenv(config.AccountsHostEnvVar)
	if accountsHost == "" {
		accountsHost = config.AccountsHost
	}
	authHost := os.Getenv(config.AuthHostEnvVar)
	if authHost == "" {
		authHost = config.AuthHost
	}
	paasHost := os.Getenv(config.PaasHostEnvVar)
	if paasHost == "" {
		paasHost = config.PaasHost
	}
	email := app.String(cli.StringOpt{
		Name:      "email",
		Desc:      "Datica Email",
		EnvVar:    config.DaticaEmailEnvVar,
		HideValue: true,
	})
	username := app.String(cli.StringOpt{
		Name:      "U username",
		Desc:      "[DEPRECATED] Datica Username (This flag is deprecated. Please use --email instead)",
		EnvVar:    config.DaticaUsernameEnvVarDeprecated,
		HideValue: true,
	})
	password := app.String(cli.StringOpt{
		Name:      "P password",
		Desc:      "Datica Password",
		EnvVar:    config.DaticaPasswordEnvVar,
		HideValue: true,
	})
	givenEnvName := app.String(cli.StringOpt{
		Name:      "E env",
		Desc:      "The local alias of the environment in which this command will be run",
		EnvVar:    config.DaticaEnvironmentEnvVar,
		HideValue: true,
	})
	if loggingLevel := os.Getenv(config.LogLevelEnvVar); loggingLevel != "" {
		if lvl, err := logrus.ParseLevel(loggingLevel); err == nil {
			logrus.SetLevel(lvl)
		}
	}

	app.Before = func() {
		if *email == "" {
			*email = *username
			if *email != "" {
				logrus.Warnf("You are using a deprecated environment variable %s. Please use %s instead. Support for %s will be removed soon.", config.DaticaUsernameEnvVarDeprecated, config.DaticaEmailEnvVar, config.DaticaUsernameEnvVarDeprecated)
			}
		}
		if config.Beta {
			logrus.Println("This is a BETA release. Please contact Datica Support at https://datica.com/support with any issues.")
		}
		r := config.FileSettingsRetriever{}
		s, err := r.GetSettings(*givenEnvName, "", accountsHost, authHost, "", paasHost, "", *email, *password)
		if err != nil {
			logrus.Println(err)
			cli.Exit(1)
		}
		*settings = *s
		skip, _ := strconv.ParseBool(os.Getenv(config.SkipVerifyEnvVar))
		settings.HTTPManager = httpclient.NewTLSHTTPManager(skip)
		logrus.Debugf("%+v", settings)

		if settings.Pods == nil || len(*settings.Pods) == 0 || settings.PodCheck < time.Now().Unix() {
			settings.PodCheck = time.Now().Unix() + 86400
			p := pods.New(settings)
			pods, err := p.List()
			if err == nil {
				settings.Pods = pods
				logrus.Debugf("%+v", settings.Pods)
			} else {
				settings.Pods = &[]models.Pod{}
				logrus.Debugf("Error listing pods: %s", err.Error())
			}
		}

		if settings.EnvironmentID == "" && *givenEnvName != "" {
			envs, errs := environments.New(settings).List()
			if errs != nil && len(errs) > 0 {
				logrus.Debugf("Error listing environments: %+v", errs)
			}
			config.StoreEnvironments(envs, settings)
			config.SetGivenEnv(*givenEnvName, settings)
		}
	}
	app.After = func() {
		config.SaveSettings(settings)
	}

	betaString := ""
	if config.Beta {
		betaString = "-BETA"
	}
	versionString := fmt.Sprintf("version %s%s %s", config.VERSION, betaString, config.ArchString())
	logrus.Debugln(versionString)
	app.Version("v version", versionString)
}

// InitLogrus sets up logrus for the correctly formatted log messages
func InitLogrus() {
	logrus.SetFormatter(&simpleLogger{})
	logrus.SetOutput(os.Stdout)
	logrus.SetLevel(config.LogLevel)
}

// InitCLI adds arguments and commands to the given cli instance
func InitCLI(app *cli.Cli, settings *models.Settings) {
	app.CommandLong(certs.Cmd.Name, certs.Cmd.ShortHelp, certs.Cmd.LongHelp, certs.Cmd.CmdFunc(settings))
	app.CommandLong(clear.Cmd.Name, clear.Cmd.ShortHelp, clear.Cmd.LongHelp, clear.Cmd.CmdFunc(settings))
	app.CommandLong(console.Cmd.Name, console.Cmd.ShortHelp, console.Cmd.LongHelp, console.Cmd.CmdFunc(settings))
	app.CommandLong(db.Cmd.Name, db.Cmd.ShortHelp, db.Cmd.LongHelp, db.Cmd.CmdFunc(settings))
	app.CommandLong(deploykeys.Cmd.Name, deploykeys.Cmd.ShortHelp, deploykeys.Cmd.LongHelp, deploykeys.Cmd.CmdFunc(settings))
	app.CommandLong(domain.Cmd.Name, domain.Cmd.ShortHelp, domain.Cmd.LongHelp, domain.Cmd.CmdFunc(settings))
	app.CommandLong(environments.Cmd.Name, environments.Cmd.ShortHelp, environments.Cmd.LongHelp, environments.Cmd.CmdFunc(settings))
	app.CommandLong(files.Cmd.Name, files.Cmd.ShortHelp, files.Cmd.LongHelp, files.Cmd.CmdFunc(settings))
	app.CommandLong(git.Cmd.Name, git.Cmd.ShortHelp, git.Cmd.LongHelp, git.Cmd.CmdFunc(settings))
	app.CommandLong(initcmd.Cmd.Name, initcmd.Cmd.ShortHelp, initcmd.Cmd.LongHelp, initcmd.Cmd.CmdFunc(settings))
	app.CommandLong(invites.Cmd.Name, invites.Cmd.ShortHelp, invites.Cmd.LongHelp, invites.Cmd.CmdFunc(settings))
	app.CommandLong(keys.Cmd.Name, keys.Cmd.ShortHelp, keys.Cmd.LongHelp, keys.Cmd.CmdFunc(settings))
	app.CommandLong(logout.Cmd.Name, logout.Cmd.ShortHelp, logout.Cmd.LongHelp, logout.Cmd.CmdFunc(settings))
	app.CommandLong(logs.Cmd.Name, logs.Cmd.ShortHelp, logs.Cmd.LongHelp, logs.Cmd.CmdFunc(settings))
	app.CommandLong(maintenance.Cmd.Name, maintenance.Cmd.ShortHelp, maintenance.Cmd.LongHelp, maintenance.Cmd.CmdFunc(settings))
	app.CommandLong(metrics.Cmd.Name, metrics.Cmd.ShortHelp, metrics.Cmd.LongHelp, metrics.Cmd.CmdFunc(settings))
	app.CommandLong(rake.Cmd.Name, rake.Cmd.ShortHelp, rake.Cmd.LongHelp, rake.Cmd.CmdFunc(settings))
	app.CommandLong(redeploy.Cmd.Name, redeploy.Cmd.ShortHelp, redeploy.Cmd.LongHelp, redeploy.Cmd.CmdFunc(settings))
	app.CommandLong(releases.Cmd.Name, releases.Cmd.ShortHelp, releases.Cmd.LongHelp, releases.Cmd.CmdFunc(settings))
	app.CommandLong(rollback.Cmd.Name, rollback.Cmd.ShortHelp, rollback.Cmd.LongHelp, rollback.Cmd.CmdFunc(settings))
	app.CommandLong(services.Cmd.Name, services.Cmd.ShortHelp, services.Cmd.LongHelp, services.Cmd.CmdFunc(settings))
	app.CommandLong(sites.Cmd.Name, sites.Cmd.ShortHelp, sites.Cmd.LongHelp, sites.Cmd.CmdFunc(settings))
	app.CommandLong(ssl.Cmd.Name, ssl.Cmd.ShortHelp, ssl.Cmd.LongHelp, ssl.Cmd.CmdFunc(settings))
	app.CommandLong(status.Cmd.Name, status.Cmd.ShortHelp, status.Cmd.LongHelp, status.Cmd.CmdFunc(settings))
	app.CommandLong(supportids.Cmd.Name, supportids.Cmd.ShortHelp, supportids.Cmd.LongHelp, supportids.Cmd.CmdFunc(settings))
	if !config.Beta {
		app.CommandLong(update.Cmd.Name, update.Cmd.ShortHelp, update.Cmd.LongHelp, update.Cmd.CmdFunc(settings))
	}
	app.CommandLong(users.Cmd.Name, users.Cmd.ShortHelp, users.Cmd.LongHelp, users.Cmd.CmdFunc(settings))
	app.CommandLong(vars.Cmd.Name, vars.Cmd.ShortHelp, vars.Cmd.LongHelp, vars.Cmd.CmdFunc(settings))
	app.CommandLong(version.Cmd.Name, version.Cmd.ShortHelp, version.Cmd.LongHelp, version.Cmd.CmdFunc(settings))
	app.CommandLong(whoami.Cmd.Name, whoami.Cmd.ShortHelp, whoami.Cmd.LongHelp, whoami.Cmd.CmdFunc(settings))
	app.CommandLong(worker.Cmd.Name, worker.Cmd.ShortHelp, worker.Cmd.LongHelp, worker.Cmd.CmdFunc(settings))
}
