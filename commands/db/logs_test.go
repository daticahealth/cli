package db

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/daticahealth/cli/commands/services"
	"github.com/daticahealth/cli/lib/compress"
	"github.com/daticahealth/cli/lib/crypto"
	"github.com/daticahealth/cli/lib/jobs"
	"github.com/daticahealth/cli/test"
)

var dbLogsTests = []struct {
	databaseName string
	jobID        string
	expectErr    bool
}{
	{dbName, dbJobID, false},
	{dbName, "invalid-job", true},
	{dbName, dbImportID, true},
	{"invalid-svc", dbJobID, true},
}

func TestDbLogs(t *testing.T) {
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
			fmt.Fprint(w, fmt.Sprintf(`{"id":"%s","type":"backup","status":"finished","backup":{"keyLogs":"0000000000000000000000000000000000000000000000000000000000000000","iv":"000000000000000000000000"}}`, dbJobID))
		},
	)
	mux.HandleFunc("/environments/"+test.EnvID+"/services/"+dbID+"/jobs/"+dbImportID,
		func(w http.ResponseWriter, r *http.Request) {
			test.AssertEquals(t, r.Method, "GET")
			fmt.Fprint(w, fmt.Sprintf(`{"id":"%s","type":"restore","status":"failed","restore":{"keyLogs":"0000000000000000000000000000000000000000000000000000000000000000","iv":"000000000000000000000000"}}`, dbJobID))
		},
	)
	mux.HandleFunc("/environments/"+test.EnvID+"/services/"+dbID+"/backup-restore-logs-url/"+dbJobID,
		func(w http.ResponseWriter, r *http.Request) {
			test.AssertEquals(t, r.Method, "GET")
			fmt.Fprint(w, fmt.Sprintf(`{"url":"%s/logs"}`, baseURL.String()))
		},
	)
	mux.HandleFunc("/logs",
		func(w http.ResponseWriter, r *http.Request) {
			test.AssertEquals(t, r.Method, "GET")
			w.Write([]byte{186, 194, 51, 73, 71, 71, 38, 3, 182, 216, 210, 144, 156, 237, 120, 227, 95, 91, 197, 59, 19}) // gcm encrypted "test"
		},
	)

	for _, data := range dbLogsTests {
		t.Logf("Data: %+v", data)

		// test
		err := CmdLogs(data.databaseName, data.jobID, New(settings, crypto.New(), compress.New(), jobs.New(settings)), services.New(settings), jobs.New(settings))

		// assert
		if err != nil {
			if !data.expectErr {
				t.Errorf("Unexpected error: %s", err)
			}
			continue
		}
	}
}
