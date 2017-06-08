package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/Sirupsen/logrus"
	"github.com/daticahealth/cli/models"
	"github.com/mitchellh/go-homedir"
)

// SettingsPath is the location of the datica config file.
const (
	settingsFormatV1 = "v1"
	settingsFormatV2 = "v2"

	OldSettingsFile = ".catalyze"
	SettingsFile    = ".datica"
	currentFormat   = settingsFormatV2
)

// SettingsRetriever defines an interface for a class responsible for generating
// a settings object used for most commands in the CLI. Some examples might be
// for retrieving settings based on the settings file or generating a settings
// object based on a directly entered environment ID and service ID.
type SettingsRetriever interface {
	GetSettings(string, string, string, string, string, string, string, string, string) (*models.Settings, error)
}

// FileSettingsRetriever reads in data from the SettingsFile and generates a
// settings object.
type FileSettingsRetriever struct{}

// GetSettings returns a Settings object for the current context
func (s FileSettingsRetriever) GetSettings(envName, svcName, accountsHost, authHost, ignoreAuthHostVersion, paasHost, ignorePaasHostVersion, username, password string) (*models.Settings, error) {
	HomeDir, err := homedir.Dir()
	if err != nil {
		return nil, err
	}

	if _, err = os.Stat(filepath.Join(HomeDir, OldSettingsFile)); err == nil {
		logrus.Debugln("Migrating settings file from .catalyze to .datica")
		err = os.Rename(filepath.Join(HomeDir, OldSettingsFile), filepath.Join(HomeDir, SettingsFile))
		if err != nil {
			return nil, fmt.Errorf("Error encountered migrating the settings file from .catalyze to .datica: %s. To fix this, please run \"mv %s %s\".", err, filepath.Join(HomeDir, OldSettingsFile), filepath.Join(HomeDir, SettingsFile))
		}
	}

	file, err := os.Open(filepath.Join(HomeDir, SettingsFile))
	if os.IsNotExist(err) {
		file, err = os.Create(filepath.Join(HomeDir, SettingsFile))
	}
	defer file.Close()
	if err != nil {
		return nil, err
	}
	var settings models.Settings
	json.NewDecoder(file).Decode(&settings)
	if settings.Format != currentFormat {
		if settings.Format == "" {
			settings.Format = "v1"
		}
		file.Seek(0, 0)
		settings, err = migrateSettings(file, settings.Format, currentFormat)
		if err != nil {
			return nil, err
		}
	}
	if settings.Environments == nil {
		settings.Environments = make(map[string]models.AssociatedEnvV2)
	}

	// try and set the given env first, if it exists
	if envName != "" {
		SetGivenEnv(envName, &settings)
	}

	settings.AccountsHost = accountsHost
	settings.AuthHost = authHost
	settings.PaasHost = paasHost
	settings.Username = username
	settings.Password = password

	authHostVersion := os.Getenv(AuthHostVersionEnvVar)
	if authHostVersion == "" {
		authHostVersion = AuthHostVersion
	}
	settings.AuthHostVersion = authHostVersion

	paasHostVersion := os.Getenv(PaasHostVersionEnvVar)
	if paasHostVersion == "" {
		paasHostVersion = PaasHostVersion
	}
	settings.PaasHostVersion = paasHostVersion

	logrus.Debugf("Accounts Host: %s", accountsHost)
	logrus.Debugf("Auth Host: %s", authHost)
	logrus.Debugf("Paas Host: %s", paasHost)
	logrus.Debugf("Auth Host Version: %s", authHostVersion)
	logrus.Debugf("Paas Host Version: %s", paasHostVersion)
	logrus.Debugf("Environment ID: %s", settings.EnvironmentID)
	logrus.Debugf("Environment Name: %s", settings.EnvironmentName)
	logrus.Debugf("Pod: %s", settings.Pod)
	logrus.Debugf("Org ID: %s", settings.OrgID)

	settings.Version = VERSION
	return &settings, nil
}

func StoreEnvironments(envs *[]models.Environment, settings *models.Settings) {
	settings.Environments = map[string]models.AssociatedEnvV2{}
	for _, env := range *envs {
		settings.Environments[env.Name] = models.AssociatedEnvV2{
			EnvironmentID: env.ID,
			Name:          env.Name,
			Pod:           env.Pod,
			OrgID:         env.OrgID,
		}
	}
}

func migrateSettings(file *os.File, oldFormat, newFormat string) (models.Settings, error) {
	if oldFormat == "v1" {
		return migrateFromV1(file)
	}
	return models.Settings{}, fmt.Errorf("Invalid or corrupt settings file. Please fix the %s file in your home directory or contact Datica support", SettingsFile)
}

func migrateFromV1(file *os.File) (models.Settings, error) {
	logrus.Debugf("Migrating settings from %s to %s", settingsFormatV1, currentFormat)
	var oldSettings models.SettingsV1
	json.NewDecoder(file).Decode(&oldSettings)
	newSettings := models.Settings{
		PrivateKeyPath: oldSettings.PrivateKeyPath,
		SessionToken:   oldSettings.SessionToken,
		UsersID:        oldSettings.UsersID,
		Environments:   map[string]models.AssociatedEnvV2{},
		Pods:           oldSettings.Pods,
		PodCheck:       oldSettings.PodCheck,
		Format:         currentFormat,
	}
	for envName, env := range oldSettings.Environments {
		newSettings.Environments[envName] = models.AssociatedEnvV2{
			EnvironmentID: env.EnvironmentID,
			Name:          env.Name,
			Pod:           env.Pod,
			OrgID:         env.OrgID,
		}
	}
	return newSettings, nil
}

// SaveSettings persists the settings to disk
func SaveSettings(settings *models.Settings) error {
	HomeDir, err := homedir.Dir()
	if err != nil {
		return err
	}
	b, _ := json.Marshal(&settings)
	return ioutil.WriteFile(filepath.Join(HomeDir, SettingsFile), b, 0644)
}

// SetGivenEnv takes the given env name and finds it in the env list
// in the given settings object. It then populates the EnvironmentID and
// ServiceID on the settings object with appropriate values.
func SetGivenEnv(envName string, settings *models.Settings) {
	for eName, e := range settings.Environments {
		if eName == envName {
			settings.EnvironmentID = e.EnvironmentID
			settings.Pod = e.Pod
			settings.EnvironmentName = envName
			settings.OrgID = e.OrgID
			break
		}
	}
}

// defaultEnvPrompt asks the user when they dont have a default environment and
// aren't in an associated directory if they would like to proceed with the
// first environment found.
func defaultEnvPrompt(envName string) error {
	var answer string
	for {
		fmt.Printf("No environment was specified. Falling back to %s\n", envName)
		fmt.Print("Do you wish to proceed? (y/n) ")
		fmt.Scanln(&answer)
		fmt.Println("")
		if answer != "y" && answer != "n" {
			fmt.Printf("%s is not a valid option. Please enter 'y' or 'n'\n", answer)
		} else {
			break
		}
	}
	if answer == "n" {
		return errors.New("Exiting")
	}
	return nil
}

// CheckRequiredAssociation ensures if an association is required for a command to run,
// that an appropriate environment has been picked and values assigned to the
// given settings object before a command is run. This is intended to be called
// before every command.
func CheckRequiredAssociation(settings *models.Settings) error {
	if settings.EnvironmentID == "" {
		err := errors.New("No Datica environment has been associated. Run \"datica associate\" from a local git repo first")
		for _, e := range settings.Environments {
			err = defaultEnvPrompt(e.Name)
			if err == nil {
				SetGivenEnv(e.Name, settings)
			}
			break
		}
		return err
	}
	return nil
}
