package db

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/daticahealth/cli/commands/services"
	"github.com/daticahealth/cli/lib/crypto"
	"github.com/daticahealth/cli/lib/jobs"
	"github.com/daticahealth/cli/lib/prompts"
	"github.com/daticahealth/cli/lib/transfer"
	"github.com/daticahealth/cli/models"
)

func CmdImport(databaseName, filePath, mongoCollection, mongoDatabase string, skipBackup bool, id IDb, ip prompts.IPrompts, is services.IServices, ij jobs.IJobs) error {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("A file does not exist at path '%s'", filePath)
	}
	service, err := is.RetrieveByLabel(databaseName)
	if err != nil {
		return err
	}
	if service == nil {
		return fmt.Errorf("Could not find a service with the label \"%s\". You can list services with the \"datica services list\" command.", databaseName)
	}
	key := make([]byte, crypto.KeySize)
	iv := make([]byte, crypto.IVSize)
	rand.Read(key)
	rand.Read(iv)
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()
	fi, err := file.Stat()
	if err != nil {
		return err
	}
	encryptFileReader, err := id.NewEncryptReader(file, key, iv)
	if err != nil {
		return err
	}
	uploadSize := encryptFileReader.CalculateTotalSize(int(fi.Size()))
	fiveTB := transfer.TB * 5
	if transfer.ByteSize(uploadSize) > fiveTB {
		return fmt.Errorf("the encrypted size of %s exceeds the maximum upload size of %s", filePath, fiveTB)
	}
	rt := transfer.NewReaderTransfer(encryptFileReader, uploadSize)
	if !skipBackup {
		logrus.Printf("Backing up \"%s\" before performing the import", databaseName)
		job, err := id.Backup(service)
		if err != nil {
			return err
		}
		logrus.Printf("Backup started (job ID = %s)", job.ID)

		// all because logrus treats print, println, and printf the same
		logrus.Println("Polling until backup finishes.")
		if job.IsSnapshotBackup != nil && *job.IsSnapshotBackup {
			logrus.Printf("This is a snapshot backup, it may be a while before this backup shows up in the \"datica db list %s\" command.", databaseName)
			err = ij.WaitToAppear(job.ID, service.ID)
			if err != nil {
				return err
			}
		}
		status, err := ij.PollTillFinished(job.ID, service.ID)
		if err != nil {
			return err
		}
		job.Status = status
		logrus.Printf("Ended in status '%s'", job.Status)
		err = id.DumpLogs("backup", job, service)
		if err != nil {
			return err
		}
		if job.Status != "finished" {
			return fmt.Errorf("Job finished with invalid status %s", job.Status)
		}
	} else {
		err := ip.YesNo("", "Are you sure you want to import data into your database without backing it up first? (y/n) ")
		if err != nil {
			return err
		}
	}
	logrus.Printf("Importing '%s' into %s (ID = %s)", filePath, databaseName, service.ID)
	job, err := id.Import(rt, key, iv, mongoCollection, mongoDatabase, service)
	if err != nil {
		return err
	}
	// all because logrus treats print, println, and printf the same
	logrus.StandardLogger().Out.Write([]byte(fmt.Sprintf("Processing import (job ID = %s).", job.ID)))

	status, err := ij.PollTillFinished(job.ID, service.ID)
	if err != nil {
		return err
	}
	job.Status = status
	logrus.Printf("\nImport complete (end status = '%s')", job.Status)
	err = id.DumpLogs("restore", job, service)
	if err != nil {
		return err
	}
	if job.Status != "finished" {
		return fmt.Errorf("Finished with invalid status %s", job.Status)
	}
	return nil
}

// Import imports data into a database service. The import is accomplished
// by encrypting the file locally, requesting a location that it can be uploaded
// to, then uploads the file. Once uploaded an automated service processes the
// file and acts according to the given parameters.
//
// The type of file that should be imported depends on the database. For
// PostgreSQL and MySQL, this should be a single `.sql` file. For Mongo, this
// should be a single tar'ed, gzipped archive (`.tar.gz`) of the database dump
// that you want to import.
func (d *SDb) Import(rt *transfer.ReaderTransfer, key, iv []byte, mongoCollection, mongoDatabase string, service *models.Service) (*models.Job, error) {
	options := map[string]string{}
	if mongoCollection != "" {
		options["databaseCollection"] = mongoCollection
	}
	if mongoDatabase != "" {
		options["database"] = mongoDatabase
	}

	var uploadInfo *models.MultipartUploadInfo
	var err error
	for attempt := 0; attempt < 5; attempt++ {
		uploadInfo, err = d.InitiateMultiPartUpload(service)
		if err == nil {
			break
		}
	}
	if err != nil {
		return nil, fmt.Errorf("Failed to initiate upload - %s", err)
	}

	chunkSize := transfer.MB * 100
	if rt.Length() > transfer.TB {
		chunkSize = transfer.MB * 500
	}

	numChunks := int(math.Ceil(float64(rt.Length() / chunkSize)))
	parts := []map[string]interface{}{}
	for i := 1; i <= numChunks; i++ {
		tmpURL, err := d.TempUploadURL(service, uploadInfo.FileName, strconv.Itoa(i), uploadInfo.UploadID)
		if err != nil {
			return nil, err
		}

		if i == numChunks {
			chunkSize = (transfer.ByteSize)(int(rt.Length()) - int(rt.Transferred()))
		}

		readBuffer := make([]byte, int(chunkSize))
		bytesRead, err := rt.Read(readBuffer)
		if err != nil {
			return nil, err
		}
		if bytesRead < int(chunkSize) {
			return nil, fmt.Errorf("Failed to read from file - attempted to read %v but read %v. Import failed.", int(chunkSize), bytesRead)
		}

		var uploadResp *http.Response
		done := make(chan bool)
		for attempt := 0; attempt < 5; attempt++ {
			tempReader := bytes.NewReader(readBuffer)
			chunkRT := transfer.NewReaderTransfer(io.LimitReader(tempReader, int64(chunkSize)), int(chunkSize))

			go printTransferStatus(false, chunkRT, i, numChunks, done)

			req, err := http.NewRequest("PUT", tmpURL.URL, chunkRT)
			req.ContentLength = int64(chunkRT.Length())

			uploadResp, err = http.DefaultClient.Do(req)
			if err == nil && uploadResp.StatusCode == 200 {
				break
			}
			logrus.Printf("Chunk upload %s failed.\nResponse code: %s\nErr: %s\nRetrying... )", strconv.Itoa(i), uploadResp.StatusCode, err)
			time.Sleep(time.Second * 15)
		}
		if err != nil {
			done <- false
			return nil, err
		}
		defer uploadResp.Body.Close()
		if uploadResp.StatusCode != 200 {
			done <- false
			b, err := ioutil.ReadAll(uploadResp.Body)
			return nil, fmt.Errorf("Failed to upload import file - received status code %d %s %s", uploadResp.StatusCode, string(b), err)
		}
		etag := uploadResp.Header.Get("ETag")
		parts = append(parts, map[string]interface{}{
			"ETag":       etag,
			"PartNumber": i,
		})
		done <- true
	}

	for attempt := 0; attempt < 5; attempt++ {
		_, err = d.CompleteMultiPartUpload(service, uploadInfo.FileName, uploadInfo.UploadID, parts)
		if err == nil {
			break
		}
	}
	if err != nil {
		return nil, fmt.Errorf("Failed to complete upload - %s", err)
	}

	importParams := map[string]interface{}{}
	for key, value := range options {
		importParams[key] = value
	}
	importParams["filename"] = uploadInfo.FileName
	importParams["encryptionKey"] = string(d.Crypto.Hex(key, crypto.KeySize*2))
	importParams["encryptionIV"] = string(d.Crypto.Hex(iv, crypto.IVSize*2))
	importParams["dropDatabase"] = false

	b, err := json.Marshal(importParams)
	if err != nil {
		return nil, err
	}
	headers := d.Settings.HTTPManager.GetHeaders(d.Settings.SessionToken, d.Settings.Version, d.Settings.Pod, d.Settings.UsersID)
	resp, statusCode, err := d.Settings.HTTPManager.Post(b, fmt.Sprintf("%s%s/environments/%s/services/%s/import", d.Settings.PaasHost, d.Settings.PaasHostVersion, d.Settings.EnvironmentID, service.ID), headers)
	if err != nil {
		return nil, err
	}
	var job models.Job
	err = d.Settings.HTTPManager.ConvertResp(resp, statusCode, &job)
	if err != nil {
		return nil, err
	}
	return &job, nil
}

func (d *SDb) InitiateMultiPartUpload(service *models.Service) (*models.MultipartUploadInfo, error) {
	headers := d.Settings.HTTPManager.GetHeaders(d.Settings.SessionToken, d.Settings.Version, d.Settings.Pod, d.Settings.UsersID)
	resp, statusCode, err := d.Settings.HTTPManager.Post(nil, fmt.Sprintf("%s%s/environments/%s/services/%s/initiate-multipart-upload", d.Settings.PaasHost, d.Settings.PaasHostVersion, d.Settings.EnvironmentID, service.ID), headers)
	if err != nil {
		return nil, err
	}
	var uploadInfo models.MultipartUploadInfo
	err = d.Settings.HTTPManager.ConvertResp(resp, statusCode, &uploadInfo)
	if err != nil {
		return nil, err
	}
	return &uploadInfo, nil
}

func (d *SDb) CompleteMultiPartUpload(service *models.Service, fileName string, uploadID string, parts []map[string]interface{}) (*models.Location, error) {
	headers := d.Settings.HTTPManager.GetHeaders(d.Settings.SessionToken, d.Settings.Version, d.Settings.Pod, d.Settings.UsersID)
	body, err := json.Marshal(parts)
	if err != nil {
		return nil, err
	}
	resp, statusCode, err := d.Settings.HTTPManager.Post(body, fmt.Sprintf("%s%s/environments/%s/services/%s/complete-multipart-upload?fileName=%s&uploadId=%s", d.Settings.PaasHost, d.Settings.PaasHostVersion, d.Settings.EnvironmentID, service.ID, fileName, uploadID), headers)
	if err != nil {
		return nil, err
	}
	var location models.Location
	err = d.Settings.HTTPManager.ConvertResp(resp, statusCode, &location)
	if err != nil {
		return nil, err
	}
	return &location, nil
}

func (d *SDb) TempUploadURL(service *models.Service, fileName string, partNumber string, uploadID string) (*models.TempURL, error) {
	headers := d.Settings.HTTPManager.GetHeaders(d.Settings.SessionToken, d.Settings.Version, d.Settings.Pod, d.Settings.UsersID)
	resp, statusCode, err := d.Settings.HTTPManager.Get(nil, fmt.Sprintf("%s%s/environments/%s/services/%s/multipart-upload-url?fileName=%s&partNumber=%s&uploadId=%s", d.Settings.PaasHost, d.Settings.PaasHostVersion, d.Settings.EnvironmentID, service.ID, fileName, partNumber, uploadID), headers)
	if err != nil {
		return nil, err
	}
	var tempURL models.TempURL
	err = d.Settings.HTTPManager.ConvertResp(resp, statusCode, &tempURL)
	if err != nil {
		return nil, err
	}
	return &tempURL, nil
}
