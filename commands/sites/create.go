package sites

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/daticahealth/cli/commands/certs"
	"github.com/daticahealth/cli/commands/services"
	"github.com/daticahealth/cli/models"
)

func CmdCreate(name, serviceName, certName, downStreamService string, clientMaxBodySize, proxyConnectTimeout, proxyReadTimeout, proxySendTimeout, proxyUpstreamTimeout int, enableCORS string, enableWebSockets, letsEncrypt bool, is ISites, ic certs.ICerts, iservices services.IServices) error {
	upstreamService, err := iservices.RetrieveByLabel(serviceName)
	if err != nil {
		return err
	}
	if upstreamService == nil {
		return fmt.Errorf("Could not find a service with the label \"%s\". You can list services with the \"datica services list\" command.", serviceName)
	}

	serviceProxy, err := iservices.RetrieveByLabel(downStreamService)
	if err != nil {
		return err
	}

	if letsEncrypt {
		err = ic.CreateLetsEncrypt(name, serviceProxy.ID)
		if err != nil {
			return err
		}
		certName = name
	}

	site, err := is.Create(name, certName, upstreamService.ID, serviceProxy.ID, generateSiteValues(clientMaxBodySize, proxyConnectTimeout, proxyReadTimeout, proxySendTimeout, proxyUpstreamTimeout, enableCORS, enableWebSockets))
	if err != nil {
		return err
	}
	logrus.Printf("Created '%s'", site.Name)
	logrus.Println("To make your site go live, you must redeploy your service proxy with the \"datica redeploy service_proxy\" command")
	return nil
}

func (s *SSites) Create(name, cert, upstreamServiceID, svcID string, siteValues map[string]interface{}) (*models.Site, error) {
	site := models.Site{
		Name:            name,
		Cert:            cert,
		UpstreamService: upstreamServiceID,
		SiteValues:      siteValues,
	}
	b, err := json.Marshal(site)
	if err != nil {
		return nil, err
	}
	headers := s.Settings.HTTPManager.GetHeaders(s.Settings.SessionToken, s.Settings.Version, s.Settings.Pod, s.Settings.UsersID)
	resp, statusCode, err := s.Settings.HTTPManager.Post(b, fmt.Sprintf("%s%s/environments/%s/services/%s/sites", s.Settings.PaasHost, s.Settings.PaasHostVersion, s.Settings.EnvironmentID, svcID), headers)
	if err != nil {
		return nil, err
	}
	var createdSite models.Site
	err = s.Settings.HTTPManager.ConvertResp(resp, statusCode, &createdSite)
	if err != nil {
		return nil, err
	}
	return &createdSite, nil
}

func generateSiteValues(clientMaxBodySize, proxyConnectTimeout, proxyReadTimeout, proxySendTimeout, proxyUpstreamTimeout int, enableCORS string, enableWebSockets bool) map[string]interface{} {
	siteValues := map[string]interface{}{}
	if clientMaxBodySize >= 0 {
		siteValues["clientMaxBodySize"] = fmt.Sprintf("%dm", clientMaxBodySize)
	}
	if proxyConnectTimeout >= 0 {
		siteValues["proxyConnectTimeout"] = fmt.Sprintf("%ds", proxyConnectTimeout)
	}
	if proxyReadTimeout >= 0 {
		siteValues["proxyReadTimeout"] = fmt.Sprintf("%ds", proxyReadTimeout)
	}
	if proxySendTimeout >= 0 {
		siteValues["proxySendTimeout"] = fmt.Sprintf("%ds", proxySendTimeout)
	}
	if proxyUpstreamTimeout >= 0 {
		siteValues["proxyUpstreamTimeout"] = fmt.Sprintf("%ds", proxyUpstreamTimeout)
	}
	if len(enableCORS) > 0 {
		var sites []string
		for _, site := range strings.Split(enableCORS, ",") {
			site = strings.TrimSpace(site)
			if len(site) > 0 {
				sites = append(sites, strings.Replace(site, ".", "\\.", -1))
			}
		}
		siteValues["enableCORSSites"] = strings.Join(sites, "|")
	}
	if enableWebSockets {
		siteValues["enableWebSockets"] = true
	}
	return siteValues
}
