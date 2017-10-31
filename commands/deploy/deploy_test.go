package deploy

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/daticahealth/cli/commands/environments"
	"github.com/daticahealth/cli/commands/services"
	"github.com/daticahealth/cli/lib/jobs"
	"github.com/daticahealth/cli/test"
)

const (
	container = "container01"
	image     = "/namespace/image:tag"
)

var deployTests = []struct {
	container string
	image     string
	expectErr bool
}{
	{container, image, false},
	{"badcontainerservice", image, true},
	{container, "/invalid/tag:name", true},
}

func TestDeploy(t *testing.T) {
	mux, server, baseURL := test.Setup()
	defer test.Teardown(server)
	settings := test.GetSettings(baseURL.String())
	mux.HandleFunc("/environments/"+test.EnvID,
		func(w http.ResponseWriter, r *http.Request) {
			test.AssertEquals(t, r.Method, "GET")
			fmt.Fprint(w, fmt.Sprintf(`{"id":"%s","name":"%s","namespace":"%s","organizationId":"%s"}`, test.EnvID, test.EnvName, test.Namespace, test.OrgID))
		},
	)
	mux.HandleFunc("/environments/"+test.EnvID+"/services",
		func(w http.ResponseWriter, r *http.Request) {
			test.AssertEquals(t, r.Method, "GET")
			fmt.Fprint(w, fmt.Sprintf(`[{"id":"%s","label":"%s","type":"container"}]`, test.SvcID, container))
		},
	)
	mux.HandleFunc("/environments/"+test.EnvID+"/services/"+test.SvcID+"/deploy",
		func(w http.ResponseWriter, r *http.Request) {
			test.AssertEquals(t, r.Method, "POST")
			if strings.Contains(string(r.URL.Query().Get("release")), "/invalid/tag:name") {
				w.WriteHeader(400)
			} else {
				w.WriteHeader(202)
			}
		},
	)

	for _, data := range deployTests {
		t.Logf("Data: %+v", data)

		// test
		err := CmdDeploy(settings.EnvironmentID, data.container, data.image, jobs.New(settings), services.New(settings), environments.New(settings))

		// assert
		if err != nil != data.expectErr {
			t.Errorf("Unexpected error: %s", err)
			continue
		}
	}
}
