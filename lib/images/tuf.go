package images

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"runtime"
	"strings"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/daticahealth/cli/lib/prompts"
	"github.com/daticahealth/cli/models"
	"github.com/docker/distribution/registry/client/auth"
	"github.com/docker/distribution/registry/client/auth/challenge"
	"github.com/docker/distribution/registry/client/transport"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	dockerClient "github.com/docker/docker/client"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/docker/go-connections/tlsconfig"
	"github.com/docker/notary"
	notaryClient "github.com/docker/notary/client"
	"github.com/docker/notary/client/changelist"
	"github.com/docker/notary/cryptoservice"
	"github.com/docker/notary/passphrase"
	"github.com/docker/notary/trustmanager"
	"github.com/docker/notary/trustpinning"
	"github.com/docker/notary/tuf/data"
	"github.com/olekukonko/tablewriter"
	digest "github.com/opencontainers/go-digest"
)

var registries = map[string]string{
	"sbox05":  "registry-sbox05.datica.com",
	"default": "registry.datica.com",
}

var notaryServers = map[string]string{
	"sbox05":  "https://notary-sandbox.datica.com",
	"default": "https://notary.datica.com",
}

// Errors for image handling
const (
	InvalidImageName             = "Invalid image name"
	IncorrectNamespace           = "Incorrect namespace for your environment"
	IncorrectRegistryOrNamespace = "Incorrect registry or namespace for your environment"
	MissingTrustData             = "does not have trust data for"
	ImageDoesNotExist            = "No such image"
	CancelingPush                = "Canceling push request"

	CanonicalTargetsRole = "targets"
)

// Constants for image handling
const (
	DefaultTag = "latest"
	trustPath  = ".docker/trust"
)

// Target contains metadata about a target
type Target struct {
	Name   string
	Digest digest.Digest
	Size   int64
	Role   string
}

// AuxData contains metadata about the content that was pushed
type AuxData struct {
	Tag    string `json:"Tag"`
	Digest string `json:"Digest"`
	Size   int64  `json:"Size"`
}

// Push parses a given name into registry/namespace/image:tag and attempts to push it to the remote registry
func (d *SImages) Push(name string, user *models.User, env *models.Environment, ip prompts.IPrompts) (*models.Image, error) {
	ctx := context.Background()
	dockerCli, err := dockerClient.NewEnvClient()
	if err != nil {
		return nil, err
	}
	defer dockerCli.Close()
	dockerCli.NegotiateAPIVersion(ctx)

	repositoryName, tag, err := d.GetGloballyUniqueNamespace(name, env, true)
	if err != nil {
		return nil, err
	}

	if tag == "" {
		tag = DefaultTag
	}
	fullImageName := strings.Join([]string{repositoryName, tag}, ":")
	if fullImageName != name {
		if err = dockerCli.ImageTag(ctx, name, fullImageName); err != nil {
			if !strings.Contains(err.Error(), ImageDoesNotExist) {
				return nil, err
			}

			// Check if the fully formatted repo name exists, and ask if user wants to push that instead
			if !localImageExists(ctx, fullImageName, dockerCli) {
				return nil, err
			} else if yesNo := ip.YesNo(err.Error(), fmt.Sprintf("Would you like to push %s instead? (y/n) ", fullImageName)); yesNo != nil {
				return nil, fmt.Errorf(CancelingPush)
			}
		} else {
			logrus.Printf("Pushing image %s to %s\n", name, fullImageName)
		}
	} else {
		logrus.Printf("Pushing image %s", fullImageName)
	}

	resp, err := dockerCli.ImagePush(ctx, fullImageName, types.ImagePushOptions{RegistryAuth: dockerAuth(user)})
	if err != nil {
		return nil, err
	}
	defer resp.Close()

	var digest models.ContentDigest
	if err = jsonmessage.DisplayJSONMessagesStream(resp, os.Stdout, os.Stdout.Fd(), true,
		func(aux *json.RawMessage) {
			var auxData AuxData
			if data, jsonErr := aux.MarshalJSON(); jsonErr == nil {
				json.Unmarshal(data, &auxData)
				hashParts := strings.Split(auxData.Digest, ":")
				digest.HashType = hashParts[0]
				digest.Hash = hashParts[1]
				digest.Size = auxData.Size
			}
		}); err != nil {
		return nil, err
	}

	return &models.Image{
		Name:   repositoryName,
		Tag:    tag,
		Digest: &digest,
	}, nil
}

// Pull parses a name into registry/namespace/image:tag and attempts to retrieve it from the remote registry
func (d *SImages) Pull(name string, target *Target, user *models.User, env *models.Environment) error {
	ctx := context.Background()
	dockerCli, err := dockerClient.NewEnvClient()
	if err != nil {
		return err
	}
	defer dockerCli.Close()
	dockerCli.NegotiateAPIVersion(ctx)

	ref := strings.Join([]string{name, string(target.Digest)}, "@")
	resp, err := dockerCli.ImagePull(ctx, ref, types.ImagePullOptions{RegistryAuth: dockerAuth(user)})
	if err != nil {
		return err
	}
	defer resp.Close()
	return jsonmessage.DisplayJSONMessagesStream(resp, os.Stdout, os.Stdout.Fd(), true, nil)
}

// InitNotaryRepo intializes a notary repository
func (d *SImages) InitNotaryRepo(repo notaryClient.Repository, rootKeyPath string) error {
	rootTrustDir := fmt.Sprintf("%s/%s", userHomeDir(), trustPath)
	if err := os.MkdirAll(rootTrustDir, 0700); err != nil {
		return err
	}

	rootKeyIDs, err := getRootKey(rootKeyPath, repo, getPassphraseRetriever())
	if err != nil {
		return err
	}
	if err = repo.Initialize(rootKeyIDs, data.CanonicalSnapshotRole); err != nil {
		return err
	}
	return nil
}

// AddTargetHash adds the given content hash to a notary repo and sends a signing request to the server
func (d *SImages) AddTargetHash(repo notaryClient.Repository, digest *models.ContentDigest, tag string, publish bool) error {
	targetHash := data.Hashes{}
	sha256, err := hex.DecodeString(digest.Hash)
	if err != nil {
		return err
	}
	targetHash[digest.HashType] = sha256

	// var targetCustom *canonicalJson.RawMessage
	target := &notaryClient.Target{Name: tag, Hashes: targetHash, Length: digest.Size}
	if err = repo.AddTarget(target, data.CanonicalTargetsRole); err != nil {
		return err
	}
	if publish {
		return d.Publish(repo)
	}
	return nil
}

// ListTargets intializes a notary repository
func (d *SImages) ListTargets(repo notaryClient.Repository, roles ...string) ([]*Target, error) {
	targets, err := repo.ListTargets(data.NewRoleList(roles)...)
	if err != nil {
		return nil, err
	}
	var ts []*Target
	for _, t := range targets {
		t1, err := convertTarget(t)
		if err != nil {
			return nil, err
		}
		ts = append(ts, t1)
	}
	return ts, nil
}

// LookupTarget searches for a specific target in a repository by tag name
func (d *SImages) LookupTarget(repo notaryClient.Repository, tag string) (*Target, error) {
	target, err := repo.GetTargetByName(tag)
	if err != nil {
		return nil, err
	}
	role := string(target.Role)
	if role != path.Join(CanonicalTargetsRole, "releases") && role != CanonicalTargetsRole {
		return nil, fmt.Errorf("no canonical target found for %s", tag)
	}
	t, err := convertTarget(target)
	if err != nil {
		return nil, err
	}
	return t, nil
}

// DeleteTargets deletes the signed targets for a list of tags
func (d *SImages) DeleteTargets(repo notaryClient.Repository, tags []string, publish bool) error {
	//TODO: Check if a target is associated with a deployed release before allowing it to be unsigned
	for _, tag := range tags {
		if err := repo.RemoveTarget(tag, "targets"); err != nil {
			return err
		}
	}

	if publish {
		return d.Publish(repo)
	}
	return nil
}

// PrintChangelist prints out the users unpublished changes in a formatted table
func (d *SImages) PrintChangelist(changes []changelist.Change) {
	data := [][]string{[]string{"#", "Action", "Scope", "Type", "Target"}, []string{"-", "------", "-----", "----", "------"}}
	data = append(data)
	for i, c := range changes {
		data = append(data, []string{fmt.Sprintf("%d\n", i), c.Action(), c.Scope().String(), c.Type(), c.Path()})
	}

	logrus.Println()
	table := tablewriter.NewWriter(logrus.StandardLogger().Out)
	table.SetBorder(false)
	table.SetRowLine(false)
	table.SetHeaderLine(false)
	table.SetAlignment(1)
	table.SetCenterSeparator("")
	table.SetColumnSeparator("")
	table.SetRowSeparator("")
	table.AppendBulk(data)
	table.Render()
	logrus.Println()
}

// CheckChangelist prompts the user if they have unpublished changes to let them clear
//	undesired changes before publishing, or verify that the changes should be published
func (d *SImages) CheckChangelist(repo notaryClient.Repository, ip prompts.IPrompts) error {
	if changelist, err := repo.GetChangelist(); err == nil {
		changes := changelist.List()
		if len(changes) > 0 {
			logrus.Println("The following unpublished changes were found in your local trust repository:")
			d.PrintChangelist(changes)
			publishWarning := "These changes will be published along with your current request."
			if err = ip.YesNo(publishWarning, "Would you like to proceed? (y/n) "); err != nil {
				logrus.Println("Use the `images targets reset <image>` command to clear undesired changes")
				return err
			}
		}
	}
	return nil
}

// GetNotaryRepository returns a pointer to the notary repository for an image
func (d *SImages) GetNotaryRepository(pod, imageName string, user *models.User) notaryClient.Repository {
	notaryServer := getServer(pod, notaryServers)
	transport, err := getTransport(imageName, notaryServer, user, readWrite)
	if err != nil {
		logrus.Fatalln(err)
	}
	rootTrustDir := fmt.Sprintf("%s/%s", userHomeDir(), trustPath)
	repo, err := notaryClient.NewFileCachedRepository(
		rootTrustDir,
		data.GUN(imageName),
		notaryServer,
		transport,
		getPassphraseRetriever(),
		trustpinning.TrustPinConfig{},
	)
	if err != nil {
		logrus.Fatalln(err)
	}
	return repo
}

// GetGloballyUniqueNamespace returns the fully formatted name for an image <registry>/<namespace>/<image> and a tag if present
func (d *SImages) GetGloballyUniqueNamespace(name string, env *models.Environment, includeRegistry bool) (string, string, error) {
	var repositoryName string
	var image string
	var tag string

	imageParts := strings.Split(strings.TrimPrefix(name, "/"), ":")
	switch len(imageParts) {
	case 1:
		image = imageParts[0]
	case 2:
		image = imageParts[0]
		tag = imageParts[1]
	default:
		return "", "", fmt.Errorf(InvalidImageName)
	}

	repoParts := strings.Split(image, "/")
	registry := getServer(env.Pod, registries)

	var repo string
	switch len(repoParts) {
	case 1:
		repo = repoParts[0]
	case 2:
		if repoParts[0] != env.Namespace {
			if repoParts[0] != registry {
				return "", "", fmt.Errorf(IncorrectNamespace)
			}
			//Allow users to pull public images
			return image, tag, nil
		}
		repo = repoParts[1]
	case 3:
		if repoParts[0] != registry || repoParts[1] != env.Namespace {
			return "", "", fmt.Errorf(IncorrectRegistryOrNamespace)
		}
		repo = repoParts[2]
	default:
		return "", "", fmt.Errorf(InvalidImageName)
	}
	if includeRegistry {
		repositoryName = fmt.Sprintf("%s/%s/%s", registry, env.Namespace, repo)
	} else {
		repositoryName = fmt.Sprintf("%s/%s", env.Namespace, repo)
	}
	return repositoryName, tag, nil
}

// Publish publishes changes to a repo
func (d *SImages) Publish(repo notaryClient.Repository) error {
	if err := repo.Publish(); err != nil {
		return err
	}
	return nil
}

// dockerAuth returns a sessionized auth string for registry and notary requests
func dockerAuth(user *models.User) string {
	authConfig := types.AuthConfig{
		Username: user.Email,
		Password: fmt.Sprintf("SessionToken=%s", user.SessionToken),
	}
	encodedJSON, err := json.Marshal(authConfig)
	if err != nil {
		logrus.Fatal(err)
	}
	return base64.URLEncoding.EncodeToString(encodedJSON)
}

func localImageExists(ctx context.Context, fullImageName string, dockerCli *dockerClient.Client) bool {
	imageList, err := dockerCli.ImageList(ctx, types.ImageListOptions{All: true, Filters: filters.NewArgs(filters.Arg("reference", fullImageName))})
	if err != nil {
		return false
	}
	return len(imageList) > 0
}

func getServer(pod string, serverMap map[string]string) string {
	if server, ok := serverMap[pod]; ok {
		return server
	}
	return serverMap["default"]
}

func getPassphraseRetriever() notary.PassRetriever {
	baseRetriever := passphrase.PromptRetriever()
	env := map[string]string{
		"root":       os.Getenv("NOTARY_ROOT_PASSPHRASE"),
		"targets":    os.Getenv("NOTARY_TARGETS_PASSPHRASE"),
		"snapshot":   os.Getenv("NOTARY_SNAPSHOT_PASSPHRASE"),
		"delegation": os.Getenv("NOTARY_DELEGATION_PASSPHRASE"),
	}

	return func(keyName string, alias string, createNew bool, numAttempts int) (string, bool, error) {
		if v := env[alias]; v != "" {
			return v, numAttempts > 1, nil
		}
		// For delegation roles, we can also try the "delegation" alias if it is specified
		// Note that we don't check if the role name is for a delegation to allow for names like "user"
		// since delegation keys can be shared across repositories
		// This cannot be a base role or imported key, though.
		if v := env["delegation"]; !data.IsBaseRole(data.RoleName(alias)) && v != "" {
			return v, numAttempts > 1, nil
		}
		return baseRetriever(keyName, alias, createNew, numAttempts)
	}
}

func getRootKey(rootKeyPath string, repo notaryClient.Repository, retriever notary.PassRetriever) ([]string, error) {
	var rootKeyList []string
	cryptoService := repo.GetCryptoService()
	if rootKeyPath != "" {
		privKey, err := readRootKey(rootKeyPath, retriever)
		if err != nil {
			return nil, err
		}
		err = cryptoService.AddKey(data.CanonicalRootRole, "", privKey)
		if err != nil {
			return nil, fmt.Errorf("Error importing key: %v", err)
		}
		rootKeyList = []string{privKey.ID()}
	} else {
		rootKeyList = cryptoService.ListKeys(data.CanonicalRootRole)
	}

	if len(rootKeyList) < 1 {
		logrus.Println("No root keys found. Generating a new root key...")
		rootPublicKey, err := cryptoService.Create(data.CanonicalRootRole, "", data.ECDSAKey)
		if err != nil {
			return nil, err
		}
		rootKeyList = []string{rootPublicKey.ID()}
	} else {
		// Chooses the first root key available, which is initialization specific
		// but should return the HW one first.
		logrus.Printf("Root key found, using: %s\n", rootKeyList[0])
		rootKeyList = rootKeyList[0:1]
	}

	return rootKeyList, nil
}

// Attempt to read a role key from a file, and return it as a data.PrivateKey
// Root key must be encrypted
func readRootKey(rootKeyPath string, retriever notary.PassRetriever) (data.PrivateKey, error) {
	keyFile, err := os.Open(rootKeyPath)
	if err != nil {
		return nil, fmt.Errorf("Opening file to import as a root key: %v", err)
	}
	defer keyFile.Close()

	pemBytes, err := ioutil.ReadAll(keyFile)
	if err != nil {
		return nil, fmt.Errorf("Error reading input root key file: %v", err)
	}
	if err = cryptoservice.CheckRootKeyIsEncrypted(pemBytes); err != nil {
		return nil, err
	}

	privKey, _, err := trustmanager.GetPasswdDecryptBytes(retriever, pemBytes, "", data.CanonicalRootRole.String())
	if err != nil {
		return nil, err
	}

	return privKey, nil
}

type httpAccess int

const (
	readOnly httpAccess = iota
	readWrite
	admin
)

func getTransport(gun, notaryServer string, user *models.User, permission httpAccess) (http.RoundTripper, error) {
	tlsConfig, err := tlsconfig.Client(tlsconfig.Options{
		CAFile:             "",
		InsecureSkipVerify: false,
		CertFile:           "",
		KeyFile:            "",
	})
	if err != nil {
		return nil, fmt.Errorf("unable to configure TLS: %s", err.Error())
	}

	base := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		Dial: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
			DualStack: true,
		}).Dial,
		TLSHandshakeTimeout: 10 * time.Second,
		TLSClientConfig:     tlsConfig,
		DisableKeepAlives:   true,
	}
	return tokenAuth(notaryServer, base, gun, user, permission)
}

func tokenAuth(trustServerURL string, baseTransport *http.Transport, gun string, user *models.User, permission httpAccess) (http.RoundTripper, error) {
	authTransport := transport.NewTransport(baseTransport)
	pingClient := &http.Client{
		Transport: authTransport,
		Timeout:   5 * time.Second,
	}
	endpoint, err := url.Parse(trustServerURL)
	if err != nil {
		return nil, fmt.Errorf("Could not parse remote trust server url (%s): %s", trustServerURL, err.Error())
	}
	if endpoint.Scheme == "" {
		return nil, fmt.Errorf("Trust server url has to be in the form of http(s)://URL:PORT. Got: %s", trustServerURL)
	}
	subPath, err := url.Parse("v2/")
	if err != nil {
		return nil, fmt.Errorf("Failed to parse v2 subpath. This error should not have been reached. Please report it as an issue at https://github.com/docker/notary/issues: %s", err.Error())
	}
	endpoint = endpoint.ResolveReference(subPath)
	req, err := http.NewRequest("GET", endpoint.String(), nil)
	if err != nil {
		return nil, err
	}
	resp, err := pingClient.Do(req)
	if err != nil {
		logrus.Errorf("could not reach %s: %s", trustServerURL, err.Error())
		logrus.Info("continuing in offline mode")
		return nil, nil
	}
	// non-nil err means we must close body
	defer resp.Body.Close()
	if (resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices) &&
		resp.StatusCode != http.StatusUnauthorized {
		// If we didn't get a 2XX range or 401 status code, we're not talking to a notary server.
		// The http client should be configured to handle redirects so at this point, 3XX is
		// not a valid status code.
		logrus.Errorf("could not reach %s: %d", trustServerURL, resp.StatusCode)
		logrus.Info("continuing in offline mode")
		return nil, nil
	}

	// challengeManager := auth.NewSimpleChallengeManager()
	challengeManager := challenge.NewSimpleManager()
	if err := challengeManager.AddResponse(resp); err != nil {
		return nil, err
	}

	// ps := passwordStore{anonymous: permission == readOnly}
	creds := credentials{
		username:     user.Email,
		sessionToken: user.SessionToken,
	}

	var actions []string
	switch permission {
	case admin:
		actions = []string{"*"}
	case readWrite:
		actions = []string{"push", "pull"}
	case readOnly:
		actions = []string{"pull"}
	default:
		return nil, fmt.Errorf("Invalid permission requested for token authentication of gun %s", gun)
	}

	tokenHandler := auth.NewTokenHandler(authTransport, creds, gun, actions...)
	basicHandler := auth.NewBasicHandler(creds)

	modifier := auth.NewAuthorizer(challengeManager, tokenHandler, basicHandler)

	if permission != readOnly {
		return newAuthRoundTripper(transport.NewTransport(baseTransport, modifier)), nil
	}

	// Try to authenticate read only repositories using basic username/password authentication
	return newAuthRoundTripper(transport.NewTransport(baseTransport, modifier),
		transport.NewTransport(baseTransport, auth.NewAuthorizer(challengeManager, auth.NewTokenHandler(authTransport, creds, gun, actions...)))), nil
}

// authRoundTripper tries to authenticate the requests via multiple HTTP transactions (until first succeed)
type authRoundTripper struct {
	trippers []http.RoundTripper
}

func newAuthRoundTripper(trippers ...http.RoundTripper) http.RoundTripper {
	return &authRoundTripper{trippers: trippers}
}

func (a *authRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	var resp *http.Response
	// Try all run all transactions
	for _, t := range a.trippers {
		var err error
		resp, err = t.RoundTrip(req)
		// Reject on error
		if err != nil {
			return resp, err
		}

		// Stop when request is authorized/unknown error
		if resp.StatusCode != http.StatusUnauthorized {
			return resp, nil
		}
	}

	// Return the last response
	return resp, nil
}

type credentials struct {
	username     string
	sessionToken string
	refreshToken string
}

func (c credentials) Basic(url *url.URL) (string, string) {
	return c.username, fmt.Sprintf("SessionToken=%s", c.sessionToken)
}

func (c credentials) RefreshToken(url *url.URL, service string) string {
	return c.refreshToken
}

func (c credentials) SetRefreshToken(realm *url.URL, service, token string) {
	c.refreshToken = token
}

func userHomeDir() string {
	env := "HOME"
	if runtime.GOOS == "windows" {
		env = "USERPROFILE"
	} else if runtime.GOOS == "plan9" {
		env = "home"
	}
	return os.Getenv(env)
}

func convertTarget(t *notaryClient.TargetWithRole) (*Target, error) {
	h, ok := t.Hashes["sha256"]
	if !ok {
		return nil, fmt.Errorf("no valid hash, expecting sha256")
	}
	return &Target{
		Name:   t.Name,
		Digest: digest.NewDigestFromHex("sha256", hex.EncodeToString(h)),
		Size:   t.Length,
		Role:   string(t.Role),
	}, nil
}
