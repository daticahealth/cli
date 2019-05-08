package httpclient

import (
	"bytes"
	"crypto/rand"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"runtime"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/daticahealth/cli/config"
	"github.com/daticahealth/cli/lib/updater"
	"github.com/daticahealth/cli/models"
)

const defaultRedirectLimit = 10

// TLSHTTPManager is a extension of HTTPManager with explicit TLSv1.2 support.
type TLSHTTPManager struct {
	client *http.Client
}

// NewTLSHTTPManager constructs and returns a new instance of HTTPManager
// with TLSv1.2 and redirect support.
func NewTLSHTTPManager(skipVerify bool) models.HTTPManager {
	var tr = &http.Transport{
		TLSClientConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
		},
	}
	if skipVerify {
		tr.TLSClientConfig.InsecureSkipVerify = true
	}
	return &TLSHTTPManager{
		client: &http.Client{
			Transport:     tr,
			CheckRedirect: redirectPolicyFunc,
		},
	}
}

func redirectPolicyFunc(req *http.Request, via []*http.Request) error {
	if len(via) == 0 {
		// No redirects
		return nil
	}

	if len(via) > defaultRedirectLimit {
		return fmt.Errorf("%d consecutive requests(redirects)", len(via))
	}

	// mutate the subsequent redirect requests with the first Header
	for key, val := range via[0].Header {
		req.Header[key] = val
	}
	return nil
}

// GetHeaders builds a map of headers for a new request.
func (m *TLSHTTPManager) GetHeaders(sessionToken, version, pod, userID string) map[string][]string {
	b := make([]byte, 32)
	rand.Read(b)
	nonce := base64.StdEncoding.EncodeToString(b)
	timestamp := time.Now().Unix()
	return map[string][]string{
		"Accept":              {"application/json"},
		"Content-Type":        {"application/json"},
		"Authorization":       {fmt.Sprintf("Bearer %s", sessionToken)},
		"X-CLI-Version":       {version},
		"X-Pod-ID":            {pod},
		"X-Request-Nonce":     {nonce},
		"X-Request-Timestamp": {fmt.Sprintf("%d", timestamp)},
		"User-Agent":          {fmt.Sprintf("datica-cli-%s %s %s %s", version, runtime.GOOS, config.ArchString(), userID)},
	}
}

// ConvertResp takes in a resp from one of the httpclient methods and
// checks if it is a successful request. If not, it is parsed as an error object
// and returned as an error. Otherwise it will be marshalled into the requested
// interface. ALWAYS PASS A POINTER INTO THIS METHOD. If you don't pass a struct
// pointer your original object will be nil or an empty struct.
func (m *TLSHTTPManager) ConvertResp(b []byte, statusCode int, s interface{}) error {
	logrus.Debugf("%d resp: %s", statusCode, string(b))
	if IsError(statusCode) {
		return convertError(b, statusCode)
	}
	if b == nil || len(b) == 0 || s == nil {
		return nil
	}
	return json.Unmarshal(b, s)
}

// ConvertError converts a known error response into a go-usable error.
func ConvertError(resp *http.Response) error {
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	return convertError(b, resp.StatusCode)
}

// ConvertError takes in a response from one of the httpclient methods and converts it
// to a usable error object.
func (m *TLSHTTPManager) ConvertError(b []byte, statusCode int) (*models.Error, error) {
	if !IsError(statusCode) {
		return nil, errors.New("tried to convert a non-error response into an error")
	}
	var resp models.Error
	err := json.Unmarshal(b, &resp)
	return &resp, err
}

// IsError checks if an HTTP response code is outside of the "OK" range.
func IsError(statusCode int) bool {
	return statusCode < 200 || statusCode >= 300
}

// convertError attempts to convert a response into a usable error object.
func convertError(b []byte, statusCode int) error {
	msg := fmt.Sprintf("(%d)", statusCode)
	if b != nil && len(b) > 0 {
		var errs models.Error
		unmarshalErr := json.Unmarshal(b, &errs)
		if unmarshalErr == nil && errs.Title != "" && errs.Description != "" {
			msg = fmt.Sprintf("(%d) %s: %s", errs.Code, errs.Title, errs.Description)
		} else {
			var reportedErr models.ReportedError
			unmarshalErr = json.Unmarshal(b, &reportedErr)
			if unmarshalErr == nil && reportedErr.Message != "" {
				msg = fmt.Sprintf("(%d) %s", reportedErr.Code, reportedErr.Message)
			} else {
				msg = fmt.Sprintf("(%d) %s", statusCode, string(b))
			}
		}
	}
	return errors.New(msg)
}

// Get performs a GET request
func (m *TLSHTTPManager) Get(body []byte, url string, headers map[string][]string) ([]byte, int, error) {
	reader := bytes.NewReader(body)
	return m.makeRequest("GET", url, reader, headers)
}

// Post performs a POST request
func (m *TLSHTTPManager) Post(body []byte, url string, headers map[string][]string) ([]byte, int, error) {
	reader := bytes.NewReader(body)
	return m.makeRequest("POST", url, reader, headers)
}

// PostFile uploads a file with a POST
func (m *TLSHTTPManager) PostFile(filepath string, url string, headers map[string][]string) ([]byte, int, error) {
	return m.uploadFile("POST", filepath, url, headers)
}

// PutFile uploads a file with a PUT
func (m *TLSHTTPManager) PutFile(filepath string, url string, headers map[string][]string) ([]byte, int, error) {
	return m.uploadFile("PUT", filepath, url, headers)
}

func (m *TLSHTTPManager) uploadFile(method, filepath, url string, headers map[string][]string) ([]byte, int, error) {
	logrus.Debugf("%s %s", method, url)
	logrus.Debugf("%+v", headers)
	logrus.Debugf("%s", filepath)
	file, err := os.Open(filepath)
	defer file.Close()
	if err != nil {
		return nil, 0, err
	}
	info, _ := file.Stat()
	req, _ := http.NewRequest(method, url, file)
	req.ContentLength = info.Size()

	resp, err := m.client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	respBody, _ := ioutil.ReadAll(resp.Body)
	return respBody, resp.StatusCode, nil
}

// Put performs a PUT request
func (m *TLSHTTPManager) Put(body []byte, url string, headers map[string][]string) ([]byte, int, error) {
	reader := bytes.NewReader(body)
	return m.makeRequest("PUT", url, reader, headers)
}

// Delete performs a DELETE request
func (m *TLSHTTPManager) Delete(body []byte, url string, headers map[string][]string) ([]byte, int, error) {
	reader := bytes.NewReader(body)
	return m.makeRequest("DELETE", url, reader, headers)
}

// MakeRequest is a generic HTTP runner that performs a request and returns
// the result body as a byte array. It's up to the caller to transform them
// into an object.
func (m *TLSHTTPManager) makeRequest(method string, url string, body io.Reader, headers map[string][]string) ([]byte, int, error) {
	logrus.Debugf("%s %s", method, url)
	logrus.Debugf("%+v", headers)
	logrus.Debugf("%s", body)
	req, _ := http.NewRequest(method, url, body)
	req.Header = headers

	resp, err := m.client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	respBody, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode == 412 {
		updater.AutoUpdater.ForcedUpgrade()
		return nil, 0, fmt.Errorf("A required update has been applied. Please re-run this command.")
	}
	return respBody, resp.StatusCode, nil
}
