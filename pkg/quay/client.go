package quay

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"time"
)

type RepositoryRole string

const (
	// Can view and pull from the repository
	ReadRepositoryRole RepositoryRole = "read"
	// Can view, pull and push to the repository
	WriteRepositoryRole RepositoryRole = "write"
	// Full admin access, pull and push on the repository
	AdminRepositoryRole RepositoryRole = "admin"
)

var AllRepositoryRoles = []RepositoryRole{ReadRepositoryRole, WriteRepositoryRole, AdminRepositoryRole}

type TeamRole string

const (
	// Inherits all permissions of the team
	MemberTeamRole TeamRole = "member"
	// Member and can create new repositories
	CreatorTeamRole TeamRole = "creator"
	// Full admin access to the organization
	AdminTeamRole TeamRole = "admin"
)

var AllTeamRoles = []TeamRole{MemberTeamRole, CreatorTeamRole, AdminTeamRole}

type Client struct {
	Token  string
	Client *http.Client
	Dry    bool
}

func NewClient(token string, timeout time.Duration, dryMode bool) (*Client, error) {
	if token == "" {
		return nil, fmt.Errorf("no OAuth2 token provided")
	}

	httpClient := &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			Dial: (&net.Dialer{
				Timeout: timeout,
			}).Dial,
			TLSHandshakeTimeout: timeout,
		},
	}

	return &Client{
		Token:  token,
		Client: httpClient,
		Dry:    dryMode,
	}, nil
}

type options interface {
	Apply(url.Values) url.Values
}

func toBody(options interface{}) io.Reader {
	var buf bytes.Buffer

	_ = json.NewEncoder(&buf).Encode(options)

	return &buf
}

type APIError struct {
	Status       int    `json:"status"`
	ErrorMessage string `json:"error_message"`
	Title        string `json:"title"`
	ErrorType    string `json:"error_type"`
	Detail       string `json:"detail"`
	Type         string `json:"type"`

	// Message is only set in some error conditions, like when adding
	// a user to a team they are already a member of. This is not documented,
	// but reality nontheless.
	Message string `json:"message"`
}

func (e *APIError) Error() string {
	if e.Message != "" {
		return e.Message
	}

	return fmt.Sprintf("%s: %s", e.Title, e.ErrorMessage)
}

func (c *Client) call(ctx context.Context, method string, path string, opt options, body io.Reader, model interface{}) error {
	if opt != nil {
		query := opt.Apply(url.Values{})
		if len(query) > 0 {
			path = fmt.Sprintf("%s?%s", path, query.Encode())
		}
	}

	if method != http.MethodGet && c.Dry {
		return nil
	}

	u := "https://quay.io/api/v1" + path

	request, err := http.NewRequestWithContext(ctx, method, u, body)
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	request.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.Token))
	if body != nil {
		request.Header.Set("Content-Type", "application/json; charset=utf-8")
	}

	response, err := c.Client.Do(request)
	if err != nil {
		return fmt.Errorf("request failed: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode >= 400 {
		e := &APIError{}
		if err := json.NewDecoder(response.Body).Decode(&e); err != nil {
			return fmt.Errorf("request failed and decoding the response also failed, HTTP status was %s: %v", response.Status, err)
		}

		return e
	}

	if response.StatusCode >= 300 {
		return fmt.Errorf("request failed with status code %s", response.Status)
	}

	if model != nil {
		if err := json.NewDecoder(response.Body).Decode(model); err != nil {
			return fmt.Errorf("failed to decode response: %v", err)
		}
	}

	return nil
}
