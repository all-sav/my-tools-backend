package docker

import (
	"context"

	"mergenator/internal/models"
)

type DockerService interface {
	GetStats(ctx context.Context) (models.DockerStats, error)
}
