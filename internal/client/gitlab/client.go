package gitlab

import "context"

type Repository struct {
	StandBranch string
	ProjectID   string
	AssigneeID  int
}

type GitLabClient interface {
	BranchExists(ctx context.Context, branch, projectID string) (bool, error)
	HasOpenMR(ctx context.Context, source, target, projectID string) (bool, int, string, error)
	CountOpenMRs(ctx context.Context, projectID string) (int, error)
	CountActiveBranches(ctx context.Context, projectID string) (int, error)
	CreateMR(ctx context.Context, sourceBranch, title string, repo Repository) (string, error)
	CreateBranch(ctx context.Context, sourceBranch, newBranch string, repo Repository) error
	DeleteBranch(ctx context.Context, branch string, repo Repository) error
	MergeBranchInto(ctx context.Context, sourceBranch string, targetBranch string, projectID string) (int, error)
	AcceptMR(ctx context.Context, mrID int, projectID string) error
	FindUserID(ctx context.Context, username string) (int, error)
}
