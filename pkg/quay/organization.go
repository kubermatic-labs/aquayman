package quay

import (
	"context"
	"fmt"
	"net/url"
)

type Organization struct {
	Name                string          `json:"name"`
	Email               string          `json:"email"`
	InvoiceEmail        bool            `json:"invoice_email"`
	InvoiceEmailAddress string          `json:"invoice_email_address"`
	IsMember            bool            `json:"is_member"`
	IsFreeAccount       bool            `json:"is_free_account"`
	IsAdmin             bool            `json:"is_admin"`
	Teams               map[string]Team `json:"teams"`
	OrderedTeams        []string        `json:"ordered_teams"`
}

type Team struct {
	Name        string   `json:"name"`
	Role        TeamRole `json:"role"`
	Description string   `json:"description"`
	IsSynced    bool     `json:"is_synced"`
	CanView     bool     `json:"can_view"`
	MemberCount int      `json:"member_count"`
	RepoCount   int      `json:"repo_count"`
}

func (c *Client) GetOrganization(ctx context.Context, name string) (*Organization, error) {
	org := &Organization{}
	path := fmt.Sprintf("/organization/%s", url.PathEscape(name))
	err := c.call(ctx, "GET", path, nil, nil, &org)

	return org, err
}

type OrganizationMember struct {
	Kind         string   `json:"kind"`
	Name         string   `json:"name"`
	Repositories []string `json:"repositories"`
	Teams        []struct {
		Name string `json:"name"`
	} `json:"teams"`
}

type getOrganizationMembersReponse struct {
	Members []OrganizationMember `json:"members"`
}

func (c *Client) GetOrganizationMembers(ctx context.Context, organization string) ([]OrganizationMember, error) {
	response := getOrganizationMembersReponse{}
	path := fmt.Sprintf("/organization/%s/members", url.PathEscape(organization))
	err := c.call(ctx, "GET", path, nil, nil, &response)

	return response.Members, err
}
