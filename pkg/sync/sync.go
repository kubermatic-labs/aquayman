package sync

import (
	"context"
	"fmt"
	"log"

	"github.com/kubermatic-labs/aquayman/pkg/config"
	"github.com/kubermatic-labs/aquayman/pkg/quay"
	"github.com/kubermatic-labs/aquayman/pkg/util"
)

func Sync(ctx context.Context, config *config.Config, client *quay.Client) error {
	if err := syncRobots(ctx, config, client); err != nil {
		return fmt.Errorf("failed to sync robots: %v", err)
	}

	if err := syncTeams(ctx, config, client); err != nil {
		return fmt.Errorf("failed to sync teams: %v", err)
	}

	if err := syncRepositories(ctx, config, client); err != nil {
		return fmt.Errorf("failed to sync repositories: %v", err)
	}

	return nil
}

func syncRobots(ctx context.Context, config *config.Config, client *quay.Client) error {
	log.Println("⇄ Syncing robots…")

	expectedRobots := []string{}

	for _, robot := range config.Robots {
		log.Printf("  ✎ ⚛ %s", robot.Name)

		// Ensure robot exists and has the correct description.
		options := quay.UpsertOrganizationRobotOptions{
			Description: robot.Description,
		}

		if err := client.UpsertOrganizationRobot(ctx, config.Organization, robot.Name, options); err != nil {
			return fmt.Errorf("failed to ensure robot: %v", err)
		}

		expectedRobots = append(expectedRobots, robot.Name)
	}

	allRobots, err := client.GetOrganizationRobots(ctx, config.Organization, quay.GetOrganizationRobotsOptions{})
	if err != nil {
		return fmt.Errorf("failed to list existing organization robots: %v", err)
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
		yesPlase := true
		getTeamOptions := quay.GetTeamMembersOptions{
			IncludePending: &yesPlase,
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

func syncRepositories(ctx context.Context, config *config.Config, client *quay.Client) error {
	log.Println("⇄ Syncing repositories…")

	options := quay.GetRepositoriesOptions{
		Namespace: config.Organization,
	}

	repositories, err := client.GetRepositories(ctx, options)
	if err != nil {
		return fmt.Errorf("failed to retrieve repositories: %v", err)
	}

	for _, repo := range repositories {
		if err := syncRepository(ctx, config, client, repo); err != nil {
			return err
		}
	}

	return nil
}

func syncRepository(ctx context.Context, config *config.Config, client *quay.Client, repo quay.Repository) error {
	// ignore repos for which we have no matching rule set
	repoConfig := config.GetRepositoryConfig(repo.Name)
	if repoConfig == nil {
		return nil
	}

	visibility := ""
	if !repo.IsPublic {
		visibility = " (private)"
	}

	log.Printf("  ⚒ %s%s", repo.Name, visibility)

	if err := syncRepositoryTeams(ctx, config, client, repo.FullName(), repoConfig); err != nil {
		return err
	}

	if err := syncRepositoryUsers(ctx, config, client, repo.FullName(), repoConfig); err != nil {
		return err
	}

	return nil
}

func syncRepositoryTeams(ctx context.Context, config *config.Config, client *quay.Client, fullRepoName string, repo *config.RepositoryConfig) error {
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

			if err := client.SetTeamRepositoryPermissions(ctx, fullRepoName, team.Name, team.Role); err != nil {
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

func syncRepositoryUsers(ctx context.Context, config *config.Config, client *quay.Client, fullRepoName string, repo *config.RepositoryConfig) error {
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

			if err := client.SetUserRepositoryPermissions(ctx, fullRepoName, user.Name, user.Role); err != nil {
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
