package gitlab

import (
	"context"

	"mergenator/internal/models"
)

type GitlabService interface {
	CreateMR(ctx context.Context, sourceBranch, repoType string, userID int) (mrURL string, err error)
	HandlePush(ctx context.Context, branch, projectID string) error
	GetStats(ctx context.Context) (models.Stats, error)
}
