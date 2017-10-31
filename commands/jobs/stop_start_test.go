package jobs

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/daticahealth/cli/commands/services"
	"github.com/daticahealth/cli/test"
)

var jobsStopTests = []struct {
	jobID     string
	svcName   string
	pod       string
	expectErr bool
}{
	{"00000000-0000-0000-0000-aaaaaaaaaaaa", test.SvcLabel, test.Pod, false},
	{"00000000-0000-0000-0000-aaaaaaaaaaaa", test.SvcLabelAlt, test.Pod, true},
	{"00000000-0000-0000-0000-aaaaaaaaaaaa", "database-1", test.Pod, true},
	{"00000000-0000-0000-0000-aaaaaaaaaaaa", "invalid-svc-name", test.Pod, true},
	{"00000000-0000-0000-0000-aaaaaaaaaaaa", "database-2", "csb01", true},
	{"00000000-0000-0000-0000-aaaaaaaaaaaa", "database-2", test.Pod, true},
}

func TestJobsStop(t *testing.T) {
	mux, server, baseURL := test.Setup()
	defer test.Teardown(server)
	settings := test.GetSettings(baseURL.String())

	// The Stop/Start commands do a lookup of a service by its name; the mocked results here help translate the service name to an ID -- which is used to call
	//	the jobs endpoint.
	mux.HandleFunc("/environments/"+test.EnvID+"/services",
		func(w http.ResponseWriter, r *http.Request) {
			test.AssertEquals(t, r.Method, "GET")
			fmt.Fprint(w, fmt.Sprintf(`[{"id":"%s","label":"%s","name":"code","redeployable":false},{"id":"%s","label":"%s","name":"code","redeployable":true},{"id":"%s","label":"database-1","name":"postgresql","redeployable":true},{"id":"%s","label":"database-2","name":"postgresql","redeployable":true}]`, test.SvcIDAlt, test.SvcLabelAlt, test.SvcID, test.SvcLabel, "3", "4"))
		},
	)
	mux.HandleFunc("/environments/"+test.EnvID+"/services/"+test.SvcID+"/jobs",
		func(w http.ResponseWriter, r *http.Request) {
			test.AssertEquals(t, r.Method, "GET")
			fmt.Fprint(w, `[{"id":"1"}]`)
		},
	)
	mux.HandleFunc("/environments/"+test.EnvID+"/services/"+test.SvcID+"/jobs/00000000-0000-0000-0000-aaaaaaaaaaaa/stop",
		func(w http.ResponseWriter, r *http.Request) {
			test.AssertEquals(t, r.Method, "POST")
		},
	)
	mux.HandleFunc("/environments/"+test.EnvID+"/services/"+test.SvcID+"/jobs/00000000-0000-0000-0000-aaaaaaaaaaaa/start",
		func(w http.ResponseWriter, r *http.Request) {
			test.AssertEquals(t, r.Method, "POST")
		},
	)
	for _, data := range jobsStopTests {
		t.Logf("Data: %+v", data)

		// test
		err := CmdStop(data.jobID, data.svcName, New(settings), services.New(settings), false, &test.FakePrompts{})

		// assert
		if err != nil != data.expectErr {
			t.Errorf("Unexpected error: %s", err)
			continue
		}

		err = CmdStart(data.jobID, data.svcName, New(settings), services.New(settings))
		t.Logf("Data: %+v", data)
		t.Logf("Error: %s", err)
		// assert
		if err != nil != data.expectErr {
			t.Errorf("Unexpected error: %s", err)
			continue
		}
	}
}
