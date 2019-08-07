package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
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
	Trigger() error
}

// CircleCIProject represents a CircleCI project
type CircleCIProject struct {
	vcsType     string
	owner       string
	projectName string
	token       string
	client      http.Client
}

func NewCircleCIProject(vcsType, owner, projectName, token string) *CircleCIProject {
	return &CircleCIProject{
		vcsType:     vcsType,
		owner:       owner,
		projectName: projectName,
		token:       token,
		client:      http.Client{},
	}
}

// FullName returns the full name of the project
func (p *CircleCIProject) FullName() string {
	return fmt.Sprintf("%s/%s", p.owner, p.projectName)
}

// Follow follows the project
func (p *CircleCIProject) Follow() error {
	url := fmt.Sprintf("https://circleci.com/api/v1.1/project/%s/%s/%s/follow?circle-token=%s",
		p.vcsType, p.owner, p.projectName, p.token)
	resp, err := http.Post(url, "", strings.NewReader(""))
	if err != nil {
		return fmt.Errorf("could not follow project %s: %v", p.FullName(), err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("could not follow project %s: %v", p.FullName(), err)
	}
	return nil
}

func (p *CircleCIProject) Setenv(name, value string) error {
	url := fmt.Sprintf("https://circleci.com/api/v1.1/project/%s/%s/%s/envvar?circle-token=%s",
		p.vcsType, p.owner, p.projectName, p.token)
	body := fmt.Sprintf(`{"name": "%s", "value": "%s"}`, name, value)
	resp, err := http.Post(url, "application/json", strings.NewReader(body))
	if err != nil {
		return fmt.Errorf("could not create environment variable %s: %v", name, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("environment variable %s not created: status %s", name, resp.Status)
	}
	return nil
}

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

func (p *CircleCIProject) Getenv(name string) (string, error) {
	return "", nil
}

func (p *CircleCIProject) Getenvs() (map[string]string, error) {
	url := fmt.Sprintf("https://circleci.com/api/v1.1/project/%s/%s/%s/envvar?circle-token=%s",
		p.vcsType, p.owner, p.projectName, p.token)
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("could not get environment variables for project %s: %v", p.FullName(), err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("could not get environment variables for project %s: %v", p.FullName(), err)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("could not read response body to get environment variables for project %s",
			p.FullName(), err)
	}

	var results []struct {
		name  string
		value string
	}
	err = json.Unmarshal(body, &results)
	if err != nil {
		return nil, fmt.Errorf("could not unmarshall response body to get environment variables for project %s",
			p.FullName(), err)
	}

	envVars := make(map[string]string)
	for _, result := range results {
		envVars[result.name] = result.value
	}

	return envVars, nil
}

func (p *CircleCIProject) Deleteenv(name string) error {
	url := fmt.Sprintf("https://circleci.com/api/v1.1/project/%s/%s/%s/envvar?circle-token=%s",
		p.vcsType, p.owner, p.projectName, p.token)
	client := http.Client{}
	req, err := http.NewRequest(http.MethodDelete, url, strings.NewReader(""))
	if err != nil {
		return fmt.Errorf("could not remove environment variable %s: %v", name, err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("could not remove environment variable %s: %v", name, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("could not remove environment variable %s: %v", name, err)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("could not read response: %v", name, err)
	}

	var status struct{ message string }
	err = json.Unmarshal(body, &status)
	if err != nil {
		return fmt.Errorf("could not unmarshal response: %v", name, err)
	}

	if status.message != "ok" {
		return fmt.Errorf("failed to remove environment variable %s: expected status 'ok' but found '%s'",
			name, status.message)
	}

	return nil
}

func (p *CircleCIProject) AddSSHKey(name, privateKey string) error {
	url := fmt.Sprintf("https://circleci.com/api/v1.1/project/%s/%s/%s/ssh-key?circle-token=%s",
		p.vcsType, p.owner, p.projectName, p.token)
	postBody := struct {
		hostname   string
		privateKey string `json:"private_key"`
	}{
		hostname:   name,
		privateKey: privateKey,
	}
	postBodyJSON, err := json.Marshal(postBody)

	resp, err := http.Post(url, "application/json", bytes.NewReader(postBodyJSON))
	if err != nil {
		return fmt.Errorf("could not add ssh key %s to project %s", name, p.FullName())
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("expected status code %d but received %d", http.StatusCreated, resp.StatusCode)
	}

	return nil
}

func (p *CircleCIProject) GetSSHKeyFingerprint(name string) (string, error) {
	return "", fmt.Errorf("Not implemented")
}

func (p *CircleCIProject) RemoveSSHKey(name string) error {
	return fmt.Errorf("Not implemented")
}

func (p *CircleCIProject) Trigger() error {
	url := fmt.Sprintf("https://circleci.com/api/v1.1/project/%s/%s/%s/build?circle-token=%s",
		p.vcsType, p.owner, p.projectName, p.token)
	resp, err := http.Post(url, "", strings.NewReader(""))
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
