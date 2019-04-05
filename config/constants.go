package config

import (
	"runtime"

	"github.com/Sirupsen/logrus"
)

const (
	// VERSION is the current cli version
	VERSION = "4.3.2"
	// Beta determines whether or not this is a beta build of the CLI
	Beta = false
	// AccountsHost is the production accounts URL
	AccountsHost = "https://product.datica.com"
	// AuthHost is the production auth URL
	AuthHost = "https://auth.datica.com"
	// AuthHostVersion is the version path for the auth host
	AuthHostVersion = ""
	// PaasHost is the production PaaS URL
	PaasHost = "https://paas-api.catalyze.io"
	// PaasHostVersion is the version path for the PaaS host
	PaasHostVersion = ""
	// LogLevel is the amount of logging to enable
	LogLevel = logrus.InfoLevel
	// JobPollTime is the amount of time in seconds to wait between polls for a job status
	JobPollTime = 5
	// LogPollTime is the amount of time in seconds to wait between polls for new logs
	LogPollTime = 3

	// AccountsHostEnvVar is the env variable used to override AccountsHost
	AccountsHostEnvVar = "ACCOUNTS_HOST"
	// AuthHostEnvVar is the env variable used to override AuthHost
	AuthHostEnvVar = "AUTH_HOST"
	// PaasHostEnvVar is the env variable used to override PaasHost
	PaasHostEnvVar = "PAAS_HOST"
	// AuthHostVersionEnvVar is the env variable used to override AuthHostVersion
	AuthHostVersionEnvVar = "AUTH_HOST_VERSION"
	// PaasHostVersionEnvVar is the env variable used to override PaasHostVersion
	PaasHostVersionEnvVar = "PAAS_HOST_VERSION"
	// DaticaEmailEnvVar is the env variable used to override the email
	DaticaEmailEnvVar = "DATICA_EMAIL"
	// DaticaPasswordEnvVar is the env variable used to override the passowrd
	DaticaPasswordEnvVar = "DATICA_PASSWORD"
	// DaticaEnvironmentEnvVar is the env variable used to override the environment used in the current command
	DaticaEnvironmentEnvVar = "DATICA_ENV"
	// LogLevelEnvVar is the env variable used to override the logging level used
	LogLevelEnvVar = "DATICA_LOG_LEVEL"
	// SkipVerifyEnvVar is the env variable used to accept invalid SSL certificates
	SkipVerifyEnvVar = "SKIP_VERIFY"
	// DaticaConfigFile points the CLI at a .datica file
	DaticaConfigFile = "DATICA_CONFIG_FILE"

	// DaticaUsernameEnvVarDeprecated is the deprecated env variable used to override the username
	DaticaUsernameEnvVarDeprecated = "DATICA_USERNAME"

	// InvalidChars is a string containing all invalid characters for naming
	InvalidChars = "/?%"
)

// ArchString translates the current architecture into an easier to read value.
// amd64 becomes 64-bit, 386 becomes 32-bit, etc.
func ArchString() string {
	archString := "other"
	switch runtime.GOARCH {
	case "386":
		archString = "32-bit"
	case "amd64":
		archString = "64-bit"
	case "arm":
		archString = "arm"
	}
	return archString
}
