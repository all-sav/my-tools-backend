package settings

import (
	"fmt"
	settingsrepo "mergenator/internal/repository/settings"
)

type service struct {
	repo settingsrepo.SettingsRepository
}

func NewSettingsService(repo settingsrepo.SettingsRepository) SettingsService {
	return &service{
		repo: repo,
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

func (s *service) Update() {

}
