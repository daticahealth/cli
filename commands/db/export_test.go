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

var exportFilePath = "db-export.sql"

var dbExportTests = []struct {
	databaseName string
	filePath     string
	force        bool
	expectErr    bool
}{
	{dbName, exportFilePath, false, false},
	{dbName, exportFilePath, false, true}, // same filename without force fails
	{dbName, exportFilePath, true, false}, // same filename with force passes
	{"invalid-svc", exportFilePath, true, true},
}
func TestDbExport(t *testing.T) {
	mux, server, baseURL := test.Setup()
	defer test.Teardown(server)
	settings := test.GetSettings(baseURL.String())

	mux.HandleFunc("/environments/"+test.EnvID+"/services",
		func(w http.ResponseWriter, r *http.Request) {
			test.AssertEquals(t, r.Method, "GET")
			fmt.Fprint(w, fmt.Sprintf(`[{"id":"%s","label":"%s"}]`, dbID, dbName))
		},
	)
	mux.HandleFunc("/environments/"+test.EnvID+"/services/"+dbID+"/backup",
		func(w http.ResponseWriter, r *http.Request) {
			test.AssertEquals(t, r.Method, "POST")
			fmt.Fprint(w, fmt.Sprintf(`{"id":"%s","isSnapshotBackup":false,"type":"backup","status":"running","backup":{"key":"0000000000000000000000000000000000000000000000000000000000000000","keyLogs":"0000000000000000000000000000000000000000000000000000000000000000","iv":"000000000000000000000000"}}`, dbJobID))
		},
	)
	mux.HandleFunc("/environments/"+test.EnvID+"/services/"+dbID+"/jobs/"+dbJobID,
		func(w http.ResponseWriter, r *http.Request) {
			test.AssertEquals(t, r.Method, "GET")
			fmt.Fprint(w, fmt.Sprintf(`{"id":"%s","isSnapshotBackup":false,"type":"backup","status":"finished","backup":{"key":"0000000000000000000000000000000000000000000000000000000000000000","keyLogs":"0000000000000000000000000000000000000000000000000000000000000000","iv":"000000000000000000000000"}}`, dbJobID))
		},
	)
	mux.HandleFunc("/environments/"+test.EnvID+"/services/"+dbID+"/backup-url/"+dbJobID,
		func(w http.ResponseWriter, r *http.Request) {
			test.AssertEquals(t, r.Method, "GET")
			fmt.Fprint(w, fmt.Sprintf(`{"url":"%s/backup"}`, baseURL.String()))
		},
	)
	mux.HandleFunc("/environments/"+test.EnvID+"/services/"+dbID+"/backup-restore-logs-url/"+dbJobID,
		func(w http.ResponseWriter, r *http.Request) {
			test.AssertEquals(t, r.Method, "GET")
			fmt.Fprint(w, fmt.Sprintf(`{"url":"%s/backup"}`, baseURL.String()))
		},
	)
	mux.HandleFunc("/logs",
		func(w http.ResponseWriter, r *http.Request) {
			test.AssertEquals(t, r.Method, "GET")
			w.Write([]byte{186, 194, 51, 73, 71, 71, 38, 3, 182, 216, 210, 144, 156, 237, 120, 227, 95, 91, 197, 59, 19}) // gcm encrypted "test"
		},
	)
	mux.HandleFunc("/backup",
		func(w http.ResponseWriter, r *http.Request) {
			test.AssertEquals(t, r.Method, "GET")
			w.Header().Set("x-amz-meta-datica-backup-compression", "gzip")
			w.Write([]byte{209, 44, 72, 61, 170, 141, 222, 50, 7, 77, 238, 154, 191, 243, 71, 35, 32, 87, 1, 202, 55, 166, 53, 255, 217, 162, 8, 99, 192, 90, 53, 141, 246, 100, 176, 40, 162, 199, 66, 105, 67, 232, 89, 36, 88, 135, 62, 247, 72, 175, 126, 189, 129, 184, 156, 50, 97, 239, 27, 52, 153, 13, 58, 31, 208, 0, 133, 34, 178, 168, 138, 94, 1, 4, 248, 121, 204, 184, 152, 136, 176}) // gcm encrypted gzip compressed "test" (concatenation of three gzip streams ["te", "st", "\n"], extra length is due to multiple compression headers)
		},
	)

	for _, data := range dbExportTests {
		t.Logf("Data: %+v", data)

		// test
		err := CmdExport(data.databaseName, data.filePath, data.force, New(settings, crypto.New(), compress.New(), jobs.New(settings)), &test.FakePrompts{}, services.New(settings), jobs.New(settings))

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
	os.Remove(exportFilePath)
}

func TestDbExportUncompressed(t *testing.T) {
	mux, server, baseURL := test.Setup()
	defer test.Teardown(server)
	settings := test.GetSettings(baseURL.String())

	mux.HandleFunc("/environments/"+test.EnvID+"/services",
		func(w http.ResponseWriter, r *http.Request) {
			test.AssertEquals(t, r.Method, "GET")
			fmt.Fprint(w, fmt.Sprintf(`[{"id":"%s","label":"%s"}]`, dbID, dbName))
		},
	)
	mux.HandleFunc("/environments/"+test.EnvID+"/services/"+dbID+"/backup",
		func(w http.ResponseWriter, r *http.Request) {
			test.AssertEquals(t, r.Method, "POST")
			fmt.Fprint(w, fmt.Sprintf(`{"id":"%s","isSnapshotBackup":false,"type":"backup","status":"running","backup":{"key":"0000000000000000000000000000000000000000000000000000000000000000","keyLogs":"0000000000000000000000000000000000000000000000000000000000000000","iv":"000000000000000000000000"}}`, dbJobID))
		},
	)
	mux.HandleFunc("/environments/"+test.EnvID+"/services/"+dbID+"/jobs/"+dbJobID,
		func(w http.ResponseWriter, r *http.Request) {
			test.AssertEquals(t, r.Method, "GET")
			fmt.Fprint(w, fmt.Sprintf(`{"id":"%s","isSnapshotBackup":false,"type":"backup","status":"finished","backup":{"key":"0000000000000000000000000000000000000000000000000000000000000000","keyLogs":"0000000000000000000000000000000000000000000000000000000000000000","iv":"000000000000000000000000"}}`, dbJobID))
		},
	)
	mux.HandleFunc("/environments/"+test.EnvID+"/services/"+dbID+"/backup-url/"+dbJobID,
		func(w http.ResponseWriter, r *http.Request) {
			test.AssertEquals(t, r.Method, "GET")
			fmt.Fprint(w, fmt.Sprintf(`{"url":"%s/backup"}`, baseURL.String()))
		},
	)
	mux.HandleFunc("/environments/"+test.EnvID+"/services/"+dbID+"/backup-restore-logs-url/"+dbJobID,
		func(w http.ResponseWriter, r *http.Request) {
			test.AssertEquals(t, r.Method, "GET")
			fmt.Fprint(w, fmt.Sprintf(`{"url":"%s/backup"}`, baseURL.String()))
		},
	)
	mux.HandleFunc("/logs",
		func(w http.ResponseWriter, r *http.Request) {
			test.AssertEquals(t, r.Method, "GET")
			w.Write([]byte{186, 194, 51, 73, 71, 71, 38, 3, 182, 216, 210, 144, 156, 237, 120, 227, 95, 91, 197, 59, 19}) // gcm encrypted "test"
		},
	)
	mux.HandleFunc("/backup",
		func(w http.ResponseWriter, r *http.Request) {
			test.AssertEquals(t, r.Method, "GET")
			w.Write([]byte{186, 194, 51, 73, 71, 71, 38, 3, 182, 216, 210, 144, 156, 237, 120, 227, 95, 91, 197, 59, 19}) // gcm encrypted "test"
		},
	)

	for _, data := range dbExportTests {
		t.Logf("Data: %+v", data)

		// test
		err := CmdExport(data.databaseName, data.filePath, data.force, New(settings, crypto.New(), compress.New(), jobs.New(settings)), &test.FakePrompts{}, services.New(settings), jobs.New(settings))

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
	os.Remove(exportFilePath)
}
