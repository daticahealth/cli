package dockerimages

import (
	"errors"
	"sort"

	"github.com/Sirupsen/logrus"
	"github.com/daticahealth/cli/commands/environments"
	"github.com/daticahealth/cli/lib/docker"
)

func cmdDockerImageList(envID string, ie environments.IEnvironments, id docker.IDocker) error {
	env, err := ie.Retrieve(envID)
	if err != nil {
		return err
	}
	if !env.DockerRegistryEnabled {
		return errors.New("This environment does not have docker registry support enabled.")
	}
	images, err := id.ListImages()
	if err != nil {
		return err
	}
	if len(*images) == 0 {
		logrus.Println("No images found for this environment. Note that images will not be visible in this list until they have been deployed.")
	} else {
		logrus.Printf("Docker images for environment \"%s\" (id %s)", env.Name, env.ID)
		logrus.Println("")
		sort.Strings(*images)
		for _, image := range *images {
			logrus.Println(image)
		}
	}
	return nil
}
