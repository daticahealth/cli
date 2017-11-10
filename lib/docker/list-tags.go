package docker

import (
	"fmt"
	"net/url"
)

// ListTags lists tags for a docker image.
func (d *SDocker) ListTags(imageName string) (*[]string, error) {
	headers := d.Settings.HTTPManager.GetHeaders(d.Settings.SessionToken, d.Settings.Version, d.Settings.Pod, d.Settings.UsersID)
	resp, statusCode, err := d.Settings.HTTPManager.Get(nil, fmt.Sprintf("%s%s/environments/%s/images/%s/tags", d.Settings.PaasHost, d.Settings.PaasHostVersion, d.Settings.EnvironmentID, url.PathEscape(url.PathEscape(imageName))), headers)
	if err != nil {
		return nil, err
	}
	var tags []string
	err = d.Settings.HTTPManager.ConvertResp(resp, statusCode, &tags)
	if err != nil {
		return nil, err
	}
	return &tags, nil
}
