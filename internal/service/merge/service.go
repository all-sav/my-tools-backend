package merge

import (
	"context"
)

type MergeService interface {
	CreateMR(ctx context.Context, sourceBranch, repoType string, userID int) (mrURL string, err error)
}
