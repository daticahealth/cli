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

var dbListTests = []struct {
	databaseName string
	page         int
	pageSize     int
	expectErr    bool
}{
	{dbName, 1, 10, false},
	{dbName, 2, 10, false},
	{"invalid-svc", 1, 10, true},
}

func TestDbList(t *testing.T) {
	mux, server, baseURL := test.Setup()
	defer test.Teardown(server)
	settings := test.GetSettings(baseURL.String())

	mux.HandleFunc("/environments/"+test.EnvID+"/services",
		func(w http.ResponseWriter, r *http.Request) {
			test.AssertEquals(t, r.Method, "GET")
			fmt.Fprint(w, fmt.Sprintf(`[{"id":"%s","label":"%s"}]`, dbID, dbName))
		},
	)
	mux.HandleFunc("/environments/"+test.EnvID+"/services/"+dbID+"/jobs",
		func(w http.ResponseWriter, r *http.Request) {
			test.AssertEquals(t, r.Method, "GET")
			fmt.Fprint(w, fmt.Sprintf(`[{"id":"%s","isSnapshotBackup":false,"type":"backup","status":"finished"}]`, dbJobID))
		},
	)

	for _, data := range dbListTests {
		t.Logf("Data: %+v", data)

		// test
		err := CmdList(data.databaseName, data.page, data.pageSize, New(settings, crypto.New(), compress.New(), jobs.New(settings)), services.New(settings))

		// assert
		if err != nil != data.expectErr {
			t.Errorf("Unexpected error: %s", err)
			continue
		}
	}
}
