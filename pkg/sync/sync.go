package sync

import (
	"context"
	"fmt"
	"log"

	"github.com/kubermatic-labs/aquayman/pkg/config"
	"github.com/kubermatic-labs/aquayman/pkg/quay"
	"github.com/kubermatic-labs/aquayman/pkg/util"
)

type Options struct {
	CreateMissingRepositories  bool
	DeleteDanglingRepositories bool
}

func DefaultOptions() Options {
	return Options{}
}

func Sync(ctx context.Context, config *config.Config, client *quay.Client, options Options) error {
	if err := syncRobots(ctx, config, client); err != nil {
		return fmt.Errorf("failed to sync robots: %v", err)
	}

	if err := syncTeams(ctx, config, client); err != nil {
		return fmt.Errorf("failed to sync teams: %v", err)
	}

	if err := syncRepositories(ctx, config, client, options); err != nil {
		return fmt.Errorf("failed to sync repositories: %v", err)
	}

	return nil
}

func syncRobots(ctx context.Context, config *config.Config, client *quay.Client) error {
	log.Println("⇄ Syncing robots…")

	allRobots, err := client.GetOrganizationRobots(ctx, config.Organization, quay.GetOrganizationRobotsOptions{})
	if err != nil {
		return fmt.Errorf("failed to list existing organization robots: %v", err)
	}

	existingNames := []string{}
	for _, robot := range allRobots {
		existingNames = append(existingNames, robot.ShortName())
	}

	expectedRobots := []string{}

	for _, robot := range config.Robots {
		expectedRobots = append(expectedRobots, robot.Name)

		// do nothing to existing robots, the quay.io API does not offer an endpoint
		// to update a robot's description
		if util.StringSliceContains(existingNames, robot.Name) {
			continue
		}

		log.Printf("  + ⚛ %s", robot.Name)

		options := quay.CreateOrganizationRobotOptions{
			Description: robot.Description,
		}

		if err := client.CreateOrganizationRobot(ctx, config.Organization, robot.Name, options); err != nil {
			return fmt.Errorf("failed to create robot: %v", err)
		}
	}

	for _, robot := range allRobots {
		shortName := robot.ShortName()

		if !util.StringSliceContains(expectedRobots, shortName) {
			log.Printf("  - ⚛ %s", shortName)

			if err := client.DeleteOrganizationRobot(ctx, config.Organization, shortName); err != nil {
				return fmt.Errorf("failed to delete robot: %v", err)
			}
		}
	}

	return nil
}

func syncTeams(ctx context.Context, config *config.Config, client *quay.Client) error {
	log.Println("⇄ Syncing teams…")

	expectedTeams := []string{}

	for _, team := range config.Teams {
		log.Printf("  ✎ ⚑ %s", team.Name)

		options := quay.UpsertTeamOptions{
			Role:        team.Role,
			Description: team.Description,
		}

		if err := client.UpsertTeam(ctx, config.Organization, team.Name, options); err != nil {
			return fmt.Errorf("failed to ensure team: %v", err)
		}

		if err := syncTeamMembers(ctx, config, client, team); err != nil {
			return fmt.Errorf("failed to ensure team members: %v", err)
		}

		expectedTeams = append(expectedTeams, team.Name)
	}

	org, err := client.GetOrganization(ctx, config.Organization)
	if err != nil {
		return err
	}

	for _, teamName := range org.OrderedTeams {
		if !util.StringSliceContains(expectedTeams, teamName) {
			log.Printf("  - ⚑ %s", teamName)

			if err := client.DeleteTeam(ctx, config.Organization, teamName); err != nil {
				return fmt.Errorf("failed to delete team: %v", err)
			}
		}
	}

	return nil
}

func syncTeamMembers(ctx context.Context, config *config.Config, client *quay.Client, team config.TeamConfig) error {
	var (
		currentMembers []quay.TeamMember
		err            error
	)

	if !client.Dry {
		yesPlease := true
		getTeamOptions := quay.GetTeamMembersOptions{
			IncludePending: &yesPlease,
		}

		currentMembers, err = client.GetTeamMembers(ctx, config.Organization, team.Name, getTeamOptions)
		if err != nil {
			return fmt.Errorf("failed to list team members: %v", err)
		}
	}

	currentMemberNames := []string{}

	for _, member := range currentMembers {
		currentMemberNames = append(currentMemberNames, member.Name)

		if !util.StringSliceContains(team.Members, member.Name) {
			log.Printf("    - ♟ %s", member.Name)

			if err := client.RemoveUserFromTeam(ctx, config.Organization, team.Name, member.Name); err != nil {
				return fmt.Errorf("failed to remove member: %v", err)
			}
		}
	}

	for _, member := range team.Members {
		if !util.StringSliceContains(currentMemberNames, member) {
			log.Printf("    + ♟ %s", member)

			if err := client.AddUserToTeam(ctx, config.Organization, team.Name, member); err != nil {
				return fmt.Errorf("failed to add member: %v", err)
			}
		}
	}

	return nil
}

func syncRepositories(ctx context.Context, config *config.Config, client *quay.Client, options Options) error {
	log.Println("⇄ Syncing repositories…")

	requestOptions := quay.GetRepositoriesOptions{
		Namespace: config.Organization,
	}

	currentRepos, err := client.GetRepositories(ctx, requestOptions)
	if err != nil {
		return fmt.Errorf("failed to retrieve repositories: %v", err)
	}

	// update/delete existing repos
	currentRepoNames := []string{}
	for _, repo := range currentRepos {
		repoConfig := config.GetRepositoryConfig(repo.Name)
		if repoConfig == nil {
			if options.DeleteDanglingRepositories {
				log.Printf("  - ⚒ %s", repo.Name)
				if err := client.DeleteRepository(ctx, repo.FullName()); err != nil {
					return err
				}
			}

			continue
		}

		log.Printf("  ✎ ⚒ %s", repo.Name)
		if err := syncRepository(ctx, client, repo, repoConfig); err != nil {
			return err
		}

		currentRepoNames = append(currentRepoNames, repo.Name)
	}

	// create missing repos on quay.io
	if options.CreateMissingRepositories {
		for _, repoConfig := range config.Repositories {
			// ignore wildcard rules
			if repoConfig.IsWildcard() {
				continue
			}

			if !util.StringSliceContains(currentRepoNames, repoConfig.Name) {
				log.Printf("  + ⚒ %s", repoConfig.Name)

				options := quay.CreateRepositoryOptions{
					Namespace:   config.Organization,
					Repository:  repoConfig.Name,
					Description: repoConfig.Description,
					Visibility:  repoConfig.Visibility,
				}

				if err := client.CreateRepository(ctx, options); err != nil {
					return err
				}

				// doing it like this instead of GETing the repo after creation makes it
				// safe for running in dry mode
				repo := quay.Repository{
					Namespace:   config.Organization,
					Name:        repoConfig.Name,
					IsPublic:    repoConfig.Visibility == quay.Public,
					Description: repoConfig.Description,
				}

				if err := syncRepository(ctx, client, repo, &repoConfig); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func syncRepository(ctx context.Context, client *quay.Client, repo quay.Repository, repoConfig *config.RepositoryConfig) error {
	if repo.Visibility() != repoConfig.Visibility {
		log.Printf("    - set visibility to %s", repoConfig.Visibility)
		if err := client.ChangeRepositoryVisibility(ctx, repo.FullName(), repoConfig.Visibility); err != nil {
			return fmt.Errorf("failed to set visibility: %v", err)
		}
	}

	if repo.Description != repoConfig.Description {
		options := quay.UpdateRepositoryOptions{
			Description: repoConfig.Description,
		}

		if err := client.UpdateRepository(ctx, repo.FullName(), options); err != nil {
			return fmt.Errorf("failed to update description: %v", err)
		}
	}

	if err := syncRepositoryTeams(ctx, client, repo.FullName(), repoConfig); err != nil {
		return fmt.Errorf("failed to teams: %v", err)
	}

	if err := syncRepositoryUsers(ctx, client, repo.FullName(), repoConfig); err != nil {
		return fmt.Errorf("failed to users: %v", err)
	}

	return nil
}

func syncRepositoryTeams(ctx context.Context, client *quay.Client, fullRepoName string, repo *config.RepositoryConfig) error {
	// amazingly, this API call does not fail if the repo does not exist, so we can
	// perform it even in dry mode
	currentTeams, err := client.GetRepositoryTeamPermissions(ctx, fullRepoName)
	if err != nil {
		return fmt.Errorf("failed to get team permissions: %v", err)
	}

	currentTeamNames := []string{}

	for _, team := range currentTeams {
		currentTeamNames = append(currentTeamNames, team.Name)

		expectedRole, exists := repo.Teams[team.Name]
		if !exists {
			log.Printf("    - ⚑ %s", team.Name)

			if err := client.RemoveTeamFromRepository(ctx, fullRepoName, team.Name); err != nil {
				return fmt.Errorf("failed to remove team: %v", err)
			}
		} else if expectedRole != team.Role {
			log.Printf("    + ⚑ %s", team.Name)

			if err := client.SetTeamRepositoryPermissions(ctx, fullRepoName, team.Name, expectedRole); err != nil {
				return fmt.Errorf("failed to set team permissions: %v", err)
			}
		}
	}

	for teamName, role := range repo.Teams {
		if !util.StringSliceContains(currentTeamNames, teamName) {
			log.Printf("    + ⚑ %s", teamName)

			if err := client.SetTeamRepositoryPermissions(ctx, fullRepoName, teamName, role); err != nil {
				return fmt.Errorf("failed to set team permissions: %v", err)
			}
		}
	}

	return nil
}

func syncRepositoryUsers(ctx context.Context, client *quay.Client, fullRepoName string, repo *config.RepositoryConfig) error {
	// amazingly, this API call does not fail if the repo does not exist, so we can
	// perform it even in dry mode
	currentUsers, err := client.GetRepositoryUserPermissions(ctx, fullRepoName)
	if err != nil {
		return fmt.Errorf("failed to get user permissions: %v", err)
	}

	currentUserNames := []string{}

	for _, user := range currentUsers {
		currentUserNames = append(currentUserNames, user.Name)

		expectedRole, exists := repo.Users[user.Name]
		if !exists {
			log.Printf("    - ♟ %s", user.Name)

			if err := client.RemoveUserFromRepository(ctx, fullRepoName, user.Name); err != nil {
				return fmt.Errorf("failed to remove user: %v", err)
			}
		} else if expectedRole != user.Role {
			log.Printf("    + ♟ %s", user.Name)

			if err := client.SetUserRepositoryPermissions(ctx, fullRepoName, user.Name, expectedRole); err != nil {
				return fmt.Errorf("failed to set user permissions: %v", err)
			}
		}
	}

	for userName, role := range repo.Users {
		if !util.StringSliceContains(currentUserNames, userName) {
			log.Printf("    + ♟ %s", userName)

			if err := client.SetUserRepositoryPermissions(ctx, fullRepoName, userName, role); err != nil {
				return fmt.Errorf("failed to set user permissions: %v", err)
			}
		}
	}

	return nil
}
