package db

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"testing"

	"github.com/daticahealth/cli/commands/services"
	"github.com/daticahealth/cli/lib/crypto"
	"github.com/daticahealth/cli/lib/jobs"
	"github.com/daticahealth/cli/test"
)

var (
	importFilePath = "db-import.sql"
	dbImportID     = "j2"
)

var dbImportTests = []struct {
	databaseName string
	filePath     string
	collection   string
	database     string
	skipBackup   bool
	expectErr    bool
}{
	{dbName, importFilePath, "", "", false, false},
	{dbName, importFilePath, "", "", true, false},
	{dbName, "invalid-file", "", "", false, true},
	{"invalid-svc", importFilePath, "", "", false, true},
}

func TestDbImport(t *testing.T) {
	ioutil.WriteFile(importFilePath, []byte("select 1;"), 0644)
	mux, server, baseURL := test.Setup()
	defer test.Teardown(server)
	settings := test.GetSettings(baseURL.String())

	backedUp := false
	mux.HandleFunc("/environments/"+test.EnvID+"/services",
		func(w http.ResponseWriter, r *http.Request) {
			test.AssertEquals(t, r.Method, "GET")
			fmt.Fprint(w, fmt.Sprintf(`[{"id":"%s","label":"%s"}]`, dbID, dbName))
		},
	)
	mux.HandleFunc("/environments/"+test.EnvID+"/services/"+dbID+"/backup",
		func(w http.ResponseWriter, r *http.Request) {
			backedUp = true
			test.AssertEquals(t, r.Method, "POST")
			fmt.Fprint(w, fmt.Sprintf(`{"id":"%s","isSnapshotBackup":false,"type":"backup","status":"running","backup":{"keyLogs":"0000000000000000000000000000000000000000000000000000000000000000","iv":"000000000000000000000000"}}`, dbJobID))
		},
	)
	mux.HandleFunc("/environments/"+test.EnvID+"/services/"+dbID+"/import",
		func(w http.ResponseWriter, r *http.Request) {
			test.AssertEquals(t, r.Method, "POST")
			fmt.Fprint(w, fmt.Sprintf(`{"id":"%s","type":"restore","status":"running","restore":{"keyLogs":"0000000000000000000000000000000000000000000000000000000000000000","iv":"000000000000000000000000"}}`, dbImportID))
		},
	)
	mux.HandleFunc("/environments/"+test.EnvID+"/services/"+dbID+"/jobs/"+dbJobID,
		func(w http.ResponseWriter, r *http.Request) {
			test.AssertEquals(t, r.Method, "GET")
			fmt.Fprint(w, fmt.Sprintf(`{"id":"%s","isSnapshotBackup":false,"type":"backup","status":"finished","backup":{"keyLogs":"0000000000000000000000000000000000000000000000000000000000000000","iv":"000000000000000000000000"}}`, dbJobID))
		},
	)
	mux.HandleFunc("/environments/"+test.EnvID+"/services/"+dbID+"/jobs/"+dbImportID,
		func(w http.ResponseWriter, r *http.Request) {
			test.AssertEquals(t, r.Method, "GET")
			fmt.Fprint(w, fmt.Sprintf(`{"id":"%s","type":"restore","status":"finished","restore":{"keyLogs":"0000000000000000000000000000000000000000000000000000000000000000","iv":"000000000000000000000000"}}`, dbImportID))
		},
	)
	mux.HandleFunc("/environments/"+test.EnvID+"/services/"+dbID+"/backup-restore-logs-url/"+dbJobID,
		func(w http.ResponseWriter, r *http.Request) {
			test.AssertEquals(t, r.Method, "GET")
			fmt.Fprint(w, fmt.Sprintf(`{"url":"%s/logs"}`, baseURL.String()))
		},
	)
	mux.HandleFunc("/environments/"+test.EnvID+"/services/"+dbID+"/backup-restore-logs-url/"+dbImportID,
		func(w http.ResponseWriter, r *http.Request) {
			test.AssertEquals(t, r.Method, "GET")
			fmt.Fprint(w, fmt.Sprintf(`{"url":"%s/logs"}`, baseURL.String()))
		},
	)
	mux.HandleFunc("/environments/"+test.EnvID+"/services/"+dbID+"/initiate-multipart-upload",
		func(w http.ResponseWriter, r *http.Request) {
			test.AssertEquals(t, r.Method, "POST")
			fmt.Fprint(w, fmt.Sprintf(`{"upload_id":"upload_id","file_name": "%s"}`, importFilePath))
		},
	)
	mux.HandleFunc("/environments/"+test.EnvID+"/services/"+dbID+"/multipart-upload-url",
		func(w http.ResponseWriter, r *http.Request) {
			test.AssertEquals(t, r.Method, "GET")
			fmt.Fprint(w, fmt.Sprintf(`{"url":"%s/restore"}`, baseURL.String()))
		},
	)
	mux.HandleFunc("/environments/"+test.EnvID+"/services/"+dbID+"/complete-multipart-upload",
		func(w http.ResponseWriter, r *http.Request) {
			test.AssertEquals(t, r.Method, "POST")
			fmt.Fprint(w, fmt.Sprintf(`{"location":"location"}`))
		},
	)
	mux.HandleFunc("/restore",
		func(w http.ResponseWriter, r *http.Request) {
			test.AssertEquals(t, r.Method, "PUT")
			ioutil.ReadAll(r.Body)
			r.Body.Close()
			w.WriteHeader(200)
		},
	)
	mux.HandleFunc("/logs",
		func(w http.ResponseWriter, r *http.Request) {
			test.AssertEquals(t, r.Method, "GET")
			w.Write([]byte{186, 194, 51, 73, 71, 71, 38, 3, 182, 216, 210, 144, 156, 237, 120, 227, 95, 91, 197, 59, 19}) // gcm encrypted "test"
		},
	)
	mux.HandleFunc("/healthcheck",
		func(w http.ResponseWriter, r *http.Request) {
			test.AssertEquals(t, r.Method, "GET")
			fmt.Fprint(w, fmt.Sprintf(`{"status":"ok","version": "4.1.0"}`))
		},
	)

	for _, data := range dbImportTests {
		t.Logf("Data: %+v", data)
		backedUp = false

		// test
		err := CmdImport(data.databaseName, data.filePath, data.collection, data.database, data.skipBackup, New(settings, crypto.New(), jobs.New(settings)), &test.FakePrompts{}, services.New(settings), jobs.New(settings))

		// assert
		if err != nil {
			if !data.expectErr {
				t.Errorf("Unexpected error: %s", err)
			}
			continue
		}

		if data.skipBackup == backedUp {
			t.Errorf("The skip backup flag was not properly handled: skip? %t - backed up? %t", data.skipBackup, backedUp)
			continue
		}
	}
	os.Remove(importFilePath)
}

func TestDbImportOverFiveGB(t *testing.T) {
	requestData := make([]byte, 5368709130)
	ioutil.WriteFile(importFilePath, requestData, 0644)
	mux, server, baseURL := test.Setup()
	defer test.Teardown(server)
	settings := test.GetSettings(baseURL.String())

	backedUp := false
	mux.HandleFunc("/environments/"+test.EnvID+"/services",
		func(w http.ResponseWriter, r *http.Request) {
			test.AssertEquals(t, r.Method, "GET")
			fmt.Fprint(w, fmt.Sprintf(`[{"id":"%s","label":"%s"}]`, dbID, dbName))
		},
	)
	mux.HandleFunc("/environments/"+test.EnvID+"/services/"+dbID+"/backup",
		func(w http.ResponseWriter, r *http.Request) {
			backedUp = true
			test.AssertEquals(t, r.Method, "POST")
			fmt.Fprint(w, fmt.Sprintf(`{"id":"%s","isSnapshotBackup":false,"type":"backup","status":"running","backup":{"keyLogs":"0000000000000000000000000000000000000000000000000000000000000000","iv":"000000000000000000000000"}}`, dbJobID))
		},
	)
	mux.HandleFunc("/environments/"+test.EnvID+"/services/"+dbID+"/import",
		func(w http.ResponseWriter, r *http.Request) {
			test.AssertEquals(t, r.Method, "POST")
			fmt.Fprint(w, fmt.Sprintf(`{"id":"%s","type":"restore","status":"running","restore":{"keyLogs":"0000000000000000000000000000000000000000000000000000000000000000","iv":"000000000000000000000000"}}`, dbImportID))
		},
	)
	mux.HandleFunc("/environments/"+test.EnvID+"/services/"+dbID+"/jobs/"+dbJobID,
		func(w http.ResponseWriter, r *http.Request) {
			test.AssertEquals(t, r.Method, "GET")
			fmt.Fprint(w, fmt.Sprintf(`{"id":"%s","isSnapshotBackup":false,"type":"backup","status":"finished","backup":{"keyLogs":"0000000000000000000000000000000000000000000000000000000000000000","iv":"000000000000000000000000"}}`, dbJobID))
		},
	)
	mux.HandleFunc("/environments/"+test.EnvID+"/services/"+dbID+"/jobs/"+dbImportID,
		func(w http.ResponseWriter, r *http.Request) {
			test.AssertEquals(t, r.Method, "GET")
			fmt.Fprint(w, fmt.Sprintf(`{"id":"%s","type":"restore","status":"finished","restore":{"keyLogs":"0000000000000000000000000000000000000000000000000000000000000000","iv":"000000000000000000000000"}}`, dbImportID))
		},
	)
	mux.HandleFunc("/environments/"+test.EnvID+"/services/"+dbID+"/backup-restore-logs-url/"+dbJobID,
		func(w http.ResponseWriter, r *http.Request) {
			test.AssertEquals(t, r.Method, "GET")
			fmt.Fprint(w, fmt.Sprintf(`{"url":"%s/logs"}`, baseURL.String()))
		},
	)
	mux.HandleFunc("/environments/"+test.EnvID+"/services/"+dbID+"/backup-restore-logs-url/"+dbImportID,
		func(w http.ResponseWriter, r *http.Request) {
			test.AssertEquals(t, r.Method, "GET")
			fmt.Fprint(w, fmt.Sprintf(`{"url":"%s/logs"}`, baseURL.String()))
		},
	)
	mux.HandleFunc("/environments/"+test.EnvID+"/services/"+dbID+"/initiate-multipart-upload",
		func(w http.ResponseWriter, r *http.Request) {
			test.AssertEquals(t, r.Method, "POST")
			fmt.Fprint(w, fmt.Sprintf(`{"upload_id":"upload_id","file_name": "%s"}`, importFilePath))
		},
	)
	mux.HandleFunc("/environments/"+test.EnvID+"/services/"+dbID+"/multipart-upload-url",
		func(w http.ResponseWriter, r *http.Request) {
			test.AssertEquals(t, r.Method, "GET")
			fmt.Fprint(w, fmt.Sprintf(`{"url":"%s/restore"}`, baseURL.String()))
		},
	)
	mux.HandleFunc("/environments/"+test.EnvID+"/services/"+dbID+"/complete-multipart-upload",
		func(w http.ResponseWriter, r *http.Request) {
			test.AssertEquals(t, r.Method, "POST")
			fmt.Fprint(w, fmt.Sprintf(`{"location":"location"}`))
		},
	)
	mux.HandleFunc("/restore",
		func(w http.ResponseWriter, r *http.Request) {
			test.AssertEquals(t, r.Method, "PUT")
			ioutil.ReadAll(r.Body)
			r.Body.Close()
			w.WriteHeader(200)
		},
	)
	mux.HandleFunc("/logs",
		func(w http.ResponseWriter, r *http.Request) {
			test.AssertEquals(t, r.Method, "GET")
			w.Write([]byte{186, 194, 51, 73, 71, 71, 38, 3, 182, 216, 210, 144, 156, 237, 120, 227, 95, 91, 197, 59, 19}) // gcm encrypted "test"
		},
	)
	mux.HandleFunc("/healthcheck",
		func(w http.ResponseWriter, r *http.Request) {
			test.AssertEquals(t, r.Method, "GET")
			fmt.Fprint(w, fmt.Sprintf(`{"status":"ok","version": "4.1.0"}`))
		},
	)

	for _, data := range dbImportTests {
		t.Logf("Data: %+v", data)
		backedUp = false

		// test
		err := CmdImport(data.databaseName, data.filePath, data.collection, data.database, data.skipBackup, New(settings, crypto.New(), jobs.New(settings)), &test.FakePrompts{}, services.New(settings), jobs.New(settings))

		// assert
		if err != nil {
			if !data.expectErr {
				t.Errorf("Unexpected error: %s", err)
			}
			continue
		}

		if data.skipBackup == backedUp {
			t.Errorf("The skip backup flag was not properly handled: skip? %t - backed up? %t", data.skipBackup, backedUp)
			continue
		}
	}
	os.Remove(importFilePath)
}

func TestDbImportFailedUpload(t *testing.T) {
	ioutil.WriteFile(importFilePath, []byte("select 1;"), 0644)
	mux, server, baseURL := test.Setup()
	defer test.Teardown(server)
	settings := test.GetSettings(baseURL.String())

	mux.HandleFunc("/environments/"+test.EnvID+"/services",
		func(w http.ResponseWriter, r *http.Request) {
			test.AssertEquals(t, r.Method, "GET")
			fmt.Fprint(w, fmt.Sprintf(`[{"id":"%s","label":"%s"}]`, dbID, dbName))
		},
	)
	mux.HandleFunc("/environments/"+test.EnvID+"/services/"+dbID+"/import",
		func(w http.ResponseWriter, r *http.Request) {
			test.AssertEquals(t, r.Method, "POST")
			fmt.Fprint(w, fmt.Sprintf(`{"id":"%s","type":"restore","status":"running","restore":{"keyLogs":"0000000000000000000000000000000000000000000000000000000000000000","iv":"000000000000000000000000"}}`, dbImportID))
		},
	)
	mux.HandleFunc("/environments/"+test.EnvID+"/services/"+dbID+"/initiate-multipart-upload",
		func(w http.ResponseWriter, r *http.Request) {
			test.AssertEquals(t, r.Method, "POST")
			fmt.Fprint(w, fmt.Sprintf(`{"upload_id":"upload_id","file_name": "%s"}`, importFilePath))
		},
	)
	mux.HandleFunc("/environments/"+test.EnvID+"/services/"+dbID+"/multipart-upload-url",
		func(w http.ResponseWriter, r *http.Request) {
			test.AssertEquals(t, r.Method, "GET")
			fmt.Fprint(w, fmt.Sprintf(`{"url":"%s/restore"}`, baseURL.String()))
		},
	)
	mux.HandleFunc("/environments/"+test.EnvID+"/services/"+dbID+"/complete-multipart-upload",
		func(w http.ResponseWriter, r *http.Request) {
			test.AssertEquals(t, r.Method, "POST")
			fmt.Fprint(w, fmt.Sprintf(`{"location":"location"}`))
		},
	)
	mux.HandleFunc("/restore",
		func(w http.ResponseWriter, r *http.Request) {
			test.AssertEquals(t, r.Method, "PUT")
			ioutil.ReadAll(r.Body)
			r.Body.Close()
			w.WriteHeader(200)
		},
	)
	mux.HandleFunc("/healthcheck",
		func(w http.ResponseWriter, r *http.Request) {
			test.AssertEquals(t, r.Method, "GET")
			fmt.Fprint(w, fmt.Sprintf(`{"status":"ok","version": "4.1.0"}`))
		},
	)

	// test
	err := CmdImport(dbName, importFilePath, "", "", true, New(settings, crypto.New(), jobs.New(settings)), &test.FakePrompts{}, services.New(settings), jobs.New(settings))

	// assert
	if err == nil {
		t.Fatalf("Expected error but got nil")
	}
	t.Log(err)

	os.Remove(importFilePath)
}

func TestDbImportPod05(t *testing.T) {
	ioutil.WriteFile(importFilePath, []byte("select 1;"), 0644)
	mux, server, baseURL := test.Setup()
	defer test.Teardown(server)
	settings := test.GetSettings(baseURL.String())

	backedUp := false
	mux.HandleFunc("/environments/"+test.EnvID+"/services",
		func(w http.ResponseWriter, r *http.Request) {
			test.AssertEquals(t, r.Method, "GET")
			fmt.Fprint(w, fmt.Sprintf(`[{"id":"%s","label":"%s"}]`, dbID, dbName))
		},
	)
	mux.HandleFunc("/environments/"+test.EnvID+"/services/"+dbID+"/backup",
		func(w http.ResponseWriter, r *http.Request) {
			backedUp = true
			test.AssertEquals(t, r.Method, "POST")
			fmt.Fprint(w, fmt.Sprintf(`{"id":"%s","isSnapshotBackup":false,"type":"backup","status":"running","backup":{"keyLogs":"0000000000000000000000000000000000000000000000000000000000000000","iv":"000000000000000000000000"}}`, dbJobID))
		},
	)
	mux.HandleFunc("/environments/"+test.EnvID+"/services/"+dbID+"/import",
		func(w http.ResponseWriter, r *http.Request) {
			test.AssertEquals(t, r.Method, "POST")
			fmt.Fprint(w, fmt.Sprintf(`{"id":"%s","type":"restore","status":"running","restore":{"keyLogs":"0000000000000000000000000000000000000000000000000000000000000000","iv":"000000000000000000000000"}}`, dbImportID))
		},
	)
	mux.HandleFunc("/environments/"+test.EnvID+"/services/"+dbID+"/jobs/"+dbJobID,
		func(w http.ResponseWriter, r *http.Request) {
			test.AssertEquals(t, r.Method, "GET")
			fmt.Fprint(w, fmt.Sprintf(`{"id":"%s","isSnapshotBackup":false,"type":"backup","status":"finished","backup":{"keyLogs":"0000000000000000000000000000000000000000000000000000000000000000","iv":"000000000000000000000000"}}`, dbJobID))
		},
	)
	mux.HandleFunc("/environments/"+test.EnvID+"/services/"+dbID+"/jobs/"+dbImportID,
		func(w http.ResponseWriter, r *http.Request) {
			test.AssertEquals(t, r.Method, "GET")
			fmt.Fprint(w, fmt.Sprintf(`{"id":"%s","type":"restore","status":"finished","restore":{"keyLogs":"0000000000000000000000000000000000000000000000000000000000000000","iv":"000000000000000000000000"}}`, dbImportID))
		},
	)
	mux.HandleFunc("/environments/"+test.EnvID+"/services/"+dbID+"/backup-restore-logs-url/"+dbJobID,
		func(w http.ResponseWriter, r *http.Request) {
			test.AssertEquals(t, r.Method, "GET")
			fmt.Fprint(w, fmt.Sprintf(`{"url":"%s/logs"}`, baseURL.String()))
		},
	)
	mux.HandleFunc("/environments/"+test.EnvID+"/services/"+dbID+"/backup-restore-logs-url/"+dbImportID,
		func(w http.ResponseWriter, r *http.Request) {
			test.AssertEquals(t, r.Method, "GET")
			fmt.Fprint(w, fmt.Sprintf(`{"url":"%s/logs"}`, baseURL.String()))
		},
	)
	mux.HandleFunc("/environments/"+test.EnvID+"/services/"+dbID+"/restore-url",
		func(w http.ResponseWriter, r *http.Request) {
			test.AssertEquals(t, r.Method, "GET")
			fmt.Fprint(w, fmt.Sprintf(`{"url":"%s/restore"}`, baseURL.String()))
		},
	)
	mux.HandleFunc("/restore",
		func(w http.ResponseWriter, r *http.Request) {
			test.AssertEquals(t, r.Method, "PUT")
			ioutil.ReadAll(r.Body)
			r.Body.Close()
			w.WriteHeader(200)
		},
	)
	mux.HandleFunc("/logs",
		func(w http.ResponseWriter, r *http.Request) {
			test.AssertEquals(t, r.Method, "GET")
			w.Write([]byte{186, 194, 51, 73, 71, 71, 38, 3, 182, 216, 210, 144, 156, 237, 120, 227, 95, 91, 197, 59, 19}) // gcm encrypted "test"
		},
	)
	mux.HandleFunc("/healthcheck",
		func(w http.ResponseWriter, r *http.Request) {
			test.AssertEquals(t, r.Method, "GET")
			fmt.Fprint(w, fmt.Sprintf(`{"status":"ok","version": "3.7.0"}`))
		},
	)

	for _, data := range dbImportTests {
		t.Logf("Data: %+v", data)
		backedUp = false

		// test
		err := CmdImport(data.databaseName, data.filePath, data.collection, data.database, data.skipBackup, New(settings, crypto.New(), jobs.New(settings)), &test.FakePrompts{}, services.New(settings), jobs.New(settings))

		// assert
		if err != nil {
			if !data.expectErr {
				t.Errorf("Unexpected error: %s", err)
			}
			continue
		}

		if data.skipBackup == backedUp {
			t.Errorf("The skip backup flag was not properly handled: skip? %t - backed up? %t", data.skipBackup, backedUp)
			continue
		}
	}
	os.Remove(importFilePath)
}
