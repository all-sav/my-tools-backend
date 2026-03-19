package models

// RepoStats отражает статистику отдельно по фронтенду и бэкенду.
type RepoStats struct {
	Frontend int `json:"frontend"`
	Backend  int `json:"backend"`
}

// Stats — данные, которые показывает Welcome.
type Stats struct {
	MrCreated      int       `json:"mrCreated"`
	ActiveBranches int       `json:"activeBranches"`
	MrByRepo       RepoStats `json:"mrByRepo"`
	BranchesByRepo RepoStats `json:"branchesByRepo"`
}
