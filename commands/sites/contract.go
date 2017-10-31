package sites

import (
	"github.com/Sirupsen/logrus"
	"github.com/daticahealth/cli/commands/certs"
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
	Name:      "sites",
	ShortHelp: "Tasks for updating sites, including hostnames, SSL certificates, and private keys",
	LongHelp: "The `sites` command gives access to hostname and SSL certificate usage for public facing services. " +
		"`sites` are different from `certs` in that `sites` use an instance of a `cert` and are associated with a single service. " +
		"`certs` can be used by multiple sites. The sites command can not be run directly but has sub commands.",
	CmdFunc: func(settings *models.Settings) func(cmd *cli.Cmd) {
		return func(cmd *cli.Cmd) {
			cmd.CommandLong(CreateSubCmd.Name, CreateSubCmd.ShortHelp, CreateSubCmd.LongHelp, CreateSubCmd.CmdFunc(settings))
			cmd.CommandLong(ListSubCmd.Name, ListSubCmd.ShortHelp, ListSubCmd.LongHelp, ListSubCmd.CmdFunc(settings))
			cmd.CommandLong(RmSubCmd.Name, RmSubCmd.ShortHelp, RmSubCmd.LongHelp, RmSubCmd.CmdFunc(settings))
			cmd.CommandLong(ShowSubCmd.Name, ShowSubCmd.ShortHelp, ShowSubCmd.LongHelp, ShowSubCmd.CmdFunc(settings))
		}
	},
}

var CreateSubCmd = models.Command{
	Name:      "create",
	ShortHelp: "Create a new site linking it to an existing cert instance",
	LongHelp: "`sites create` allows you to create a site configuration that is tied to a single service. " +
		"To create a site, you must specify an existing cert made by the [certs create](#certs-create) command or use the \"-l\" flag to automatically create a Let's Encrypt certificate. " +
		"A site has three pieces of information: a name, the service it's tied to, and the cert instance it will use. " +
		"The name is the `server_name` that will be injected into this site's Nginx configuration file. " +
		"It is important that this site name match what URL your site will respond to. " +
		"If this is a bare domain, using `mysite.com` is sufficient. " +
		"If it should respond to the APEX domain and all subdomains, it should be named `.mysite.com` notice the leading `.`. " +
		"The service is a code service that will use this site configuration. " +
		"Lastly, the cert instance must be specified by the `CERT_NAME` argument used in the [certs create](#certs-create) command or by the \"-l\" flag indicating a new Let's Encrypt certificate should be created. " +
		"You can also set Nginx configuration values directly by specifying one of the above flags. " +
		"Specifying `--enable-cors` will add the following lines to your Nginx configuration\n\n" +
		"```\nadd_header 'Access-Control-Allow-Origin' '$http_origin' always;\n" +
		"add_header 'Access-Control-Allow-Credentials' 'true' always;\n" +
		"add_header 'Access-Control-Allow-Methods' 'GET, POST, OPTIONS, DELETE, PUT, HEAD, PATCH' always;\n" +
		"add_header 'Access-Control-Allow-Headers' 'DNT,Keep-Alive,User-Agent,X-Requested-With,If-Modified-Since,Cache-Control,Content-Type,Accept,Authorization' always;\n" +
		"add_header 'Access-Control-Max-Age' 1728000 always;\n" +
		"if ($request_method = 'OPTIONS') {\n" +
		"  return 204;\n" +
		"}\n```\n\n" +
		"Specifying `--enable-websockets` will add the following lines to your Nginx configuration\n\n" +
		"```\nproxy_http_version 1.1;\n" +
		"proxy_set_header Upgrade $http_upgrade;\n" +
		"proxy_set_header Connection \"upgrade\";\n```\n\n" +
		"Here are some sample commands\n\n" +
		"```\ndatica -E \"<your_env_name>\" sites create .mysite.com app01 wildcard_mysitecom\n" +
		"datica -E \"<your_env_name>\" sites create .mysite.com app01 wildcard_mysitecom --client-max-body-size 50 --enable-cors\n" +
		"datica -E \"<your_env_name>\" sites create app01.mysite.com app01 --lets-encrypt --enable-websockets\n```",
	CmdFunc: func(settings *models.Settings) func(cmd *cli.Cmd) {
		return func(subCmd *cli.Cmd) {
			name := subCmd.StringArg("SITE_NAME", "", "The name of the site to be created. This will be used in this site's nginx configuration file (e.g. \".example.com\")")
			serviceName := subCmd.StringArg("SERVICE_NAME", "", "The name of the service to add this site configuration to (e.g. 'app01')")
			certName := subCmd.StringArg("CERT_NAME", "", "The name of the cert created with the 'certs' command (e.g. \"star_example_com\")")
			clientMaxBodySize := subCmd.IntOpt("client-max-body-size", -1, "The 'client_max_body_size' nginx config specified in megabytes")
			proxyConnectTimeout := subCmd.IntOpt("proxy-connect-timeout", -1, "The 'proxy_connect_timeout' nginx config specified in seconds")
			proxyReadTimeout := subCmd.IntOpt("proxy-read-timeout", -1, "The 'proxy_read_timeout' nginx config specified in seconds")
			proxySendTimeout := subCmd.IntOpt("proxy-send-timeout", -1, "The 'proxy_send_timeout' nginx config specified in seconds")
			proxyUpstreamTimeout := subCmd.IntOpt("proxy-upstream-timeout", -1, "The 'proxy_next_upstream_timeout' nginx config specified in seconds")
			enableCORS := subCmd.BoolOpt("enable-cors", false, "Enable or disable all features related to full CORS support")
			enableWebSockets := subCmd.BoolOpt("enable-websockets", false, "Enable or disable all features related to full websockets support")
			letsEncrypt := subCmd.BoolOpt("l lets-encrypt", false, "Whether or not this site should create an auto-renewing Let's Encrypt certificate")
			subCmd.Action = func() {
				if _, err := auth.New(settings, prompts.New()).Signin(); err != nil {
					logrus.Fatal(err.Error())
				}
				if err := config.CheckRequiredAssociation(settings); err != nil {
					logrus.Fatal(err.Error())
				}
				err := CmdCreate(*name, *serviceName, *certName, *clientMaxBodySize, *proxyConnectTimeout, *proxyReadTimeout, *proxySendTimeout, *proxyUpstreamTimeout, *enableCORS, *enableWebSockets, *letsEncrypt, New(settings), certs.New(settings), services.New(settings))
				if err != nil {
					logrus.Fatal(err.Error())
				}
			}
			subCmd.Spec = "SITE_NAME SERVICE_NAME (CERT_NAME | -l) [--client-max-body-size] [--proxy-connect-timeout] [--proxy-read-timeout] [--proxy-send-timeout] [--proxy-upstream-timeout] [--enable-cors] [--enable-websockets]"
		}
	},
}

var ListSubCmd = models.Command{
	Name:      "list",
	ShortHelp: "List details for all site configurations",
	LongHelp: "`sites list` lists all sites for the given environment. " +
		"The names printed out can be used in the other sites commands. Here is a sample command\n\n" +
		"```\ndatica -E \"<your_env_name>\" sites list\n```",
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
	ShortHelp: "Remove a site configuration",
	LongHelp: "`sites rm` allows you to remove a site by name. " +
		"Since sites cannot be updated, if you want to change the name of a site, you must `rm` the site and then [create](#sites-create) it again. " +
		"If you simply need to update your SSL certificates, you can use the [certs update](#certs-update) command on the cert instance used by the site in question. " +
		"Here is a sample command\n\n" +
		"```\ndatica -E \"<your_env_name>\" sites rm mywebsite.com\n```",
	CmdFunc: func(settings *models.Settings) func(cmd *cli.Cmd) {
		return func(subCmd *cli.Cmd) {
			name := subCmd.StringArg("NAME", "", "The name of the site configuration to delete")
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
			subCmd.Spec = "NAME"
		}
	},
}

var ShowSubCmd = models.Command{
	Name:      "show",
	ShortHelp: "Shows the details for a given site",
	LongHelp: "`sites show` will print out detailed information for a single site. " +
		"The name of the site can be found from the [sites list](#sites-list) command. Here is a sample command\n\n" +
		"```\ndatica -E \"<your_env_name>\" sites show mywebsite.com\n```",
	CmdFunc: func(settings *models.Settings) func(cmd *cli.Cmd) {
		return func(subCmd *cli.Cmd) {
			name := subCmd.StringArg("NAME", "", "The name of the site configuration to show")
			subCmd.Action = func() {
				if _, err := auth.New(settings, prompts.New()).Signin(); err != nil {
					logrus.Fatal(err.Error())
				}
				if err := config.CheckRequiredAssociation(settings); err != nil {
					logrus.Fatal(err.Error())
				}
				err := CmdShow(*name, New(settings), services.New(settings))
				if err != nil {
					logrus.Fatal(err.Error())
				}
			}
			subCmd.Spec = "NAME"
		}
	},
}

// ISites
type ISites interface {
	Create(name, cert, upstreamServiceID, svcID string, siteValues map[string]interface{}) (*models.Site, error)
	List(svcID string) (*[]models.Site, error)
	Retrieve(siteID int, svcID string) (*models.Site, error)
	Rm(siteID int, svcID string) error
}

// SSites is a concrete implementation of ISites
type SSites struct {
	Settings *models.Settings
}

// New returns an instance of ISites
func New(settings *models.Settings) ISites {
	return &SSites{
		Settings: settings,
	}
}
