package models

// RepoStats отражает статистику отдельно по фронтенду и бэкенду.
type RepoStats struct {
	Frontend int `json:"frontend"`
	Backend  int `json:"backend"`
}

type DockerContainerStats struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Service       string `json:"service"`
	Stack         string `json:"stack"`
	ComposeFile   string `json:"composeFile"`
	State         string `json:"state"`
	Status        string `json:"status,omitempty"`
	StartedAt     string `json:"startedAt,omitempty"`
	UptimeSeconds int64  `json:"uptimeSeconds"`
	UptimeHuman   string `json:"uptimeHuman"`
}

type DockerStats struct {
	RunningContainers int                    `json:"runningContainers"`
	Containers        []DockerContainerStats `json:"containers"`
	Errors            []string               `json:"errors,omitempty"`
}

// Stats — данные, которые показывает Welcome.
type Stats struct {
	MrCreated      int         `json:"mrCreated"`
	ActiveBranches int         `json:"activeBranches"`
	MrByRepo       RepoStats   `json:"mrByRepo"`
	BranchesByRepo RepoStats   `json:"branchesByRepo"`
	Docker         DockerStats `json:"docker"`
}
