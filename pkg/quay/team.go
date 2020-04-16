package quay

import (
	"context"
	"fmt"
	"net/url"
)

type TeamMembershipKind string

const (
	TeamMemberUser  TeamMembershipKind = "user"
	TeamMemberRobot TeamMembershipKind = "robot" // maybe?
)

type TeamMember struct {
	Kind    TeamMembershipKind `json:"kind"`
	Name    string             `json:"name"`
	Invited bool               `json:"invited"`
	IsRobot bool               `json:"is_robot"`
}

type getTeamMembersReponse struct {
	Name    string       `json:"name"`
	CanEdit bool         `json:"can_edit"`
	Members []TeamMember `json:"members"`
}

type GetTeamMembersOptions struct {
	IncludePending *bool
}

func (o *GetTeamMembersOptions) Apply(v url.Values) url.Values {
	if o.IncludePending != nil {
		v.Set("includePending", fmt.Sprintf("%v", *o.IncludePending))
	}

	return v
}

func (c *Client) GetTeamMembers(ctx context.Context, org string, team string, opt GetTeamMembersOptions) ([]TeamMember, error) {
	response := getTeamMembersReponse{}
	path := fmt.Sprintf("/organization/%s/team/%s/members", url.PathEscape(org), url.PathEscape(team))
	err := c.call(ctx, "GET", path, &opt, nil, &response)

	return response.Members, err
}

func (c *Client) AddUserToTeam(ctx context.Context, org string, team string, member string) error {
	path := fmt.Sprintf("/organization/%s/team/%s/members/%s", url.PathEscape(org), url.PathEscape(team), url.PathEscape(member))

	return c.call(ctx, "PUT", path, nil, nil, nil)
}

func (c *Client) RemoveUserFromTeam(ctx context.Context, org string, team string, member string) error {
	path := fmt.Sprintf("/organization/%s/team/%s/members/%s", url.PathEscape(org), url.PathEscape(team), url.PathEscape(member))

	return c.call(ctx, "DELETE", path, nil, nil, nil)
}

type UpsertTeamOptions struct {
	Role        TeamRole `json:"role"`
	Description string   `json:"description"`
}

func (c *Client) UpsertTeam(ctx context.Context, org string, team string, opt UpsertTeamOptions) error {
	path := fmt.Sprintf("/organization/%s/team/%s", url.PathEscape(org), url.PathEscape(team))

	return c.call(ctx, "PUT", path, nil, toBody(opt), nil)
}

func (c *Client) DeleteTeam(ctx context.Context, org string, team string) error {
	path := fmt.Sprintf("/organization/%s/team/%s", url.PathEscape(org), url.PathEscape(team))

	return c.call(ctx, "DELETE", path, nil, nil, nil)
}
