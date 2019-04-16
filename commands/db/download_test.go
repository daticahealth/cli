package db

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/daticahealth/cli/commands/services"
	"github.com/daticahealth/cli/lib/compress"
	"github.com/daticahealth/cli/lib/crypto"
	"github.com/daticahealth/cli/lib/jobs"
	"github.com/daticahealth/cli/test"
)

var downloadFilePath = "db-download.sql"

var dbDownloadTests = []struct {
	databaseName string
	backupID     string
	filePath     string
	force        bool
	expectErr    bool
}{
	{dbName, dbJobID, downloadFilePath, false, false},
	{dbName, dbJobID, downloadFilePath, false, true}, // same filename without force fails
	{dbName, dbJobID, downloadFilePath, true, false}, // same filename with force passes
	{dbName, "invalid-job", downloadFilePath, true, true},
	{"invalid-svc", dbJobID, downloadFilePath, true, true},
}

func TestDbDownload(t *testing.T) {
	mux, server, baseURL := test.Setup()
	defer test.Teardown(server)
	settings := test.GetSettings(baseURL.String())

	mux.HandleFunc("/environments/"+test.EnvID+"/services",
		func(w http.ResponseWriter, r *http.Request) {
			test.AssertEquals(t, r.Method, "GET")
			fmt.Fprint(w, fmt.Sprintf(`[{"id":"%s","label":"%s"}]`, dbID, dbName))
		},
	)
	mux.HandleFunc("/environments/"+test.EnvID+"/services/"+dbID+"/jobs/"+dbJobID,
		func(w http.ResponseWriter, r *http.Request) {
			test.AssertEquals(t, r.Method, "GET")
			fmt.Fprint(w, fmt.Sprintf(`{"id":"%s","isSnapshotBackup":false,"type":"backup","status":"finished","backup":{"key":"0000000000000000000000000000000000000000000000000000000000000000","iv":"000000000000000000000000"}}`, dbJobID))
		},
	)
	mux.HandleFunc("/environments/"+test.EnvID+"/services/"+dbID+"/backup-url/"+dbJobID,
		func(w http.ResponseWriter, r *http.Request) {
			test.AssertEquals(t, r.Method, "GET")
			fmt.Fprint(w, fmt.Sprintf(`{"url":"%s/backup"}`, baseURL.String()))
		},
	)
	mux.HandleFunc("/backup",
		func(w http.ResponseWriter, r *http.Request) {
			test.AssertEquals(t, r.Method, "GET")
			w.Write([]byte{186, 194, 51, 73, 71, 71, 38, 3, 182, 216, 210, 144, 156, 237, 120, 227, 95, 91, 197, 59, 19}) // gcm encrypted "test"
		},
	)

	for _, data := range dbDownloadTests {
		t.Logf("Data: %+v", data)

		// test
		err := CmdDownload(data.databaseName, data.backupID, data.filePath, data.force, New(settings, crypto.New(), compress.New(), jobs.New(settings)), &test.FakePrompts{}, services.New(settings))

		// assert
		if err != nil {
			if !data.expectErr {
				t.Errorf("Unexpected error: %s", err)
			}
			continue
		}

		b, _ := ioutil.ReadFile(data.filePath)
		if strings.TrimSpace(string(b)) != "test" {
			t.Errorf("Unexpected file contents. Expected: test, actual: %s", string(b))
		}
	}
	os.Remove(downloadFilePath)
}
