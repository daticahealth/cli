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

const (
	dbName  = "database1"
	dbID    = "d1"
	dbJobID = "j1"
)

var dbBackupTests = []struct {
	databaseName string
	skipPoll     bool
	expectErr    bool
}{
	{dbName, false, false},
	{dbName, true, false},
	{"invalid-svc", false, true},
}

func TestDbBackup(t *testing.T) {
	mux, server, baseURL := test.Setup()
	defer test.Teardown(server)
	settings := test.GetSettings(baseURL.String())

	queriedJob := false
	mux.HandleFunc("/environments/"+test.EnvID+"/services",
		func(w http.ResponseWriter, r *http.Request) {
			test.AssertEquals(t, r.Method, "GET")
			fmt.Fprint(w, fmt.Sprintf(`[{"id":"%s","label":"%s"}]`, dbID, dbName))
		},
	)
	mux.HandleFunc("/environments/"+test.EnvID+"/services/"+dbID+"/backup",
		func(w http.ResponseWriter, r *http.Request) {
			test.AssertEquals(t, r.Method, "POST")
			fmt.Fprint(w, fmt.Sprintf(`{"id":"%s","isSnapshotBackup":false,"type":"backup","status":"running","backup":{"keyLogs":"0000000000000000000000000000000000000000000000000000000000000000","iv":"000000000000000000000000"}}`, dbJobID))
		},
	)
	mux.HandleFunc("/environments/"+test.EnvID+"/services/"+dbID+"/jobs/"+dbJobID,
		func(w http.ResponseWriter, r *http.Request) {
			queriedJob = true
			test.AssertEquals(t, r.Method, "GET")
			fmt.Fprint(w, fmt.Sprintf(`{"id":"%s","status":"finished"}`, dbJobID))
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

	for _, data := range dbBackupTests {
		t.Logf("Data: %+v", data)
		queriedJob = false

		// test
		err := CmdBackup(data.databaseName, data.skipPoll, New(settings, crypto.New(), compress.New(), jobs.New(settings)), services.New(settings), jobs.New(settings))

		// assert
		if err != nil != data.expectErr {
			t.Errorf("Unexpected error: %s", err)
			continue
		}

		if !data.expectErr && data.skipPoll == queriedJob {
			t.Errorf("The skip poll flag was not properly handled: skip? %t - job queried? %t", data.skipPoll, queriedJob)
			continue
		}
	}
}
