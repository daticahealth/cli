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

var (
	backupJobID  = "j3"
	restoreJobID = "j4"
)

var dbRestoreTests = []struct {
	databaseName string
	backupID     string
	skipConfirm  bool
	expectErr    bool
}{
	{dbName, backupJobID, true, false},
	{dbName, backupJobID, false, false},
	{dbName, "invalid-backup", true, true},
	{"invalid-svc", backupJobID, true, true},
}

func TestDbRestore(t *testing.T) {
	mux, server, baseURL := test.Setup()
	defer test.Teardown(server)
	settings := test.GetSettings(baseURL.String())

	mux.HandleFunc("/environments/"+test.EnvID+"/services",
		func(w http.ResponseWriter, r *http.Request) {
			test.AssertEquals(t, r.Method, "GET")
			fmt.Fprint(w, fmt.Sprintf(`[{"id":"%s","label":"%s"}]`, dbID, dbName))
		},
	)
	mux.HandleFunc("/environments/"+test.EnvID+"/services/"+dbID+"/jobs/"+backupJobID,
		func(w http.ResponseWriter, r *http.Request) {
			test.AssertEquals(t, r.Method, "GET")
			fmt.Fprint(w, fmt.Sprintf(`{"id":"%s","type":"backup","status":"finished","backup":{"keyLogs":"0000000000000000000000000000000000000000000000000000000000000000","iv":"000000000000000000000000"}}`, dbImportID))
		},
	)
	mux.HandleFunc("/environments/"+test.EnvID+"/services/"+dbID+"/jobs/"+restoreJobID,
		func(w http.ResponseWriter, r *http.Request) {
			test.AssertEquals(t, r.Method, "GET")
			fmt.Fprint(w, fmt.Sprintf(`{"id":"%s","type":"restore","status":"finished","restore":{"keyLogs":"0000000000000000000000000000000000000000000000000000000000000000","iv":"000000000000000000000000"}}`, restoreJobID))
		},
	)
	mux.HandleFunc("/environments/"+test.EnvID+"/services/"+dbID+"/restore",
		func(w http.ResponseWriter, r *http.Request) {
			test.AssertEquals(t, r.Method, "POST")
			fmt.Fprint(w, fmt.Sprintf(`{"id":"%s","type":"restore","status":"started","restore":{"keyLogs":"0000000000000000000000000000000000000000000000000000000000000000","iv":"000000000000000000000000"}}`, restoreJobID))
		},
	)
	mux.HandleFunc("/environments/"+test.EnvID+"/services/"+dbID+"/backup-restore-logs-url/"+restoreJobID,
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

	for _, data := range dbRestoreTests {
		t.Logf("Data: %+v", data)

		// test
		err := CmdRestore(data.databaseName, data.backupID, data.skipConfirm, New(settings, crypto.New(), compress.New(), jobs.New(settings)), &test.FakePrompts{}, services.New(settings))

		// assert
		if err != nil {
			if !data.expectErr {
				t.Errorf("Unexpected error: %s", err)
			}
			continue
		}
	}
}

func TestDbRestoreBackupNotFinished(t *testing.T) {
	mux, server, baseURL := test.Setup()
	defer test.Teardown(server)
	settings := test.GetSettings(baseURL.String())

	mux.HandleFunc("/environments/"+test.EnvID+"/services",
		func(w http.ResponseWriter, r *http.Request) {
			test.AssertEquals(t, r.Method, "GET")
			fmt.Fprint(w, fmt.Sprintf(`[{"id":"%s","label":"%s"}]`, dbID, dbName))
		},
	)
	mux.HandleFunc("/environments/"+test.EnvID+"/services/"+dbID+"/jobs/"+backupJobID,
		func(w http.ResponseWriter, r *http.Request) {
			test.AssertEquals(t, r.Method, "GET")
			fmt.Fprint(w, fmt.Sprintf(`{"id":"%s","type":"backup","status":"started","backup":{"keyLogs":"0000000000000000000000000000000000000000000000000000000000000000","iv":"000000000000000000000000"}}`, dbImportID))
		},
	)

	for _, data := range dbRestoreTests {
		t.Logf("Data: %+v", data)

		// test
		err := CmdRestore(data.databaseName, data.backupID, data.skipConfirm, New(settings, crypto.New(), compress.New(), jobs.New(settings)), &test.FakePrompts{}, services.New(settings))

		// assert
		if err == nil {
			t.Fatalf("Expected error but got nil")
		}
		t.Log(err)
	}
}

func TestDbRestoreJobNotBackup(t *testing.T) {
	mux, server, baseURL := test.Setup()
	defer test.Teardown(server)
	settings := test.GetSettings(baseURL.String())

	mux.HandleFunc("/environments/"+test.EnvID+"/services",
		func(w http.ResponseWriter, r *http.Request) {
			test.AssertEquals(t, r.Method, "GET")
			fmt.Fprint(w, fmt.Sprintf(`[{"id":"%s","label":"%s"}]`, dbID, dbName))
		},
	)
	mux.HandleFunc("/environments/"+test.EnvID+"/services/"+dbID+"/jobs/"+backupJobID,
		func(w http.ResponseWriter, r *http.Request) {
			test.AssertEquals(t, r.Method, "GET")
			fmt.Fprint(w, fmt.Sprintf(`{"id":"%s","type":"restore","status":"finished","backup":{"keyLogs":"0000000000000000000000000000000000000000000000000000000000000000","iv":"000000000000000000000000"}}`, dbImportID))
		},
	)

	for _, data := range dbRestoreTests {
		t.Logf("Data: %+v", data)

		// test
		err := CmdRestore(data.databaseName, data.backupID, data.skipConfirm, New(settings, crypto.New(), compress.New(), jobs.New(settings)), &test.FakePrompts{}, services.New(settings))

		// assert
		if err == nil {
			t.Fatalf("Expected error but got nil")
		}
		t.Log(err)
	}
}

func TestDbRestoreRestoreNeverFinishes(t *testing.T) {
	mux, server, baseURL := test.Setup()
	defer test.Teardown(server)
	settings := test.GetSettings(baseURL.String())

	mux.HandleFunc("/environments/"+test.EnvID+"/services",
		func(w http.ResponseWriter, r *http.Request) {
			test.AssertEquals(t, r.Method, "GET")
			fmt.Fprint(w, fmt.Sprintf(`[{"id":"%s","label":"%s"}]`, dbID, dbName))
		},
	)
	mux.HandleFunc("/environments/"+test.EnvID+"/services/"+dbID+"/jobs/"+backupJobID,
		func(w http.ResponseWriter, r *http.Request) {
			test.AssertEquals(t, r.Method, "GET")
			fmt.Fprint(w, fmt.Sprintf(`{"id":"%s","type":"backup","status":"finished","backup":{"keyLogs":"0000000000000000000000000000000000000000000000000000000000000000","iv":"000000000000000000000000"}}`, dbImportID))
		},
	)
	mux.HandleFunc("/environments/"+test.EnvID+"/services/"+dbID+"/jobs/"+restoreJobID,
		func(w http.ResponseWriter, r *http.Request) {
			test.AssertEquals(t, r.Method, "GET")
			fmt.Fprint(w, fmt.Sprintf(`{"id":"%s","type":"restore","status":"fake_status","restore":{"keyLogs":"0000000000000000000000000000000000000000000000000000000000000000","iv":"000000000000000000000000"}}`, restoreJobID))
		},
	)
	mux.HandleFunc("/environments/"+test.EnvID+"/services/"+dbID+"/restore",
		func(w http.ResponseWriter, r *http.Request) {
			test.AssertEquals(t, r.Method, "POST")
			fmt.Fprint(w, fmt.Sprintf(`{"id":"%s","type":"restore","status":"running","restore":{"keyLogs":"0000000000000000000000000000000000000000000000000000000000000000","iv":"000000000000000000000000"}}`, restoreJobID))
		},
	)

	for _, data := range dbRestoreTests {
		t.Logf("Data: %+v", data)

		// test
		err := CmdRestore(data.databaseName, data.backupID, data.skipConfirm, New(settings, crypto.New(), compress.New(), jobs.New(settings)), &test.FakePrompts{}, services.New(settings))

		// assert
		if err == nil {
			t.Fatalf("Expected error but got nil")
		}
		t.Log(err)
	}
}
