package gitlab

import (
	"context"
)

type GitlabService interface {
	CreateMR(ctx context.Context, sourceBranch, repoType string, userID int) (mrURL string, err error)
	HandlePush(ctx context.Context, branch, projectID string) error
}
