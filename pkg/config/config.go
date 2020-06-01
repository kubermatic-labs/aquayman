package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/kubermatic-labs/aquayman/pkg/quay"
	"github.com/kubermatic-labs/aquayman/pkg/util"
)

type Config struct {
	Organization string             `yaml:"organization"`
	Teams        []TeamConfig       `yaml:"teams,omitempty"`
	Repositories []RepositoryConfig `yaml:"repositories,omitempty"`
	Robots       []RobotConfig      `yaml:"robots,omitempty"`
}

type TeamConfig struct {
	Name        string        `yaml:"name"`
	Role        quay.TeamRole `yaml:"role"`
	Description string        `yaml:"description,omitempty"`
	Members     []string      `yaml:"members,omitempty"`
}

type RepositoryConfig struct {
	Name        string                         `yaml:"name"`
	Visibility  quay.RepositoryVisibility      `yaml:"visibility"`
	Description string                         `yaml:"description,omitempty"`
	Teams       map[string]quay.RepositoryRole `yaml:"teams,omitempty"`
	Users       map[string]quay.RepositoryRole `yaml:"users,omitempty"`
}

func (c *RepositoryConfig) IsWildcard() bool {
	return strings.Contains(c.Name, "*")
}

type RobotConfig struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description,omitempty"`
}

func LoadFromFile(filename string) (*Config, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	config := &Config{
		Teams:        []TeamConfig{},
		Repositories: []RepositoryConfig{},
	}

	if err := yaml.NewDecoder(f).Decode(config); err != nil {
		return nil, err
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}

	return config, nil
}

func SaveToFile(config *Config, filename string) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	if err := yaml.NewEncoder(f).Encode(config); err != nil {
		return err
	}

	return nil
}

func validTeamRole(role quay.TeamRole) bool {
	for _, r := range quay.AllTeamRoles {
		if r == role {
			return true
		}
	}

	return false
}

func validRepositoryRole(role quay.RepositoryRole) bool {
	for _, r := range quay.AllRepositoryRoles {
		if r == role {
			return true
		}
	}

	return false
}

func (c *Config) Validate() error {
	if c.Organization == "" {
		return errors.New("no organization configured")
	}

	teamNames := []string{}

	for _, team := range c.Teams {
		if util.StringSliceContains(teamNames, team.Name) {
			return fmt.Errorf("duplicate team %q defined", team.Name)
		}

		if !validTeamRole(team.Role) {
			return fmt.Errorf("role for team %q is invalid (%q), must be one of %v", team.Name, team.Role, quay.AllTeamRoles)
		}

		teamNames = append(teamNames, team.Name)
	}

	repoNames := []string{}
	visibilities := []string{
		string(quay.Public),
		string(quay.Private),
	}

	for _, repo := range c.Repositories {
		if util.StringSliceContains(repoNames, repo.Name) {
			return fmt.Errorf("duplicate repository %q defined", repo.Name)
		}

		if !util.StringSliceContains(visibilities, string(repo.Visibility)) {
			return fmt.Errorf("invalid visibility %q for repository %q, must be one of %v", repo.Visibility, repo.Name, visibilities)
		}

		for teamName, roleName := range repo.Teams {
			if !validRepositoryRole(roleName) {
				return fmt.Errorf("role for team %s in repo %q is invalid (%q), must be one of %v", teamName, repo.Name, roleName, quay.AllRepositoryRoles)
			}
		}

		for userName, roleName := range repo.Users {
			if !validRepositoryRole(roleName) {
				return fmt.Errorf("role for user %s in repo %q is invalid (%q), must be one of %v", userName, repo.Name, roleName, quay.AllRepositoryRoles)
			}
		}

		repoNames = append(repoNames, repo.Name)
	}

	robotNames := []string{}
	robotPattern := regexp.MustCompile(`^[a-z][a-z0-9_]{1,254}$`)
	prefix := c.Organization + "+"

	for _, robot := range c.Robots {
		if util.StringSliceContains(robotNames, robot.Name) {
			return fmt.Errorf("duplicate robot %q defined", robot.Name)
		}

		if strings.HasPrefix(robot.Name, prefix) {
			return fmt.Errorf("robot %q must be given as a short name, without the organization prefix (must be \"%s\")", robot.Name, strings.TrimPrefix(robot.Name, prefix))
		}

		if !robotPattern.MatchString(robot.Name) {
			return fmt.Errorf("robot %q has an invalid name, must be alphanumeric lowercase", robot.Name)
		}

		robotNames = append(robotNames, robot.Name)
	}

	return nil
}

func (c *Config) GetRepositoryConfig(repo string) *RepositoryConfig {
	// first try: exact match
	for idx, r := range c.Repositories {
		if r.Name == repo {
			return &c.Repositories[idx]
		}
	}

	// longest glob match wins
	longestMatch := 0
	var result RepositoryConfig

	for idx, r := range c.Repositories {
		if !r.IsWildcard() || len(r.Name) < longestMatch {
			continue
		}

		if match, _ := filepath.Match(r.Name, repo); match {
			result = c.Repositories[idx]
			longestMatch = len(r.Name)
		}
	}

	if longestMatch > 0 {
		return &result
	}

	return nil
}
