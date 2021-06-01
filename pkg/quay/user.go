package quay

import (
	"context"
	"fmt"
	"net/url"
)

type User struct {
	Username string `json:"username"`
}

func (c *Client) GetUser(ctx context.Context, username string) (*User, error) {
	response := User{}
	path := fmt.Sprintf("/users/%s", url.PathEscape(username))
	err := c.call(ctx, "GET", path, nil, nil, &response)

	return &response, err
}
