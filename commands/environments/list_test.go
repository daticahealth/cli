package environments

import (
	"fmt"
	"net/http"
	"reflect"
	"testing"

	"github.com/daticahealth/cli/models"
	"github.com/daticahealth/cli/test"
)

func TestList(t *testing.T) {
	mux, server, baseURL := test.Setup()
	defer test.Teardown(server)
	settings := test.GetSettings(baseURL.String())
	mux.HandleFunc("/environments",
		func(w http.ResponseWriter, r *http.Request) {
			test.AssertEquals(t, r.Method, "GET")
			if r.Header.Get("X-Pod-ID") == test.Pod {
				fmt.Fprint(w, fmt.Sprintf(`[{"id":"%s","name":"%s","namespace":"%s","organizationId":"%s"}]`, test.EnvID, test.EnvName, test.Namespace, test.OrgID))
			} else {
				fmt.Fprint(w, fmt.Sprintf(`[{"id":"%s","name":"%s","namespace":"%s","organizationId":"%s"}]`, test.EnvIDAlt, test.EnvNameAlt, test.NamespaceAlt, test.OrgIDAlt))
			}
		},
	)

	err := CmdList(settings, New(settings))

	// assert
	if err != nil {
		t.Fatalf("Unexpected error: %s", err)
	}
	// check that the local cache was updated
	expected := map[string]models.AssociatedEnvV2{
		test.EnvName: models.AssociatedEnvV2{
			EnvironmentID: test.EnvID,
			Name:          test.EnvName,
			Pod:           test.Pod,
			OrgID:         test.OrgID,
		},
		test.EnvNameAlt: models.AssociatedEnvV2{
			EnvironmentID: test.EnvIDAlt,
			Name:          test.EnvNameAlt,
			Pod:           test.PodAlt,
			OrgID:         test.OrgIDAlt,
		},
	}
	if !reflect.DeepEqual(settings.Environments, expected) {
		t.Fatalf("Environment cache differs. Expected %+v, actual %+v", expected, settings.Environments)
	}
}

func TestListWithPodError(t *testing.T) {
	mux, server, baseURL := test.Setup()
	defer test.Teardown(server)
	settings := test.GetSettings(baseURL.String())
	mux.HandleFunc("/environments",
		func(w http.ResponseWriter, r *http.Request) {
			test.AssertEquals(t, r.Method, "GET")
			w.WriteHeader(500)
		},
	)

	err := CmdList(settings, New(settings))

	// assert
	if err != nil {
		t.Fatalf("Unexpected error: %s", err)
	}
}
