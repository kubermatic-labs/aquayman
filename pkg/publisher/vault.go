package publisher

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hashicorp/vault/api"
	"github.com/kubermatic-labs/aquayman/pkg/config"
)

type Vault struct {
	client *api.Client
	org    string
}

// NewVaultPublisher relies on VAULT_ADDR and VAULT_TOKEN env
// variables be set.
func NewVaultPublisher(organization string) (*Vault, error) {
	client, err := api.NewClient(nil)
	if err != nil {
		return nil, fmt.Errorf("could not build Vault client: %w", err)
	}

	return &Vault{
		client: client,
		org:    organization,
	}, nil
}

func (v *Vault) UpdateRobot(ctx context.Context, robot *config.RobotConfig, token string) error {
	if robot.VaultSecret == "" {
		return nil
	}

	addr, err := v.getAddress(robot)
	if err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// fetch current state, so we do not need to bump if there are no changes to the token
	secret, err := v.client.Logical().Read(addr.path)
	if err != nil {
		return fmt.Errorf("failed to read from Vault: %w", err)
	}

	// precalculate a nice to use docker pull config JSON file
	configJson, err := v.getDockerConfig(robot, token)
	if err != nil {
		return fmt.Errorf("failed to create Docker config: %w", err)
	}

	secretUpToDate := false
	existingData := map[string]interface{}{}

	if secret != nil {
		// the secrets are wrapped in a "data" field,
		// that's just how kv stores in Vault work

		if data, exists := secret.Data["data"]; exists {
			if m, ok := data.(map[string]interface{}); ok {
				existingData = m

				if value, exists := m[addr.key+"-token"]; exists {
					if svalue, ok := value.(string); ok {
						secretUpToDate = svalue == token
					}
				}

				if secretUpToDate {
					if value, exists := m[addr.key+"-config"]; exists {
						if svalue, ok := value.(string); ok {
							secretUpToDate = svalue == configJson
						}
					}
				}
			}
		}
	}

	if !secretUpToDate {
		existingData[addr.key+"-token"] = token
		existingData[addr.key+"-config"] = configJson

		secret.Data["data"] = existingData

		if _, err := v.client.Logical().Write(addr.path, secret.Data); err != nil {
			return fmt.Errorf("failed to update Vault: %w", err)
		}
	}

	return nil
}

type dockerConfig struct {
	Auths map[string]dockerAuth `json:"auths"`
}

type dockerAuth struct {
	Auth  string `json:"auth"`
	Email string `json:"email"`
}

func (v *Vault) getDockerConfig(robot *config.RobotConfig, token string) (string, error) {
	auth := fmt.Sprintf("%s+%s:%s", v.org, robot.Name, token)
	encoded := base64.StdEncoding.EncodeToString([]byte(auth))

	cfg := dockerConfig{
		Auths: map[string]dockerAuth{
			"quay.io": {
				Auth: encoded,
			},
		},
	}

	var buf bytes.Buffer

	encoder := json.NewEncoder(&buf)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(cfg); err != nil {
		return "", fmt.Errorf("failed to encode docker config as JSON: %w", err)
	}

	return buf.String(), nil
}

func (v *Vault) DeleteRobot(ctx context.Context, robot *config.RobotConfig) error {
	if robot.VaultSecret == "" {
		return nil
	}

	addr, err := v.getAddress(robot)
	if err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// fetch current state, so we do not need to bump if there are no changes to the token
	secret, err := v.client.Logical().Read(addr.path)
	if err != nil {
		return fmt.Errorf("failed to read from Vault: %w", err)
	}

	if secret == nil {
		return nil
	}

	// the secrets are wrapped in a "data" field,
	// that's just how kv stores in Vault work

	if data, exists := secret.Data["data"]; exists {
		if m, ok := data.(map[string]interface{}); ok {
			if _, exists := m[addr.key]; exists {
				delete(m, addr.key+"-token")
				delete(m, addr.key+"-config")

				secret.Data["data"] = m

				if _, err := v.client.Logical().Write(addr.path, secret.Data); err != nil {
					return fmt.Errorf("failed to update Vault: %w", err)
				}
			}
		}
	}

	return nil
}

type address struct {
	path string
	key  string
}

func (v *Vault) getAddress(robot *config.RobotConfig) (*address, error) {
	parts := strings.Split(robot.VaultSecret, "#")
	if len(parts) > 2 {
		return nil, fmt.Errorf("invalid path %q: must not contain more than one # symbol", robot.VaultSecret)
	}

	a := address{
		path: parts[0],
		key:  fmt.Sprintf("quay.io-%s-%s", v.org, robot.Name),
	}

	if len(parts) > 1 {
		a.key = parts[1]
	}

	return &a, nil
}
