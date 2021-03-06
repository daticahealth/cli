package invites

import (
	"fmt"

	"github.com/Sirupsen/logrus"
	"github.com/daticahealth/cli/models"
)

func CmdList(envName string, ii IInvites) error {
	invts, err := ii.List()
	if err != nil {
		return err
	}
	if invts == nil || len(*invts) == 0 {
		logrus.Printf("There are no pending invites for %s", envName)
		return nil
	}
	logrus.Printf("Pending invites for %s:", envName)
	for _, invite := range *invts {
		logrus.Printf("\t%s %s", invite.Email, invite.ID)
	}
	return nil
}

// List lists all pending invites for a given org.
func (i *SInvites) List() (*[]models.Invite, error) {
	headers := i.Settings.HTTPManager.GetHeaders(i.Settings.SessionToken, i.Settings.Version, i.Settings.Pod, i.Settings.UsersID)
	resp, statusCode, err := i.Settings.HTTPManager.Get(nil, fmt.Sprintf("%s%s/orgs/%s/invites", i.Settings.AuthHost, i.Settings.AuthHostVersion, i.Settings.OrgID), headers)
	if err != nil {
		return nil, err
	}
	var invites []models.Invite
	err = i.Settings.HTTPManager.ConvertResp(resp, statusCode, &invites)
	if err != nil {
		return nil, err
	}
	return &invites, nil
}

// ListOrgGroups lists all available groups for an organization and their members
func (i *SInvites) ListOrgGroups() (*[]models.Group, error) {
	headers := i.Settings.HTTPManager.GetHeaders(i.Settings.SessionToken, i.Settings.Version, i.Settings.Pod, i.Settings.UsersID)
	resp, statusCode, err := i.Settings.HTTPManager.Get(nil, fmt.Sprintf("%s%s/acls/%s/groups", i.Settings.AuthHost, i.Settings.AuthHostVersion, i.Settings.OrgID), headers)
	if err != nil {
		return nil, err
	}
	var groupWrapper models.GroupWrapper
	err = i.Settings.HTTPManager.ConvertResp(resp, statusCode, &groupWrapper)
	if err != nil {
		return nil, err
	}
	return groupWrapper.Groups, nil
}
