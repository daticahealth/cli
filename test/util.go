package test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/daticahealth/cli/lib/httpclient"
	"github.com/daticahealth/cli/models"
)

const (
	Alias     = "1"
	EnvID     = "env1"
	EnvName   = "cli-tests1"
	Namespace = "namespace1"
	OrgID     = "org1"
	Pod       = "pod1"
	SvcID     = "svc1"
	SvcLabel  = "code1"

	AliasAlt     = "2"
	EnvIDAlt     = "env2"
	EnvNameAlt   = "cli-tests2"
	NamespaceAlt = "namespace2"
	OrgIDAlt     = "org2"
	PodAlt       = "pod2"
	SvcIDAlt     = "svc2"
	SvcLabelAlt  = "code2"
)

func Setup() (*http.ServeMux, *httptest.Server, *url.URL) {
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)

	baseURL, _ := url.Parse(server.URL)
	return mux, server, baseURL
}

func Teardown(server *httptest.Server) {
	server.Close()
}

func AssertEquals(t *testing.T, expected, actual string) {
	if expected != actual {
		t.Errorf("Expected: %s, actual: %s", expected, actual)
	}
}

func GetSettings(baseURL string) *models.Settings {
	return &models.Settings{
		SessionToken:   "token",
		PrivateKeyPath: "ssh_rsa",
		HTTPManager:    httpclient.NewTLSHTTPManager(false),
		PaasHost:       baseURL,
		Environments: map[string]models.AssociatedEnvV2{
			Alias: models.AssociatedEnvV2{
				Name:          EnvName,
				EnvironmentID: EnvID,
				Pod:           Pod,
				OrgID:         OrgID,
			},
		},
		OrgID:         OrgID,
		EnvironmentID: EnvID,
		Pods: &[]models.Pod{
			models.Pod{
				Name: Pod,
			},
			models.Pod{
				Name: PodAlt,
			},
		},
	}
}

type FakePrompts struct{}

func (f *FakePrompts) EmailPassword(e, p string) (string, string, error) {
	return "email", "password", nil
}
func (f *FakePrompts) KeyPassphrase(string) string {
	return "passphrase"
}
func (f *FakePrompts) Password(msg string) string {
	return "password"
}
func (f *FakePrompts) PHI() error {
	return nil
}
func (f *FakePrompts) YesNo(msg, prompt string) error {
	return nil
}
func (f *FakePrompts) OTP(string) string {
	return "123456"
}

func (f *FakePrompts) GenericPrompt(msg, prompt string, validOptions []string) string {
	return "y"
}

func (f *FakePrompts) CaptureInput(msg string) string {
	return "input"
}
