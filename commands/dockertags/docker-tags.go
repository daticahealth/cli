package dockertags

import (
	"sort"

	"github.com/Sirupsen/logrus"
	"github.com/daticahealth/cli/lib/docker"
)

func cmdDockerTagList(id docker.IDocker, image string) error {
	tags, err := id.ListTags(image)
	if err != nil {
		return err
	}
	logrus.Printf("Docker tags for image \"%s\"", image)
	logrus.Println("")
	sort.Strings(*tags)
	for _, tag := range *tags {
		logrus.Println(tag)
	}
	return nil
}

func cmdDockerTagDelete(id docker.IDocker, image, tag string) error {
	err := id.DeleteTag(image, tag)
	if err == nil {
		logrus.Println("Tag deleted successfully.")
	}
	return err
}
