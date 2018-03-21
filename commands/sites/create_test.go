package sites

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/daticahealth/cli/commands/certs"
	"github.com/daticahealth/cli/commands/services"
	"github.com/daticahealth/cli/models"
	"github.com/daticahealth/cli/test"
)

var createTests = []struct {
	name                 string
	svcName              string
	certName             string
	downStream           string
	clientMaxBodySize    int
	proxyConnectTimeout  int
	proxyReadTimeout     int
	proxySendTimeout     int
	proxyUpstreamTimeout int
	enableCORS           string
	enableWebSockets     bool
	letsEncrypt          bool
	expectErr            bool
}{
	{"test.example.com", test.SvcLabel, "test_example_com", test.DownStream, -1, -1, -1, -1, -1, "", false, false, false},
	{"test.example.com", test.SvcLabel, "test_example_com", test.DownStream, -1, -1, -1, -1, -1, "", false, true, false},
	{"test.example.com", test.SvcLabel, "test_example_com", test.DownStream, 1, 2, 3, 4, 5, "test.example.com", true, false, false},
	{"test.example.com", test.SvcLabel, "test_example_com", test.DownStream, 1, 2, 3, 4, 5, "test.example.com", true, true, false},
	{"test.example.com", "code-invalid", "test_example_com", test.DownStream, 1, 2, 3, 4, 5, "test.example.com", true, false, true},
}

func TestCreate(t *testing.T) {
	mux, server, baseURL := test.Setup()
	defer test.Teardown(server)
	settings := test.GetSettings(baseURL.String())
	certNameSent := ""
	mux.HandleFunc("/environments/"+test.EnvID+"/services/"+test.SvcIDAlt+"/certs",
		func(w http.ResponseWriter, r *http.Request) {
			test.AssertEquals(t, r.Method, "POST")
			defer r.Body.Close()
			body, _ := ioutil.ReadAll(r.Body)
			var certReq struct {
				Name        string `json:"name"`
				LetsEncrypt bool   `json:"letsEncrypt"`
			}
			err := json.Unmarshal(body, &certReq)
			if err != nil || !certReq.LetsEncrypt {
				w.WriteHeader(400)
			}
			fmt.Fprint(w, `{}`)
		},
	)
	mux.HandleFunc("/environments/"+test.EnvID+"/services/"+test.SvcIDAlt+"/sites",
		func(w http.ResponseWriter, r *http.Request) {
			test.AssertEquals(t, r.Method, "POST")
			defer r.Body.Close()
			body, _ := ioutil.ReadAll(r.Body)
			var site models.Site
			json.Unmarshal(body, &site)
			certNameSent = site.Name
			fmt.Fprint(w, `{}`)
		},
	)
	mux.HandleFunc("/environments/"+test.EnvID+"/services",
		func(w http.ResponseWriter, r *http.Request) {
			test.AssertEquals(t, r.Method, "GET")
			fmt.Fprint(w, fmt.Sprintf(`[{"id":"%s","label":"%s"},{"id":"%s","label":"%s"}]`, test.SvcID, test.SvcLabel, test.SvcIDAlt, test.DownStream))
		},
	)

	for _, data := range createTests {
		certNameSent = ""
		t.Logf("Data: %+v", data)

		// test
		err := CmdCreate(data.name, data.svcName, data.certName, data.downStream, data.clientMaxBodySize, data.proxyConnectTimeout, data.proxyReadTimeout, data.proxySendTimeout, data.proxyUpstreamTimeout, data.enableCORS, data.enableWebSockets, data.letsEncrypt, New(settings), certs.New(settings), services.New(settings))

		// assert
		if err != nil != data.expectErr {
			t.Errorf("Unexpected error: %s", err)
			continue
		}

		if data.letsEncrypt && certNameSent != data.name {
			t.Errorf("Let's Encrypt : %s", err)
			continue
		}
	}
}
