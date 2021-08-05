package config

import (
	"context"
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

var (
	userRegexp  = regexp.MustCompile(`^[a-z0-9][.a-z0-9_-]*$`)
	teamRegexp  = regexp.MustCompile(`^[a-z][a-z0-9]+$`)
	repoRegexp  = regexp.MustCompile(`^[a-z0-9][.a-z0-9_-]*$`)
	robotRegexp = regexp.MustCompile(`^[a-z][a-z0-9_]{1,254}$`)
	orgRegexp   = regexp.MustCompile(`^[a-z0-9][.a-z0-9_-]{1,254}$`)
)

func validateUsername(ctx context.Context, client *quay.Client, name string, cache map[string]struct{}) error {
	if !userRegexp.MatchString(name) {
		return fmt.Errorf("username is invalid, must be %v", userRegexp)
	}

	if _, ok := cache[name]; ok {
		return nil
	}

	_, err := client.GetUser(ctx, name)
	if err != nil {
		return fmt.Errorf("failed to fetch user: %w", err)
	}

	cache[name] = struct{}{}

	return nil
}

func (c *Config) Validate(ctx context.Context, client *quay.Client) error {
	if c.Organization == "" {
		return errors.New("no organization configured")
	}

	if !orgRegexp.MatchString(c.Organization) {
		return fmt.Errorf("organization name %q is invalid, must be %v", c.Organization, orgRegexp)
	}

	// runtime cache
	existingUsers := map[string]struct{}{}

	robotNames := []string{}
	prefix := c.Organization + "+"

	for _, robot := range c.Robots {
		fullName := fmt.Sprintf("%s+%s", c.Organization, robot.Name)

		if util.StringSliceContains(robotNames, fullName) {
			return fmt.Errorf("duplicate robot %q defined", robot.Name)
		}

		if strings.HasPrefix(robot.Name, prefix) {
			return fmt.Errorf("robot %q must be given as a short name, without the organization prefix (must be \"%s\")", robot.Name, strings.TrimPrefix(robot.Name, prefix))
		}

		if !robotRegexp.MatchString(robot.Name) {
			return fmt.Errorf("robot name %q is invalid, must be %v", robot.Name, robotRegexp)
		}

		robotNames = append(robotNames, fullName)
	}

	teamNames := []string{}

	for _, team := range c.Teams {
		if util.StringSliceContains(teamNames, team.Name) {
			return fmt.Errorf("duplicate team %q defined", team.Name)
		}

		if !validTeamRole(team.Role) {
			return fmt.Errorf("role for team %q is invalid (%q), must be one of %v", team.Name, team.Role, quay.AllTeamRoles)
		}

		if !teamRegexp.MatchString(team.Name) {
			return fmt.Errorf("team name %q is invalid, must be %v", team.Name, teamRegexp)
		}

		teamNames = append(teamNames, team.Name)

		if client != nil {
			for _, member := range team.Members {
				if quay.IsRobotUsername(member) {
					if !util.StringSliceContains(robotNames, member) {
						return fmt.Errorf("robot %q in team %q does not exist", member, team.Name)
					}
				} else if err := validateUsername(ctx, client, member, existingUsers); err != nil {
					return fmt.Errorf("user %q in team %q is invalid: %v", member, team.Name, err)
				}
			}
		}
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

		if !repoRegexp.MatchString(repo.Name) {
			return fmt.Errorf("repository name %q is invalid, must be %v", repo.Name, repoRegexp)
		}

		for teamName, roleName := range repo.Teams {
			if !util.StringSliceContains(teamNames, teamName) {
				return fmt.Errorf("invalid team %q assigned to repo %q: team does not exist", teamName, repo.Name)
			}

			if !validRepositoryRole(roleName) {
				return fmt.Errorf("role for team %s in repo %q is invalid (%q), must be one of %v", teamName, repo.Name, roleName, quay.AllRepositoryRoles)
			}
		}

		for userName, roleName := range repo.Users {
			if !validRepositoryRole(roleName) {
				return fmt.Errorf("role for user %s in repo %q is invalid (%q), must be one of %v", userName, repo.Name, roleName, quay.AllRepositoryRoles)
			}

			if quay.IsRobotUsername(userName) {
				if !util.StringSliceContains(robotNames, userName) {
					return fmt.Errorf("invalid robot %q assigned to repo %q: robot does not exist", userName, repo.Name)
				}
			} else if client != nil {
				if _, err := client.GetUser(ctx, userName); err != nil {
					return fmt.Errorf("invalid user %q assigned to repo %q: user does not exist", userName, repo.Name)
				}
			}
		}

		repoNames = append(repoNames, repo.Name)
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
