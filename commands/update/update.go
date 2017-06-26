package update

import (
	"fmt"
	"runtime"

	"github.com/Sirupsen/logrus"

	"github.com/daticahealth/cli/commands/environments"
	"github.com/daticahealth/cli/config"
	"github.com/daticahealth/cli/lib/pods"
	"github.com/daticahealth/cli/lib/updater"
)

func CmdUpdate(iu IUpdate) error {
	logrus.Println("Checking for available updates...")
	needsUpdate, err := iu.Check()
	if err != nil {
		return err
	}
	// check if we can overwrite exe
	if needsUpdate && (runtime.GOOS == "linux" || runtime.GOOS == "darwin") {
		err = verifyExeDirWriteable()
		if err != nil {
			return err
		}
	}
	if !needsUpdate {
		logrus.Println("You are already running the latest version of the Datica CLI")
		return nil
	}
	logrus.Printf("Version %s is available. Updating your CLI...", updater.AutoUpdater.Info.Version)
	err = iu.Update()
	if err != nil {
		return err
	}
	iu.UpdatePods()
	iu.UpdateEnvironments()
	logrus.Printf("Your CLI has been updated to version %s", updater.AutoUpdater.Info.Version)
	return nil
}

func exeGenericError() error {
	return fmt.Errorf("There was an error trying to find where your CLI is on your system. You may need to manually update your CLI")
}

func (u *SUpdate) Check() (bool, error) {
	updater.AutoUpdater.FetchInfo()
	if updater.AutoUpdater.CurrentVersion >= updater.AutoUpdater.Info.Version {
		return false, nil
	}
	return true, nil
}

// Update updates the  CLI if a new update is available.
func (u *SUpdate) Update() error {
	updater.AutoUpdater.ForcedUpgrade()
	return nil
}

// UpdatePods retrieves the latest list of pods and refreshes the local cache. If an error occurs,
// the local cache is unchanged.
func (u *SUpdate) UpdatePods() {
	logrus.Debugf("Updating cached pods...")
	p := pods.New(u.Settings)
	pods, err := p.List()
	if err == nil {
		u.Settings.Pods = pods
	} else {
		logrus.Debugf("Error listing pods: %s", err.Error())
	}
}

// UpdateEnvironments retrieves all environments visible to the current user and stores them in the local cache.
func (u *SUpdate) UpdateEnvironments() {
	envs, errs := environments.New(u.Settings).List()
	if errs != nil && len(errs) > 0 {
		logrus.Debugf("Error listing environments: %+v", errs)
	}
	config.StoreEnvironments(envs, u.Settings)
}
