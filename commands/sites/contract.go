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
	LongHelp: "The <code>sites</code> command gives access to hostname and SSL certificate usage for public facing services. " +
		"<code>sites</code> are different from <code>certs</code> in that <code>sites</code> use an instance of a <code>cert</code> and are associated with a single service. " +
		"<code>certs</code> can be used by multiple sites. The sites command can not be run directly but has subcommands.",
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
	LongHelp: "<code>sites create</code> allows you to create a site configuration that is tied to a single service. " +
		"To create a site, you must specify an existing cert made by the certs create command or use the \"-l\" flag to automatically create a Let's Encrypt certificate. " +
		"A site has three pieces of information: a name, the service it's tied to, and the cert instance it will use. " +
		"The name is the <code>server_name</code> that will be injected into this site's Nginx configuration file. " +
		"It is important that this site name match what URL your site will respond to. " +
		"If this is a bare domain, using <code>mysite.com</code> is sufficient. " +
		"If it should respond to the APEX domain and all subdomains, it should be named <code>.mysite.com</code> notice the leading <code>.</code>. " +
		"The service is a code service that will use this site configuration. " +
		"Lastly, the cert instance must be specified by the <code>CERT_NAME</code> argument used in the certs create command or by the \"-l\" flag indicating a new Let's Encrypt certificate should be created. " +
		"You can also set Nginx configuration values directly by specifying one of the above flags. " +
		"Specifying <code>--enable-websockets</code> will add the following lines to your Nginx configuration\n\n" +
		"<pre>\nproxy_http_version 1.1;\n" +
		"proxy_set_header Upgrade $http_upgrade;\n" +
		"proxy_set_header Connection \"upgrade\";\n</pre>\n\n" +
		"Here are some sample commands\n\n" +
		"<pre>\ndatica -E \"<your_env_name>\" sites create .mysite.com app01 wildcard_mysitecom\n" +
		"datica -E \"<your_env_name>\" sites create .mysite.com app01 wildcard_mysitecom --client-max-body-size 50 --enable-cors=www.mysite.com,mysite.com,api.mysite.com\n" +
		"datica -E \"<your_env_name>\" sites create app01.mysite.com app01 --lets-encrypt --enable-websockets\n</pre>",
	CmdFunc: func(settings *models.Settings) func(cmd *cli.Cmd) {
		return func(subCmd *cli.Cmd) {
			name := subCmd.StringArg("SITE_NAME", "", "The name of the site to be created. This will be used in this site's nginx configuration file (e.g. \".example.com\")")
			serviceName := subCmd.StringArg("SERVICE_NAME", "", "The name of the service to add this site configuration to (e.g. 'app01')")
			certName := subCmd.StringArg("CERT_NAME", "", "The name of the cert created with the 'certs' command (e.g. \"star_example_com\")")
			downStream := subCmd.StringOpt("down-stream", "service_proxy", "The name of the down-stream service. Defaults to \"service_proxy\"")
			clientMaxBodySize := subCmd.IntOpt("client-max-body-size", -1, "The 'client_max_body_size' nginx config specified in megabytes")
			proxyConnectTimeout := subCmd.IntOpt("proxy-connect-timeout", -1, "The 'proxy_connect_timeout' nginx config specified in seconds")
			proxyReadTimeout := subCmd.IntOpt("proxy-read-timeout", -1, "The 'proxy_read_timeout' nginx config specified in seconds")
			proxySendTimeout := subCmd.IntOpt("proxy-send-timeout", -1, "The 'proxy_send_timeout' nginx config specified in seconds")
			proxyUpstreamTimeout := subCmd.IntOpt("proxy-upstream-timeout", -1, "The 'proxy_next_upstream_timeout' nginx config specified in seconds")
			enableCORS := subCmd.StringOpt("enable-cors", "", "Enable or disable CORS support for specific sites (in a comma-delimited list)")
			enableWebSockets := subCmd.BoolOpt("enable-websockets", false, "Enable or disable all features related to full websockets support")
			letsEncrypt := subCmd.BoolOpt("l lets-encrypt", false, "Whether or not this site should create an auto-renewing Let's Encrypt certificate")
			subCmd.Action = func() {
				if _, err := auth.New(settings, prompts.New()).Signin(); err != nil {
					logrus.Fatal(err.Error())
				}
				if err := config.CheckRequiredAssociation(settings); err != nil {
					logrus.Fatal(err.Error())
				}
				err := CmdCreate(*name, *serviceName, *certName, *downStream, *clientMaxBodySize, *proxyConnectTimeout, *proxyReadTimeout, *proxySendTimeout, *proxyUpstreamTimeout, *enableCORS, *enableWebSockets, *letsEncrypt, New(settings), certs.New(settings), services.New(settings))
				if err != nil {
					logrus.Fatal(err.Error())
				}
			}
			subCmd.Spec = "SITE_NAME SERVICE_NAME (CERT_NAME | -l) [--down-stream] [--client-max-body-size] [--proxy-connect-timeout] [--proxy-read-timeout] [--proxy-send-timeout] [--proxy-upstream-timeout] [--enable-cors] [--enable-websockets]"
		}
	},
}

var ListSubCmd = models.Command{
	Name:      "list",
	ShortHelp: "List details for all site configurations",
	LongHelp: "<code>sites list</code> lists all sites for the given environment. " +
		"The names printed out can be used in the other sites commands. Here is a sample command\n\n" +
		"<pre>\ndatica -E \"<your_env_name>\" sites list\n</pre>",
	CmdFunc: func(settings *models.Settings) func(cmd *cli.Cmd) {
		return func(subCmd *cli.Cmd) {
			downStream := subCmd.StringOpt("down-stream", "service_proxy", "The name of the down-stream service. Defaults to \"service_proxy\"")
			subCmd.Action = func() {
				if _, err := auth.New(settings, prompts.New()).Signin(); err != nil {
					logrus.Fatal(err.Error())
				}
				if err := config.CheckRequiredAssociation(settings); err != nil {
					logrus.Fatal(err.Error())
				}
				err := CmdList(New(settings), services.New(settings), *downStream)
				if err != nil {
					logrus.Fatal(err.Error())
				}
			}
			subCmd.Spec = "[--down-stream]"
		}
	},
}

var RmSubCmd = models.Command{
	Name:      "rm",
	ShortHelp: "Remove a site configuration",
	LongHelp: "<code>sites rm</code> allows you to remove a site by name. " +
		"Since sites cannot be updated, if you want to change the name of a site, you must <code>rm</code> the site and then create it again. " +
		"If you simply need to update your SSL certificates, you can use the certs update command on the cert instance used by the site in question. " +
		"Here is a sample command\n\n" +
		"<pre>\ndatica -E \"<your_env_name>\" sites rm mywebsite.com\n</pre>",
	CmdFunc: func(settings *models.Settings) func(cmd *cli.Cmd) {
		return func(subCmd *cli.Cmd) {
			name := subCmd.StringArg("NAME", "", "The name of the site configuration to delete")
			downStream := subCmd.StringOpt("down-stream", "service_proxy", "The name of the down-stream service. Defaults to \"service_proxy\"")
			subCmd.Action = func() {
				if _, err := auth.New(settings, prompts.New()).Signin(); err != nil {
					logrus.Fatal(err.Error())
				}
				if err := config.CheckRequiredAssociation(settings); err != nil {
					logrus.Fatal(err.Error())
				}
				err := CmdRm(*name, New(settings), services.New(settings), *downStream)
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
	LongHelp: "<code>sites show</code> will print out detailed information for a single site. " +
		"The name of the site can be found from the sites list command. Here is a sample command\n\n" +
		"<pre>\ndatica -E \"<your_env_name>\" sites show mywebsite.com\n</pre>",
	CmdFunc: func(settings *models.Settings) func(cmd *cli.Cmd) {
		return func(subCmd *cli.Cmd) {
			name := subCmd.StringArg("NAME", "", "The name of the site configuration to show")
			downStream := subCmd.StringOpt("down-stream", "service_proxy", "The name of the down-stream service. Defaults to \"service_proxy\"")
			subCmd.Action = func() {
				if _, err := auth.New(settings, prompts.New()).Signin(); err != nil {
					logrus.Fatal(err.Error())
				}
				if err := config.CheckRequiredAssociation(settings); err != nil {
					logrus.Fatal(err.Error())
				}
				err := CmdShow(*name, New(settings), services.New(settings), *downStream)
				if err != nil {
					logrus.Fatal(err.Error())
				}
			}
			subCmd.Spec = "NAME [--down-stream]"
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
