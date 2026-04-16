package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"mergenator/internal/infr/logger"
	"mergenator/internal/models"
	"mergenator/internal/service/settings"

	"github.com/rs/zerolog"
)

var commandContext = exec.CommandContext

type service struct {
	settingsService settings.SettingsService
	log             *zerolog.Logger
	now             func() time.Time
}

type mergenatorSettings struct {
	DockerComposePaths []string `json:"docker_compose_paths"`
}

type composePSContainer struct {
	ID         string `json:"ID"`
	Name       string `json:"Name"`
	Project    string `json:"Project"`
	Service    string `json:"Service"`
	State      string `json:"State"`
	Status     string `json:"Status"`
	RunningFor string `json:"RunningFor"`
}

type dockerInspectContainer struct {
	ID    string `json:"Id"`
	Name  string `json:"Name"`
	State struct {
		StartedAt string `json:"StartedAt"`
	} `json:"State"`
}

func NewDockerService(settingsService settings.SettingsService) DockerService {
	return &service{
		settingsService: settingsService,
		log:             logger.Get(),
		now:             time.Now,
	}
}

func (s *service) GetStats(ctx context.Context) (models.DockerStats, error) {
	cfg, err := s.getMergenatorSettings()
	if err != nil {
		return models.DockerStats{}, err
	}

	if len(cfg.DockerComposePaths) == 0 {
		return models.DockerStats{
			Containers: []models.DockerContainerStats{},
		}, nil
	}

	seen := make(map[string]models.DockerContainerStats)
	errors := make([]string, 0)

	for _, composePath := range cfg.DockerComposePaths {
		containers, err := s.getComposeContainers(ctx, composePath)
		if err != nil {
			s.log.Warn().Err(err).Str("compose_path", composePath).Msg("failed to collect docker compose stats")
			errors = append(errors, err.Error())
			continue
		}

		for _, container := range containers {
			seen[container.ID] = container
		}
	}

	result := make([]models.DockerContainerStats, 0, len(seen))
	for _, container := range seen {
		result = append(result, container)
	}

	slices.SortFunc(result, func(a, b models.DockerContainerStats) int {
		if cmp := strings.Compare(a.Stack, b.Stack); cmp != 0 {
			return cmp
		}
		if cmp := strings.Compare(a.Service, b.Service); cmp != 0 {
			return cmp
		}
		return strings.Compare(a.Name, b.Name)
	})

	return models.DockerStats{
		RunningContainers: len(result),
		Containers:        result,
		Errors:            errors,
	}, nil
}

func (s *service) getComposeContainers(ctx context.Context, composePath string) ([]models.DockerContainerStats, error) {
	composePath = strings.TrimSpace(composePath)
	if composePath == "" {
		return nil, fmt.Errorf("docker compose path is empty")
	}

	cmd := commandContext(ctx, "docker", "compose", "-f", composePath, "ps", "--format", "json")
	cmd.Dir = filepath.Dir(composePath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("docker compose ps failed for %s: %w: %s", composePath, err, strings.TrimSpace(string(output)))
	}

	psContainers, err := parseComposePSOutput(output)
	if err != nil {
		return nil, fmt.Errorf("parse docker compose ps output for %s: %w", composePath, err)
	}

	running := make([]composePSContainer, 0, len(psContainers))
	for _, container := range psContainers {
		if strings.EqualFold(container.State, "running") {
			running = append(running, container)
		}
	}

	if len(running) == 0 {
		return []models.DockerContainerStats{}, nil
	}

	inspectByID, err := s.inspectContainers(ctx, composePath, running)
	if err != nil {
		return nil, err
	}

	result := make([]models.DockerContainerStats, 0, len(running))
	for _, container := range running {
		stack := strings.TrimSpace(container.Project)
		if stack == "" {
			stack = deriveStackName(composePath)
		}

		startedAtRaw := ""
		uptimeSeconds := int64(0)
		uptimeHuman := "0s"

		if inspected, ok := findInspectedContainer(inspectByID, container); ok {
			startedAtRaw = inspected.State.StartedAt
			if startedAt, err := time.Parse(time.RFC3339Nano, inspected.State.StartedAt); err == nil {
				uptime := s.now().Sub(startedAt)
				if uptime < 0 {
					uptime = 0
				}
				uptimeSeconds = int64(uptime.Seconds())
				uptimeHuman = formatDuration(uptime)
			}
		}

		if uptimeSeconds == 0 {
			if parsedSeconds, parsedHuman, ok := parseDockerStatusUptime(container.Status); ok {
				uptimeSeconds = parsedSeconds
				uptimeHuman = parsedHuman
			}
		}

		result = append(result, models.DockerContainerStats{
			ID:            container.ID,
			Name:          container.Name,
			Service:       container.Service,
			Stack:         stack,
			ComposeFile:   composePath,
			State:         container.State,
			Status:        container.Status,
			StartedAt:     startedAtRaw,
			UptimeSeconds: uptimeSeconds,
			UptimeHuman:   uptimeHuman,
		})
	}

	return result, nil
}

func (s *service) inspectContainers(ctx context.Context, composePath string, containers []composePSContainer) (map[string]dockerInspectContainer, error) {
	args := make([]string, 0, len(containers)+1)
	args = append(args, "inspect")
	for _, container := range containers {
		args = append(args, container.ID)
	}

	cmd := commandContext(ctx, "docker", args...)
	cmd.Dir = filepath.Dir(composePath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("docker inspect failed for %s: %w: %s", composePath, err, strings.TrimSpace(string(output)))
	}

	var inspected []dockerInspectContainer
	if err := json.Unmarshal(output, &inspected); err != nil {
		return nil, fmt.Errorf("decode docker inspect output for %s: %w", composePath, err)
	}

	result := make(map[string]dockerInspectContainer, len(inspected))
	for _, container := range inspected {
		id := strings.TrimSpace(container.ID)
		if id == "" {
			continue
		}
		result[id] = container
	}

	return result, nil
}

func findInspectedContainer(inspected map[string]dockerInspectContainer, container composePSContainer) (dockerInspectContainer, bool) {
	id := strings.TrimSpace(container.ID)
	if id != "" {
		if match, ok := inspected[id]; ok {
			return match, true
		}
		for fullID, item := range inspected {
			if strings.HasPrefix(fullID, id) {
				return item, true
			}
		}
	}

	name := strings.TrimSpace(container.Name)
	if name != "" {
		for _, item := range inspected {
			inspectName := strings.TrimPrefix(strings.TrimSpace(item.Name), "/")
			if inspectName == name {
				return item, true
			}
		}
	}

	return dockerInspectContainer{}, false
}

func parseDockerStatusUptime(status string) (int64, string, bool) {
	status = strings.TrimSpace(status)
	if status == "" || !strings.HasPrefix(status, "Up ") {
		return 0, "", false
	}

	value := strings.TrimPrefix(status, "Up ")
	if idx := strings.Index(value, " ("); idx >= 0 {
		value = value[:idx]
	}
	value = strings.TrimSpace(value)

	switch value {
	case "Less than a second":
		return 0, "0s", true
	case "About a minute":
		return int64(time.Minute.Seconds()), "1m", true
	case "About an hour":
		return int64(time.Hour.Seconds()), "1h", true
	}

	parts := strings.Fields(value)
	if len(parts) < 2 {
		return 0, "", false
	}

	amount, err := parsePositiveInt64(parts[0])
	if err != nil {
		return 0, "", false
	}

	unit := strings.ToLower(parts[1])
	switch {
	case strings.HasPrefix(unit, "second"):
		return amount, formatDuration(time.Duration(amount) * time.Second), true
	case strings.HasPrefix(unit, "minute"):
		return amount * 60, formatDuration(time.Duration(amount) * time.Minute), true
	case strings.HasPrefix(unit, "hour"):
		return amount * 3600, formatDuration(time.Duration(amount) * time.Hour), true
	case strings.HasPrefix(unit, "day"):
		return amount * 86400, formatDuration(time.Duration(amount) * 24 * time.Hour), true
	case strings.HasPrefix(unit, "week"):
		return amount * 7 * 86400, formatDuration(time.Duration(amount) * 7 * 24 * time.Hour), true
	case strings.HasPrefix(unit, "month"):
		return amount * 30 * 86400, formatDuration(time.Duration(amount) * 30 * 24 * time.Hour), true
	case strings.HasPrefix(unit, "year"):
		return amount * 365 * 86400, formatDuration(time.Duration(amount) * 365 * 24 * time.Hour), true
	default:
		return 0, "", false
	}
}

func parsePositiveInt64(value string) (int64, error) {
	var parsed int64
	for _, ch := range value {
		if ch < '0' || ch > '9' {
			return 0, fmt.Errorf("invalid integer value %q", value)
		}
		parsed = parsed*10 + int64(ch-'0')
	}
	return parsed, nil
}

func (s *service) getMergenatorSettings() (mergenatorSettings, error) {
	var cfg mergenatorSettings

	settingsData, err := s.settingsService.Get("mergenator")
	if err != nil {
		return cfg, fmt.Errorf("failed to get mergenator settings: %w", err)
	}
	if settingsData == nil {
		return cfg, nil
	}

	raw, err := json.Marshal(settingsData)
	if err != nil {
		return cfg, fmt.Errorf("failed to marshal mergenator settings: %w", err)
	}
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return cfg, fmt.Errorf("failed to unmarshal mergenator settings: %w", err)
	}

	return cfg, nil
}

func parseComposePSOutput(raw []byte) ([]composePSContainer, error) {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" {
		return []composePSContainer{}, nil
	}

	var asArray []composePSContainer
	if err := json.Unmarshal([]byte(trimmed), &asArray); err == nil {
		return asArray, nil
	}

	lines := strings.Split(trimmed, "\n")
	result := make([]composePSContainer, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var item composePSContainer
		if err := json.Unmarshal([]byte(line), &item); err != nil {
			return nil, fmt.Errorf("invalid json line %q: %w", line, err)
		}
		result = append(result, item)
	}

	return result, nil
}

func deriveStackName(composePath string) string {
	base := filepath.Base(filepath.Dir(composePath))
	if base == "." || base == string(filepath.Separator) || base == "" {
		return filepath.Base(composePath)
	}
	return base
}

func formatDuration(duration time.Duration) string {
	if duration < time.Second {
		return "0s"
	}

	totalSeconds := int64(duration.Seconds())
	days := totalSeconds / 86400
	hours := (totalSeconds % 86400) / 3600
	minutes := (totalSeconds % 3600) / 60
	seconds := totalSeconds % 60

	parts := make([]string, 0, 2)
	if days > 0 {
		parts = append(parts, fmt.Sprintf("%dd", days))
	}
	if hours > 0 {
		parts = append(parts, fmt.Sprintf("%dh", hours))
	}
	if len(parts) < 2 && minutes > 0 {
		parts = append(parts, fmt.Sprintf("%dm", minutes))
	}
	if len(parts) == 0 {
		parts = append(parts, fmt.Sprintf("%ds", seconds))
	}

	return strings.Join(parts, " ")
}
