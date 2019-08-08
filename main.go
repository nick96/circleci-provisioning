package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strconv"

	yaml "gopkg.in/yaml.v2"
)

// Config represents the configuration of a CircleCI project
type Config struct {
	VcsType     string            `yaml:"vcsType"`     // Type of VCS used (e.g. git)
	Owner       string            `yaml:"owner"`       // Project owner (e.g. user or org)
	ProjectName string            `yaml:"projectName"` // Project to be followed
	EnvVars     map[string]string `yaml:"envVars"`     // Env vars to set
	SSHKeys     map[string]string `yaml:"sshKeys"`     // SSH keys to add
}

func main() {
	tokenEnv := os.Getenv("CIRCLECI_TOKEN")
	configFileEnv := os.Getenv("CIRCLECI_CONFIG")
	isCanonicalEnv, err := strconv.ParseBool(os.Getenv("CIRCLECI_CANONICAL"))
	if err != nil {
		isCanonicalEnv = false
	}
	shouldTriggerEnv, err := strconv.ParseBool(os.Getenv("CIRCLECI_TRIGGER"))
	if err != nil {
		shouldTriggerEnv = false
	}
	shouldUnfollowEnv, err := strconv.ParseBool(os.Getenv("CIRCLECI_UNFOLLOW"))
	if err != nil {
		shouldUnfollowEnv = false
	}

	token := flag.String("token", tokenEnv, "Circle CI token")
	configFile := flag.String("config", configFileEnv, "Circle CI provisioning config")
	isCanonical := flag.Bool("canonical", isCanonicalEnv,
		"Project should be exactly as described in the config. "+
			" WARNING: This may remove environment variables and ssh keys")
	shouldTrigger := flag.Bool("trigger", shouldTriggerEnv, "Trigger a build of the project once it is setup")
	shouldUnfollow := flag.Bool("unfollow", shouldUnfollowEnv, "Unfollow the project")
	flag.Parse()

	if token == nil || *token == "" {
		log.Fatal("-token is required or CIRCLECI_TOKEN should be set")
	}

	if configFile == nil || *configFile == "" {
		log.Fatal("-config is required or CIRCLECI_CONFIG should be set")
	}

	config, err := readConfig(*configFile)
	if err != nil {
		log.Fatalf("Could not read config file %s: %v", *configFile, err)
	}

	project := NewCircleCIProject(config.VcsType, config.Owner, config.ProjectName, *token)

	if *shouldUnfollow {
		log.Printf("Unfollowing %s", project.FullName())
		project.Unfollow()
		return
	}

	log.Printf("Following %s", project.FullName())
	err = project.Follow()
	if err != nil {
		log.Fatalf("Error: Could not follow %s: %v", project.FullName(), err)
	}

	if *isCanonical {
		log.Printf("Making config %s canonical for project %s", *configFile, project.FullName())
		err = cleanProject(project)
		if err != nil {
			log.Fatalf("Error: Could not make config %s canonical for project %s: %v",
				*configFile, project.FullName(), err)
		}
	}

	log.Printf("Setting environment variables for project %s", project.FullName())
	err = setEnvVars(project, config.EnvVars)
	if err != nil {
		log.Fatalf("Error: Could not set environment variables for project %s: %v", project.FullName(), err)
	}

	log.Printf("Adding ssh keys for project %s", project.FullName())
	err = addSSHKeys(project, config.SSHKeys)
	if err != nil {
		log.Fatalf("Error: Could not add SSH Keys for project %s: %v", project.FullName(), err)
	}

	if *shouldTrigger {
		log.Printf("Triggering build of %s", project.FullName())
		err := project.Trigger()
		if err != nil {
			log.Fatalf("Error: Could not trigger build for project %s: %v", project.FullName(), err)
		}
	}

	log.Printf("Project %s has been successfully provisioned using %s", project.FullName(), *configFile)
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

func addSSHKeys(project Project, sshKeys map[string]string) error {
	for name, path := range sshKeys {
		fh, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("could not open SSH key at path %s: %v", path, err)
		}
		defer fh.Close()
		content, err := ioutil.ReadAll(fh)
		if err != nil {
			return fmt.Errorf("could not read SSH Key at path %s: %v", path, err)
		}
		err = project.AddSSHKey(name, string(content))
		if err != nil {
			return fmt.Errorf("could not add SSH key %s for project %s: %v", path, project.FullName(), err)
		}
	}
	return nil
}

func cleanProject(project Project) error {
	err := project.Clearenv()
	if err != nil {
		return fmt.Errorf("there was an error clearing environment variables from project %s: %v",
			project.FullName(), err)
	}

	err = project.ClearSSHKeys()
	if err != nil {
		return fmt.Errorf("there was an error clearing SSH keys from project %s: %v", project.FullName(), err)
	}
	return nil
}

func setEnvVars(project Project, envVars map[string]string) error {
	for k, v := range envVars {
		log.Printf("Setting environment variable %s for project %s", k, project.FullName())
		err := project.Setenv(k, v)
		if err != nil {
			return fmt.Errorf("could not set environment variable %s for project %s: %v",
				k, project.FullName(), err)
		}
	}
	return nil
}
