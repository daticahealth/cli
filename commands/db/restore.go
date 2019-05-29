package db

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/Sirupsen/logrus"
	"github.com/daticahealth/cli/commands/services"
	"github.com/daticahealth/cli/lib/prompts"
	"github.com/daticahealth/cli/models"
)

func CmdRestore(databaseName, backupID, mongoDatabase string, skipConfirm bool, id IDb, ip prompts.IPrompts, is services.IServices) error {
	if !skipConfirm {
		err := ip.YesNo("A database restore will be performed immediately. All current data will be lost if not included in the specified backup. No backup will be taken beforehand - please do so now if you need to.", "Do you wish to proceed (y/n)? ")
		if err != nil {
			return err
		}
	}
	service, err := is.RetrieveByLabel(databaseName)
	if err != nil {
		return err
	}
	if service == nil {
		return fmt.Errorf("Could not find a service with the label \"%s\". You can list services with the \"datica services list\" command.", databaseName)
	}
	err = id.Restore(backupID, service, mongoDatabase)
	if err != nil {
		return err
	}
	logrus.Printf("Backup %s restored to %s", backupID, databaseName)
	return nil
}

// Restore a backup to the database.
func (d *SDb) Restore(backupID string, service *models.Service, mongoDatabase string) error {
	sj, err := d.Jobs.Retrieve(backupID, service.ID, false)
	if err != nil {
		return err
	}
	if sj.Type != "backup" || (sj.Status != "finished") {
		return errors.New("Only 'finished' 'backup' jobs may be restored")
	}

	restoreParams := map[string]string{}
	if mongoDatabase != "" {
		restoreParams["database"] = mongoDatabase
	}
	restoreParams["jobId"] = backupID

	headers := d.Settings.HTTPManager.GetHeaders(d.Settings.SessionToken, d.Settings.Version, d.Settings.Pod, d.Settings.UsersID)
	body, err := json.Marshal(restoreParams)
	if err != nil {
		return err
	}
	resp, statusCode, err := d.Settings.HTTPManager.Post(body, fmt.Sprintf("%s%s/environments/%s/services/%s/restore", d.Settings.PaasHost, d.Settings.PaasHostVersion, d.Settings.EnvironmentID, service.ID), headers)
	if err != nil {
		return err
	}
	var job models.Job
	err = d.Settings.HTTPManager.ConvertResp(resp, statusCode, &job)
	if err != nil {
		return err
	}
	status, err := d.Jobs.PollTillFinished(job.ID, service.ID)
	if err != nil {
		return err
	}
	job.Status = status
	logrus.Printf("\nEnded in status '%s'", job.Status)
	err = d.DumpLogs("restore", &job, service)
	if err != nil {
		return err
	}
	if job.Status != "finished" {
		return fmt.Errorf("Job finished with invalid status %s", job.Status)
	}
	return nil
}
