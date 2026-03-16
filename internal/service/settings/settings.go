package settings

import (
	"fmt"
	"mergenator/internal/infr/logger"
	settingsrepo "mergenator/internal/repository/settings"
	"slices"

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
	if err := s.repo.Save(module, data); err != nil {
		s.log.Err(err).Msg("settings saving error")
		return err
	}

	s.log.Info().Interface("data", data).Msg("settings was updated")
	return nil
}

func (s *service) HasModule(module string) bool {
	return slices.Contains(availableModules, module)
}
