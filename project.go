package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
	"strings"

	"bytes"
)

// Project represents a project
type Project interface {
	FullName() string
	Follow() error
	Unfollow() error
	Setenv(name, value string) error
	Getenv(name string) (string, error)
	Getenvs() (map[string]string, error)
	Deleteenv(name string) error
	Clearenv() error
	AddSSHKey(name string, privateKey string) error
	GetSSHKeyFingerprint(name string) (string, error)
	RemoveSSHKey(name string) error
	ClearSSHKeys() error
	Trigger() error
}

type Client interface {
	BaseURL() string
	Get(url string) (*http.Response, error)
	Post(url, contentType string, body io.Reader) (*http.Response, error)
	Delete(url string) (*http.Response, error)
}

type CircleCIClient struct {
	baseURL string
	client *http.Client
}

// CircleCIProject represents a CircleCI project
type CircleCIProject struct {
	vcsType     string
	owner       string
	projectName string
	token       string
	client      Client
}

// NewCircleCIProject creates a Circle CI project representation.
func NewCircleCIProject(vcsType, owner, projectName, token string) *CircleCIProject {
	return &CircleCIProject{
		vcsType:     vcsType,
		owner:       owner,
		projectName: projectName,
		token:       token,
		client:      &CircleCIClient{"https://circleci.com/api/v1.1", &http.Client{}},
	}
}

// BaseURL gets the base URL for the client
func (c *CircleCIClient) BaseURL() string {
	return c.baseURL
}

func (c *CircleCIClient) do(method, url string, body io.Reader) (*http.Response, error) {
	if c.baseURL != "" && !strings.HasPrefix(url, c.baseURL) {
		url = path.Join(c.baseURL, url)
	}
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	return c.client.Do(req)
}

// Get performs a GET request
func (c *CircleCIClient) Get(url string) (*http.Response, error) {
	return c.do(http.MethodGet, url, nil)
}

// Post performs a POST request
func (c *CircleCIClient) Post(url, contentType string, body io.Reader) (*http.Response, error) {
	return c.do(http.MethodPost, url, body)
}

// Delete performs a DELETE request
func (c *CircleCIClient) Delete(url string) (*http.Response, error) {
	return c.do(http.MethodDelete, url, nil)
}

// fmtURI formats a URI to be used for Circle CI API requests.
func (p *CircleCIProject) fmtURI(resource, action string) string {
	url, _ := url.Parse(p.client.BaseURL())
	url.Path = path.Join(url.Path, resource, p.vcsType, p.owner, p.projectName, action)
	query := url.Query()
	query.Set("circle-token", p.token)
	url.RawQuery = query.Encode()
	return url.String()
}

// FullName returns the full name of the project
func (p *CircleCIProject) FullName() string {
	return fmt.Sprintf("%s/%s", p.owner, p.projectName)
}

// Follow follows the project
func (p *CircleCIProject) Follow() error {
	url := p.fmtURI("project", "follow")
	resp, err := p.client.Post(url, "", strings.NewReader(""))
	if err != nil {
		return fmt.Errorf("could not follow project %s: %v", p.FullName(), err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("error following project %s: expected status %d, found %d",
			p.FullName(), http.StatusCreated, resp.StatusCode)
	}
	return nil
}

// Unfollow unfollows the project.
func (p *CircleCIProject) Unfollow() error {
	url := p.fmtURI("project", "unfollow")
	resp, err := p.client.Post(url, "", strings.NewReader(""))
	if err != nil {
		return fmt.Errorf("could not unfollow project: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("expected status %d, found %d", http.StatusOK, resp.StatusCode)
	}

	return nil
}

// Setenv sets an environment variable in a project
func (p *CircleCIProject) Setenv(name, value string) error {
	url := p.fmtURI("project", "envvar")
	body := fmt.Sprintf(`{"name": "%s", "value": "%s"}`, name, value)
	resp, err := p.client.Post(url, "application/json", strings.NewReader(body))
	if err != nil {
		return fmt.Errorf("could not create environment variable %s: %v", name, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("environment variable %s not created: status %s", name, resp.Status)
	}
	return nil
}

// Clearenv removes all environment variables from a project.
func (p *CircleCIProject) Clearenv() error {
	envVars, err := p.Getenvs()
	if err != nil {
		return fmt.Errorf("could not clean environment variables for project %s: %v", p.FullName(), err)
	}

	for name := range envVars {
		err = p.Deleteenv(name)
		if err != nil {
			return fmt.Errorf("could not remove environment variable %s from project %s: %v",
				name, p.FullName(), err)
		}
	}
	return nil
}

// Getenv gets the named environment variable in a project.
func (p *CircleCIProject) Getenv(name string) (string, error) {
	return "", nil
}

// Getenvs gets all the environment variables in the project.
func (p *CircleCIProject) Getenvs() (map[string]string, error) {
	url := p.fmtURI("project", "envvar")
	resp, err := p.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("could not get environment variables for project %s: %v", p.FullName(), err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("could not get environment variables for project %s: %v", p.FullName(), err)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("could not read response body to get environment variables for project %s: %v",
			p.FullName(), err)
	}

	var results []struct {
		name  string
		value string
	}
	err = json.Unmarshal(body, &results)
	if err != nil {
		return nil, fmt.Errorf("could not unmarshall response body to get environment variables for project %s: %v",
			p.FullName(), err)
	}

	envVars := make(map[string]string)
	for _, result := range results {
		envVars[result.name] = result.value
	}

	return envVars, nil
}

// Deleteenv deletes the named environment variable in the project.
func (p *CircleCIProject) Deleteenv(name string) error {
	url := p.fmtURI("project", "envvar")
	resp, err := p.client.Delete(url)
	if err != nil {
		return fmt.Errorf("could not remove environment variable %s: %v", name, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("could not remove environment variable %s: %v", name, err)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("could not read response: %v", err)
	}

	var status struct{ message string }
	err = json.Unmarshal(body, &status)
	if err != nil {
		return fmt.Errorf("could not unmarshal response: %v", err)
	}

	if status.message != "ok" {
		return fmt.Errorf("failed to remove environment variable %s: expected status 'ok' but found '%s'",
			name, status.message)
	}

	return nil
}

// AddSSHKey adds an ssh key.
func (p *CircleCIProject) AddSSHKey(name, privateKey string) error {
	url := p.fmtURI("project", "ssh-key")
	postBody := struct {
		Hostname   string `json:"hostname"`
		PrivateKey string `json:"private_key"`
	}{
		Hostname:   name,
		PrivateKey: privateKey,
	}
	postBodyJSON, err := json.Marshal(postBody)

	resp, err := p.client.Post(url, "application/json", bytes.NewReader(postBodyJSON))
	if err != nil {
		return fmt.Errorf("could not add ssh key %s to project %s", name, p.FullName())
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("expected status code %d but received %d", http.StatusCreated, resp.StatusCode)
	}

	return nil
}

// GetSSHKeyFingerprint gets the fingerprint of the named SSH key.
func (p *CircleCIProject) GetSSHKeyFingerprint(name string) (string, error) {
	return "", fmt.Errorf("Not implemented")
}

// RemoveSSHKey removes the named SSH key from the project.
func (p *CircleCIProject) RemoveSSHKey(name string) error {
	return fmt.Errorf("Not implemented")
}

// Trigger triggers a build of the project
func (p *CircleCIProject) Trigger() error {
	url := p.fmtURI("project", "build")
	resp, err := p.client.Post(url, "", strings.NewReader(""))
	if err != nil {
		return fmt.Errorf("could not trigger build of project %s: %v", p.FullName(), err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("unexpected status code %d, expected %d", resp.StatusCode, http.StatusCreated)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %v", err)
	}

	var message struct {
		status int
		body   string
	}
	err = json.Unmarshal(body, &message)
	if err != nil {
		return fmt.Errorf("failed to unmarshal response body: %v", err)
	}

	if message.status != 200 {
		return fmt.Errorf("expected message status to be '200' but found %d", message.status)
	} else if message.body != "Build created" {
		return fmt.Errorf("expected message body to be 'Build created but found %s", message.body)
	}

	return nil
}

// ClearSSHKeys clears all SSH keys for the project.
func (p *CircleCIProject) ClearSSHKeys() error {
	return fmt.Errorf("Not implemented")
}
