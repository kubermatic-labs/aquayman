package export

import (
	"context"
	"fmt"
	"log"
	"sort"

	"github.com/kubermatic-labs/aquayman/pkg/config"
	"github.com/kubermatic-labs/aquayman/pkg/quay"
)

func ExportConfiguration(ctx context.Context, organization string, client *quay.Client) (*config.Config, error) {
	cfg := &config.Config{
		Organization: organization,
	}

	if err := exportRobots(ctx, client, cfg); err != nil {
		return cfg, fmt.Errorf("failed to export robots: %v", err)
	}

	if err := exportRepositories(ctx, client, cfg); err != nil {
		return cfg, fmt.Errorf("failed to export repositories: %v", err)
	}

	if err := exportTeams(ctx, client, cfg); err != nil {
		return cfg, fmt.Errorf("failed to export teams: %v", err)
	}

	return cfg, nil
}

func exportRobots(ctx context.Context, client *quay.Client, cfg *config.Config) error {
	log.Println("⇄ Exporting robots…")

	robots, err := client.GetOrganizationRobots(ctx, cfg.Organization, quay.GetOrganizationRobotsOptions{})
	if err != nil {
		return err
	}

	for _, robot := range robots {
		log.Printf("  ⚛ %s", robot.ShortName())

		cfg.Robots = append(cfg.Robots, config.RobotConfig{
			Name:        robot.ShortName(),
			Description: robot.Description,
		})
	}

	return nil
}

func exportRepositories(ctx context.Context, client *quay.Client, cfg *config.Config) error {
	log.Println("⇄ Exporting repositories…")

	repos, err := client.GetRepositories(ctx, quay.GetRepositoriesOptions{Namespace: cfg.Organization})
	if err != nil {
		return err
	}

	for _, repo := range repos {
		visibilitySuffix := ""
		if !repo.IsPublic {
			visibilitySuffix = " (private)"
		}

		log.Printf("  ⚒ %s%s", repo.Name, visibilitySuffix)

		teamPermissions, err := client.GetRepositoryTeamPermissions(ctx, repo.FullName())
		if err != nil {
			return fmt.Errorf("failed to fetch team permissions: %v", err)
		}

		teams := map[string]quay.RepositoryRole{}

		for _, team := range teamPermissions {
			teams[team.Name] = team.Role
		}

		userPermissions, err := client.GetRepositoryUserPermissions(ctx, repo.FullName())
		if err != nil {
			return fmt.Errorf("failed to fetch user permissions: %v", err)
		}

		users := map[string]quay.RepositoryRole{}

		for _, user := range userPermissions {
			users[user.Name] = user.Role
		}

		visibility := quay.Private
		if repo.IsPublic {
			visibility = quay.Public
		}

		cfg.Repositories = append(cfg.Repositories, config.RepositoryConfig{
			Name:        repo.Name,
			Description: repo.Description,
			Visibility:  visibility,
			Teams:       teams,
			Users:       users,
		})
	}

	return nil
}

func exportTeams(ctx context.Context, client *quay.Client, cfg *config.Config) error {
	log.Println("⇄ Exporting teams…")

	org, err := client.GetOrganization(ctx, cfg.Organization)
	if err != nil {
		return err
	}

	for _, teamName := range org.OrderedTeams {
		team := org.Teams[teamName]

		log.Printf("  ⚑ %s", team.Name)

		yes := true
		options := quay.GetTeamMembersOptions{
			IncludePending: &yes,
		}

		members, err := client.GetTeamMembers(ctx, cfg.Organization, team.Name, options)
		if err != nil {
			return fmt.Errorf("failed to fetch team: %v", err)
		}

		memberNames := []string{}
		for _, member := range members {
			memberNames = append(memberNames, member.Name)
		}

		sort.Strings(memberNames)

		cfg.Teams = append(cfg.Teams, config.TeamConfig{
			Name:    team.Name,
			Role:    team.Role,
			Members: memberNames,
		})
	}

	return nil
}
