package quay

import (
	"context"
	"fmt"
	"net/url"
	"sort"
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
