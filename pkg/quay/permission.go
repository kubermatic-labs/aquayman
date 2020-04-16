package quay

import (
	"context"
	"fmt"
	"net/url"
)

type Permission struct {
	Role RepositoryRole `json:"role"`
	Name string         `json:"name"`

	// these are only set for user permissions
	IsOrgMember bool `json:"is_org_member"`
	IsRobot     bool `json:"is_robot"`
}

func (c *Client) GetRepositoryUserPermissions(ctx context.Context, repo string) (map[string]Permission, error) {
	return c.getRepositoryPermissions(ctx, repo, "user")
}

func (c *Client) GetRepositoryTeamPermissions(ctx context.Context, repo string) (map[string]Permission, error) {
	return c.getRepositoryPermissions(ctx, repo, "team")
}

type getRepositoryPermissionsReponse struct {
	Permissions map[string]Permission `json:"permissions"`
}

func (c *Client) getRepositoryPermissions(ctx context.Context, repo string, kind string) (map[string]Permission, error) {
	response := getRepositoryPermissionsReponse{}
	path := fmt.Sprintf("/repository/%s/permissions/%s/", repo, kind) // the trailing slash is important
	err := c.call(ctx, "GET", path, nil, nil, &response)

	return response.Permissions, err
}

func (c *Client) SetUserRepositoryPermissions(ctx context.Context, repo string, user string, role RepositoryRole) error {
	return c.setRepositoryPermissions(ctx, repo, "user", user, role)
}

func (c *Client) SetTeamRepositoryPermissions(ctx context.Context, repo string, team string, role RepositoryRole) error {
	return c.setRepositoryPermissions(ctx, repo, "team", team, role)
}

type permissionBody struct {
	Role RepositoryRole `json:"role"`
}

func (c *Client) setRepositoryPermissions(ctx context.Context, repo string, kind string, user string, role RepositoryRole) error {
	path := fmt.Sprintf("/repository/%s/permissions/%s/%s", repo, kind, url.PathEscape(user))
	body := permissionBody{
		Role: role,
	}

	return c.call(ctx, "PUT", path, nil, toBody(body), nil)
}

func (c *Client) RemoveUserFromRepository(ctx context.Context, repo string, user string) error {
	return c.removeFromRepository(ctx, repo, "user", user)
}

func (c *Client) RemoveTeamFromRepository(ctx context.Context, repo string, team string) error {
	return c.removeFromRepository(ctx, repo, "team", team)
}

func (c *Client) removeFromRepository(ctx context.Context, repo string, kind string, user string) error {
	path := fmt.Sprintf("/repository/%s/permissions/%s/%s", repo, kind, url.PathEscape(user))

	return c.call(ctx, "DELETE", path, nil, nil, nil)
}
