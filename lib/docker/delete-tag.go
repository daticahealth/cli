package docker

import (
	"fmt"
	"net/url"
)

// DeleteTag deletes a tag for a docker image.
func (d *SDocker) DeleteTag(imageName, tagName string) error {
	headers := d.Settings.HTTPManager.GetHeaders(d.Settings.SessionToken, d.Settings.Version, d.Settings.Pod, d.Settings.UsersID)
	resp, statusCode, err := d.Settings.HTTPManager.Delete(nil, fmt.Sprintf("%s%s/environments/%s/images/%s/tags/%s", d.Settings.PaasHost, d.Settings.PaasHostVersion, d.Settings.EnvironmentID, url.PathEscape(url.PathEscape(imageName)), tagName), headers)
	if err != nil {
		return err
	}
	return d.Settings.HTTPManager.ConvertResp(resp, statusCode, nil)
}
