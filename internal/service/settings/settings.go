package settings

import (
	"encoding/json"
	"fmt"
	"mergenator/internal/infr/logger"
	settingsrepo "mergenator/internal/repository/settings"
	"slices"
	"strings"

	"github.com/rs/zerolog"
)

var availableModules = []string{"mergenator"}

type service struct {
	repo settingsrepo.SettingsRepository
	log  *zerolog.Logger
}

func NewSettingsService(repo settingsrepo.SettingsRepository) SettingsService {
	return &service{
		repo: repo,
		log:  logger.Get(),
	}
}

func (s *service) Get(module string) (any, error) {
	data, err := s.repo.Get(module)
	if err != nil {
		return nil, fmt.Errorf("failed to get settings of %s module", module)
	}

	// todo: Тут наверное можно сохранять в память и в дальнейшем отдавать настройки из памяти(в Update, который ниже - обновлять в памяти после сохранения)

	return data, nil
}

func (s *service) Update(module string, data any) error {
	toSave := data
	switch module {
	case "mergenator":
		merged, err := s.mergeMergenatorSettings(data)
		if err != nil {
			s.log.Err(err).Msg("mergenator settings merge failed")
			return err
		}
		toSave = merged
	default:
		return fmt.Errorf("module %s not found", module)
	}

	if err := s.repo.Save(module, toSave); err != nil {
		s.log.Err(err).Msg("settings saving error")
		return err
	}

	s.log.Info().Interface("data", toSave).Msg("settings was updated")
	return nil
}

func (s *service) mergeMergenatorSettings(incoming any) (map[string]any, error) {
	existing, err := s.repo.Get("mergenator")
	if err != nil {
		return nil, fmt.Errorf("read existing mergenator settings: %w", err)
	}

	out := map[string]any{}
	if existing != nil {
		raw, err := json.Marshal(existing)
		if err != nil {
			return nil, fmt.Errorf("marshal existing settings: %w", err)
		}
		if err := json.Unmarshal(raw, &out); err != nil {
			return nil, fmt.Errorf("decode existing settings: %w", err)
		}
	}

	inc := map[string]any{}
	incRaw, err := json.Marshal(incoming)
	if err != nil {
		return nil, fmt.Errorf("marshal incoming settings: %w", err)
	}
	if err := json.Unmarshal(incRaw, &inc); err != nil {
		return nil, fmt.Errorf("decode incoming settings: %w", err)
	}

	for k, v := range inc {
		out[k] = v
	}

	if _, ok := out["docker_compose_paths"]; ok {
		out["docker_compose_paths"] = normalizeDockerComposePaths(out["docker_compose_paths"])
	}

	return out, nil
}

func normalizeDockerComposePaths(v any) []string {
	if v == nil {
		return nil
	}
	switch t := v.(type) {
	case []string:
		return trimNonEmptyStrings(t)
	case []any:
		out := make([]string, 0, len(t))
		for _, e := range t {
			s, ok := e.(string)
			if !ok {
				continue
			}
			s = strings.TrimSpace(s)
			if s != "" {
				out = append(out, s)
			}
		}
		return out
	default:
		raw, err := json.Marshal(v)
		if err != nil {
			return nil
		}
		var asAny []any
		if err := json.Unmarshal(raw, &asAny); err != nil {
			return nil
		}
		return normalizeDockerComposePaths(asAny)
	}
}

func trimNonEmptyStrings(in []string) []string {
	out := make([]string, 0, len(in))
	for _, s := range in {
		s = strings.TrimSpace(s)
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}

func (s *service) HasModule(module string) bool {
	return slices.Contains(availableModules, module)
}
