package publisher

import (
	"context"

	"github.com/kubermatic-labs/aquayman/pkg/config"
)

type Publisher interface {
	UpdateRobot(ctx context.Context, robot *config.RobotConfig, token string) error
	DeleteRobot(ctx context.Context, robot *config.RobotConfig) error
}
