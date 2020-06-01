package quay

import (
	"context"
	"fmt"
	"net/url"
	"sort"
)

type RepositoryKind string

const (
	ImageRepository       RepositoryKind = "image"
	ApplicationRepository RepositoryKind = "application"
)

type RepositoryVisibility string

const (
	Public  RepositoryVisibility = "public"
	Private RepositoryVisibility = "private"
)

type Repository struct {
	Kind        string      `json:"kind"`
	Name        string      `json:"name"`
	Namespace   string      `json:"namespace"`
	State       interface{} `json:"state"`
	IsPublic    bool        `json:"is_public"`
	IsStarred   bool        `json:"is_starred"`
	Description string      `json:"description"`
}

func (r *Repository) FullName() string {
	return r.Namespace + "/" + r.Name
}

type getRepositoriesReponse struct {
	Repositories []Repository `json:"repositories"`
}

type GetRepositoriesOptions struct {
	Namespace string
	Starred   *bool
	Public    *bool
}

func (o *GetRepositoriesOptions) Apply(v url.Values) url.Values {
	if o.Namespace != "" {
		v.Set("namespace", o.Namespace)
	}

	if o.Starred != nil {
		v.Set("starred", fmt.Sprintf("%v", *o.Starred))
	}

	if o.Public != nil {
		v.Set("public", fmt.Sprintf("%v", *o.Public))
	}

	return v
}

type RepoByName []Repository

func (a RepoByName) Len() int           { return len(a) }
func (a RepoByName) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a RepoByName) Less(i, j int) bool { return a[i].Name < a[j].Name }

func (c *Client) GetRepositories(ctx context.Context, options GetRepositoriesOptions) ([]Repository, error) {
	response := getRepositoriesReponse{}
	err := c.call(ctx, "GET", "/repository", &options, nil, &response)

	repositories := response.Repositories
	sort.Sort(RepoByName(repositories))

	return repositories, err
}

type CreateRepositoryOptions struct {
	Kind        RepositoryKind       `json:"kind"`
	Namespace   string               `json:"namespace"`
	Repository  string               `json:"repository"`
	Visibility  RepositoryVisibility `json:"visibility"`
	Description string               `json:"description"`
}

func (c *Client) CreateRepository(ctx context.Context, opt CreateRepositoryOptions) error {
	return c.call(ctx, "POST", "/repository", nil, toBody(opt), nil)
}

type UpdateRepositoryOptions struct {
	Description string `json:"description"`
}

func (c *Client) UpdateRepository(ctx context.Context, repo string, opt UpdateRepositoryOptions) error {
	return c.call(ctx, "PUT", fmt.Sprintf("/repository/%s", repo), nil, toBody(opt), nil)
}

type changeRepositoryVisibilityBody struct {
	Visibility RepositoryVisibility `json:"visibility"`
}

func (c *Client) ChangeRepositoryVisibility(ctx context.Context, repo string, visibility RepositoryVisibility) error {
	url := fmt.Sprintf("/repository/%s/changevisibility", repo)
	body := toBody(changeRepositoryVisibilityBody{
		Visibility: visibility,
	})

	return c.call(ctx, "POST", url, nil, body, nil)
}

func (c *Client) DeleteRepository(ctx context.Context, repo string) error {
	return c.call(ctx, "DELETE", fmt.Sprintf("/repository/%s", repo), nil, nil, nil)
}
