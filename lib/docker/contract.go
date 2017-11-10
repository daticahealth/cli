package docker

import "github.com/daticahealth/cli/models"

// IDocker describes docker-related functionality
type IDocker interface {
	ListImages() (*[]string, error)
	ListTags(imageName string) (*[]string, error)
	DeleteTag(imageName, tagName string) error
}

// SDocker is a concrete implementation of IDocker
type SDocker struct {
	Settings *models.Settings
}

// New constructs an implementation of IDocker
func New(settings *models.Settings) IDocker {
	return &SDocker{Settings: settings}
}
