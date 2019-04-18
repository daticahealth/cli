package db

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/daticahealth/cli/commands/services"
	"github.com/daticahealth/cli/lib/httpclient"
	"github.com/daticahealth/cli/lib/jobs"
	"github.com/daticahealth/cli/lib/prompts"
	"github.com/daticahealth/cli/lib/transfer"
	"github.com/daticahealth/cli/models"
)

func CmdExport(databaseName, filePath string, force bool, id IDb, ip prompts.IPrompts, is services.IServices, ij jobs.IJobs) error {
	err := ip.PHI()
	if err != nil {
		return err
	}
	if !force {
		if _, err := os.Stat(filePath); err == nil {
			return fmt.Errorf("File already exists at path '%s'. Specify `--force` to overwrite", filePath)
		}
	} else {
		os.Remove(filePath)
	}
	service, err := is.RetrieveByLabel(databaseName)
	if err != nil {
		return err
	}
	if service == nil {
		return fmt.Errorf("Could not find a service with the label \"%s\". You can list services with the \"datica services list\" command.", databaseName)
	}
	job, err := id.Backup(service)
	if err != nil {
		return err
	}
	logrus.Printf("Export started (job ID = %s)", job.ID)
	// all because logrus treats print, println, and printf the same
	logrus.StandardLogger().Out.Write([]byte("Polling until backup finishes."))
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
	if job.Status != "finished" {
		id.DumpLogs("backup", job, service)
		return fmt.Errorf("Job finished with invalid status %s", job.Status)
	}

	err = id.Export(filePath, job, service)
	if err != nil {
		return err
	}
	err = id.DumpLogs("backup", job, service)
	if err != nil {
		return err
	}
	logrus.Printf("%s exported successfully to %s", service.Name, filePath)
	return nil
}

// Export dumps all data from a database service and downloads the encrypted
// data to the local machine. The export is accomplished by first creating a
// backup. Once finished, the CLI asks where the file can be downloaded from.
// The file is downloaded, decrypted, decompressed, and saved locally.
func (d *SDb) Export(filePath string, job *models.Job, service *models.Service) error {
	tempURL, err := d.TempDownloadURL(job.ID, service)
	if err != nil {
		return err
	}
	tr := &http.Transport{ // gzip encoded backups must first be decrypted
		DisableCompression: true, // Disable automatic decompress
	}
	client := &http.Client{Transport: tr}
	resp, err := client.Get(tempURL.URL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if httpclient.IsError(resp.StatusCode) {
		return httpclient.ConvertError(resp)
	}
	contentLength := resp.Header.Get("Content-Length")
	if contentLength == "" {
		return fmt.Errorf("Export succeeded, but Content-Length was not present in the response.")
	}
	size, err := strconv.Atoi(contentLength)
	if err != nil {
		return err
	}
	var file io.WriteCloser
	file, err = os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	contentEncoding := resp.Header.Get("Content-Encoding")
	if contentEncoding == "gzip" {
		file, err = d.Compress.NewDecompressWriteCloser(file)
		if err != nil {
			return err
		}
	}

	dfw, err := d.Crypto.NewDecryptWriteCloser(file, job.Backup.Key, job.Backup.IV)
	if err != nil {
		return err
	}

	wct := transfer.NewWriteCloserTransfer(dfw, size)
	done := make(chan bool)
	go printTransferStatus(true, wct, 1, 1, done)

	_, err = io.Copy(wct, resp.Body)
	if err != nil {
		done <- false
		dfw.Close()
		return err
	}
	done <- true
	return dfw.Close()
}

func printTransferStatus(isDownload bool, tr transfer.Transfer, partNumber, totalParts int, done <-chan bool) {
	action := "downloaded"
	final := "Download"
	status := "Finished"
	if isDownload {
		logrus.Println("Decrypting and Downloading...")
	} else {
		logrus.Printf("\nEncrypting and Uploading part %d of %d...\n", partNumber, totalParts)
		action = "uploaded"
		final = "Upload"
	}
	lastLen := 0
	success := true
	isDone := false
loop:
	for i, l := tr.Transferred(), tr.Length(); i < l; i = tr.Transferred() {
		select {
		case success = <-done:
			isDone = true
			break loop
		case <-time.After(time.Millisecond * 100):
			percent := uint64(i / l * 100)
			s := fmt.Sprintf("\r\033[m\t%s of %s (%d%%) %s", i, l, percent, action)
			fmt.Print(s)
			sLen := len(s)
			// this clears any dangling characters at the end with empty space
			if sLen < lastLen {
				fmt.Print(strings.Repeat(" ", lastLen-sLen))
			} else {
				lastLen = sLen
			}
		}
	}
	if !isDone {
		success = <-done
	}

	total := tr.Transferred()
	l := tr.Length()
	s := fmt.Sprintf("\r\033[m\t%s of %s (%d%%) %s", total, l, uint64(total/l*100), action)
	fmt.Print(s)
	sLen := len(s)
	// this clears any dangling characters at the end with empty space
	if sLen < lastLen {
		fmt.Print(strings.Repeat(" ", lastLen-sLen))
	}

	if !success {
		status = "Failed"
	}
	logrus.Printf("\n%s %s!\n", final, status)
}
