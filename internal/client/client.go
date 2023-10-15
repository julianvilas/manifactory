package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

const (
	catalogScope    = "registry:catalog:*"
	repositoryScope = "repository:*:pull"
)

var (
	accept = []string{
		"application/vnd.docker.distribution.manifest.v2+json",
	}
)

type Options struct {
	Insecure  bool
	Timeout   time.Duration
	BasicAuth bool
}

type Client struct {
	registry   string
	user, pass string

	creds     map[string]credentials
	http      *http.Client
	basicAuth bool
}

type credentials struct {
	scope string
	token string
}

func New(registry, user, pass string, opts Options) *Client {
	creds := make(map[string]credentials)

	tr := http.DefaultTransport.(*http.Transport).Clone()
	tlsConf := tr.TLSClientConfig.Clone()
	tlsConf.InsecureSkipVerify = opts.Insecure
	tr.TLSClientConfig = tlsConf

	cli := &http.Client{
		Transport: tr,
		Timeout:   30 * time.Second,
	}
	if timeout := opts.Timeout; timeout != 0 {
		cli.Timeout = timeout
	}

	return &Client{
		registry:  registry,
		user:      user,
		pass:      pass,
		creds:     creds,
		http:      cli,
		basicAuth: opts.BasicAuth,
	}
}

func (cli *Client) Catalog() ([]string, error) {
	catalogEndpoint, err := url.JoinPath(cli.registry, "v2/_catalog")
	if err != nil {
		return nil, err
	}
	body, err := cli.request(catalogEndpoint, catalogScope, nil)
	if err != nil {
		return nil, err
	}
	catalogResponse := struct {
		Repositories []string `json:"repositories"`
	}{}
	if err := json.Unmarshal(body, &catalogResponse); err != nil {
		return nil, err
	}

	return catalogResponse.Repositories, nil
}

func (cli *Client) Tags(repository string) ([]string, error) {
	tagsEndpoint, err := url.JoinPath(cli.registry, fmt.Sprintf("v2/%s/tags/list/", repository))
	if err != nil {
		return nil, err
	}
	body, err := cli.request(tagsEndpoint, repositoryScope, nil)
	if err != nil {
		return nil, err
	}
	tagsResponse := struct {
		Name string   `json:"name"`
		Tags []string `json:"tags"`
	}{}
	if err := json.Unmarshal(body, &tagsResponse); err != nil {
		return nil, err
	}

	return tagsResponse.Tags, nil
}

type Manifest struct {
	SchemaVersion int    `json:"schemaVersion"`
	MediaType     string `json:"mediaType"`
	Config        struct {
		MediaType string `json:"mediaType"`
		Size      int    `json:"size"`
		Digest    string `json:"digest"`
	} `json:"config"`
	Layers []struct {
		MediaType string `json:"mediaType"`
		Size      int    `json:"size"`
		Digest    string `json:"digest"`
	} `json:"layers"`
}

func (cli *Client) Manifest(repository, tag string) (Manifest, error) {
	manifestsEndpoint, err := url.JoinPath(cli.registry, fmt.Sprintf("v2/%s/manifests/%s", repository, tag))
	if err != nil {
		return Manifest{}, err
	}
	body, err := cli.request(manifestsEndpoint, repositoryScope, accept)
	if err != nil {
		return Manifest{}, err
	}

	m := Manifest{}
	if err := json.Unmarshal(body, &m); err != nil {
		return Manifest{}, err
	}

	if strings.Contains(m.MediaType, "list") {
		log.Printf("manifest list in %s/%s", repository, tag)
		return Manifest{}, nil
	}

	return m, nil
}

func (cli *Client) request(endpoint, scope string, accept []string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}

	for _, header := range accept {
		req.Header.Add("Accept", header)
	}

	if cli.basicAuth {
		req.SetBasicAuth(cli.user, cli.pass)
	} else {
		cred, ok := cli.creds[scope]
		if !ok {
			token, err := cli.Token(scope)
			if err != nil {
				return nil, fmt.Errorf("can not retrieve token: %w", err)
			}
			cred = credentials{
				scope: scope,
				token: token,
			}
			cli.creds[scope] = cred
		}
		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", cred.token))
	}

	resp, err := cli.http.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status code: %d", resp.StatusCode)
	}
	defer resp.Body.Close()

	return io.ReadAll(resp.Body)
}

// Token generates a bearer token for the Docker Registry API (v2).  Reference:
// https://docs.docker.com/registry/spec/api/#api-version-check
func (cli *Client) Token(scope string) (string, error) {
	// Check that the registry API supports version 2.
	endpoint, err := url.JoinPath(cli.registry, "/v2/")
	if err != nil {
		return "", err
	}
	resp, err := cli.http.Get(endpoint)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusUnauthorized {
		return "", fmt.Errorf("status code: %d", resp.StatusCode)
	}

	versionH := resp.Header["Docker-Distribution-Api-Version"]
	found := false
	for _, v := range versionH {
		if v == "registry/2.0" {
			found = true
			break
		}
	}
	if !found {
		return "", errors.New("missing or unexpected registry version header")
	}

	// Request token to the auth service specified via authenticate header.
	re, err := regexp.Compile(`Bearer realm="(.+)",service="(.+)"`)
	if err != nil {
		return "", err
	}

	var realm string
	var service string
	authH := resp.Header["Www-Authenticate"]
	found = false
	for _, v := range authH {
		matches := re.FindStringSubmatch(v)
		if len(matches) == 3 {
			found = true
			realm = matches[1]
			service = matches[2]
			break
		}
	}
	if !found {
		return "", errors.New("missing or unexpected registry authentication header")
	}

	req, err := http.NewRequest("GET", realm, nil)
	if err != nil {
		return "", err
	}

	q := req.URL.Query()
	q.Add("service", service)
	q.Add("scope", scope)
	req.URL.RawQuery = q.Encode()

	req.SetBasicAuth(cli.user, cli.pass)

	resp, err = cli.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	t := struct {
		Token string
	}{}
	if err := json.Unmarshal(body, &t); err != nil {
		return "", err
	}

	return t.Token, nil
}
