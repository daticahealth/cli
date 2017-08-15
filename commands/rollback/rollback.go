package rollback

import (
	"fmt"
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/daticahealth/cli/commands/releases"
	"github.com/daticahealth/cli/commands/services"
	"github.com/daticahealth/cli/config"
	"github.com/daticahealth/cli/lib/jobs"
)

func CmdRollback(svcName, releaseName string, ij jobs.IJobs, irs releases.IReleases, is services.IServices) error {
	if strings.ContainsAny(releaseName, config.InvalidChars) {
		return fmt.Errorf("Invalid release name. Names must not contain the following characters: %s", config.InvalidChars)
	}
	service, err := is.RetrieveByLabel(svcName)
	if err != nil {
		return err
	}
	if service == nil {
		return fmt.Errorf("Could not find a service with the label \"%s\". You can list services with the \"datica services list\" command.", svcName)
	}
	logrus.Printf("Rolling back %s to %s", svcName, releaseName)
	release, err := irs.Retrieve(releaseName, service.ID)
	if err != nil {
		return err
	}
	if release == nil {
		return fmt.Errorf("Could not find a release with the name \"%s\". You can list releases for this code service with the \"datica releases list %s\" command.", releaseName, svcName)
	}
	err = ij.DeployRelease(releaseName, service.ID)
	if err != nil {
		return err
	}
	logrus.Println("Rollback successful! Check the status with \"datica status\" and your logging dashboard for updates.")
	return nil
}
