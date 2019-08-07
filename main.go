package main

import (
	"flag"
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
)

// Config represents the configuration of a CircleCI project
type Config struct {
	VcsType     string            `yaml:"vcsType"`     // Type of VCS used (e.g. git)
	Owner       string            `yaml:"owner"`       // Project owner (e.g. user or org)
	ProjectName string            `yaml:"projectName"` // Project to be followed
	EnvVars     map[string]string `yaml:"envVars"`     // Env vars to set
	SSHKeys     []string          `yaml:"sshKeys"`     // SSHKeys to add
}

// Project represents a CircleCI project
type Project struct {
	VcsType     string
	Owner       string
	ProjectName string
}

func main() {
	tokenEnv := os.Getenv("CIRCLECI_TOKEN")
	configFileEnv := os.Getenv("CIRCLECI_CONFIG")
	shouldUnfollowEnv, err := strconv.ParseBool(os.Getenv("CIRCLECI_UNFOLLOW"))
	if err != nil {
		shouldUnfollowEnv = false
	}
	isCanonicalEnv, err := strconv.ParseBool(os.Getenv("CIRCLECI_CANONICAL"))
	if err != nil {
		isCanonicalEnv = false
	}
	shouldTriggerEnv, err := strconv.ParseBool(os.Getenv("CIRCLECI_TRIGGER"))
	if err != nil {
		shouldTriggerEnv = false
	}

	token := flag.String("token", tokenEnv, "Circle CI token")
	configFile := flag.String("config", configFileEnv, "Circle CI provisioning config")
	shouldUnfollow := flag.Bool("unfollow", shouldUnfollowEnv, "Unfollow the project described in the config")
	isCanonical := flag.Bool("canonical", isCanonicalEnv,
		"Project should be exactly as described in the config. "+
			" WARNING: This may remove environment variables and ssh keys")
	shouldTrigger := flag.Bool("trigger", shouldTriggerEnv, "Trigger a build of the project once it is setup")
	flag.Parse()

	if token == nil {
		log.Fatal("-token is required")
	}

	if configFile == nil {
		log.Fatal("-config is required")
	}

	config, err := readConfig(*configFile)
	if err != nil {
		log.Fatalf("Could not read config file %s: %v", *configFile, err)
	}

	project := Project{config.VcsType, config.Owner, config.ProjectName}

	if *shouldUnfollow {
		log.Printf("Unfollowing %s", project.String())
		err := unfollowProject(*token, project)
		if err != nil {
			log.Fatalf("Error: Could not unfollow %s: %v", project.String(), err)
		}
		return
	}

	log.Printf("Following %s", project.String())
	err = followProject(*token, project)
	if err != nil {
		log.Fatalf("Error: Could not follow %s: %v", project, err)
	}

	log.Printf("Setting environment variables for project %s", project)
	if *isCanonical {
		log.Printf("Project config is canonical, removing all environment variables currently set")
		cleanEnvVars(*token, project)
	}
	for k, v := range config.EnvVars {
		log.Printf("Setting environment variable %s for project %s", k, project.String())
		err := setEnvVar(*token, project, k, v)
		if err != nil {
			log.Fatalf("Error: Could not set environment variable %s for project %s: %v", k, project.String(), err)
		}
	}

	log.Printf("Adding ssh keys for project %s", project)
	for _, path := range config.SSHKeys {
		log.Printf("Adding ssh key %s for project %s", path, project)
		if err != nil {
			log.Fatalf("Error: Could not add SSH key %s for project %s")
		}
	}

	if *shouldTrigger {
		log.Printf("Triggering build of %s", project.String())
	}
}

func readConfig(configFile string) (Config, error) {
	config := Config{}
	fh, err := os.Open(configFile)
	if err != nil {
		return config, err
	}
	defer fh.Close()

	data, err := ioutil.ReadAll(fh)
	if err != nil {
		return config, fmt.Errorf("could not read %s: %v", configFile, err)
	}
	err = yaml.Unmarshal([]byte(data), &config)
	if err != nil {
		return config, fmt.Errorf("could not unmarshal %s: %v", configFile, err)
	}

	return config, nil
}

func unfollowProject(token string, project Project) error {
	return nil
}

func followProject(token string, project Project) error {
	url := fmt.Sprintf("https://circleci.com/api/v1.1/project/%s/%s/%s/follow?circle-token=%s", project.VcsType, project.Owner, project.ProjectName, token)
	resp, err := http.Post(url, "", strings.NewReader(""))
	if err != nil {
		return fmt.Errorf("could not follow project %s: %v", project.String(), err)
	}

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("could not follow project %s: %v", project.String(), err)
	}
	return nil
}

func setEnvVar(token string, project Project, name, value string) error {
	url := fmt.Sprintf("https://circleci.com/api/v1.1/project/%s/%s/%s/envvar?circle-token=%s", project.VcsType, project.Owner, project.ProjectName, token)
	body := fmt.Sprintf(`{"name": "%s", "value": "%s"}`, name, value)
	resp, err := http.Post(url, "application/json", strings.NewReader(body))
	if err != nil {
		return fmt.Errorf("could not create environment variable %s: %v", name, err)
	}

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("environment variable %s not created: status %s", name, resp.Status)
	}
	return nil
}

func cleanEnvVars(token string, project Project) error {
	return nil
}

func (p Project) String() string {
	return fmt.Sprintf("%s/%s", p.Owner, p.ProjectName)
}
