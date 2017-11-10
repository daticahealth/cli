package docker

import "fmt"

// ListImages lists docker images for an environment.
func (d *SDocker) ListImages() (*[]string, error) {
	headers := d.Settings.HTTPManager.GetHeaders(d.Settings.SessionToken, d.Settings.Version, d.Settings.Pod, d.Settings.UsersID)
	resp, statusCode, err := d.Settings.HTTPManager.Get(nil, fmt.Sprintf("%s%s/environments/%s/images", d.Settings.PaasHost, d.Settings.PaasHostVersion, d.Settings.EnvironmentID), headers)
	if err != nil {
		return nil, err
	}
	var images []string
	err = d.Settings.HTTPManager.ConvertResp(resp, statusCode, &images)
	if err != nil {
		return nil, err
	}
	return &images, nil
}
