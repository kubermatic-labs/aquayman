package quay

import (
	"context"
	"fmt"
	"net/url"
	"sort"
	"strings"
)

type Robot struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Token       string `json:"token"`

	// Repositories is only set if the permissions option is set
	// when fetching robots.
	Repositories []string `json:"repositories"`

	// Teams is only set if the permissions option is set
	// when fetching robots.
	Teams []struct {
		Name string `json:"name"`
	} `json:"teams"`
}

func (r *Robot) ShortName() string {
	parts := strings.SplitN(r.Name, "+", 2)

	return parts[len(parts)-1]
}

type getOrganizationRobotsReponse struct {
	Robots []Robot `json:"robots"`
}

type GetOrganizationRobotsOptions struct {
	Token       *bool
	Permissions *bool
}

func (o *GetOrganizationRobotsOptions) Apply(v url.Values) url.Values {
	if o.Token != nil {
		v.Set("token", fmt.Sprintf("%v", *o.Token))
	}

	if o.Permissions != nil {
		v.Set("permissions", fmt.Sprintf("%v", *o.Permissions))
	}

	return v
}

type RobotByName []Robot

func (a RobotByName) Len() int           { return len(a) }
func (a RobotByName) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a RobotByName) Less(i, j int) bool { return a[i].Name < a[j].Name }

func (c *Client) GetOrganizationRobots(ctx context.Context, org string, options GetOrganizationRobotsOptions) ([]Robot, error) {
	response := getOrganizationRobotsReponse{}
	path := fmt.Sprintf("/organization/%s/robots", url.PathEscape(org))
	err := c.call(ctx, "GET", path, &options, nil, &response)

	robots := response.Robots
	sort.Sort(RobotByName(robots))

	return robots, err
}

type CreateOrganizationRobotOptions struct {
	Description string `json:"description"`
}

func (c *Client) CreateOrganizationRobot(ctx context.Context, org string, shortName string, opt CreateOrganizationRobotOptions) error {
	path := fmt.Sprintf("/organization/%s/robots/%s", url.PathEscape(org), url.PathEscape(shortName))

	return c.call(ctx, "PUT", path, nil, toBody(opt), nil)
}

func (c *Client) DeleteOrganizationRobot(ctx context.Context, org string, shortName string) error {
	path := fmt.Sprintf("/organization/%s/robots/%s", url.PathEscape(org), url.PathEscape(shortName))

	return c.call(ctx, "DELETE", path, nil, nil, nil)
}
