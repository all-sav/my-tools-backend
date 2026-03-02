package filerepo

// todo: доделать(хранение файла - где храним? прикольно бы сделать где-то в /var/lib/mytools, но учитывать os)

import (
	"encoding/json"
	"log"
	settingsrepo "mergenator/internal/repository/settings"
	"os"
	"path/filepath"

	gap "github.com/muesli/go-app-paths"
)

// const fileName string = "settings.json"

type settingsFileRepository struct {
	scope *gap.Scope
}

func NewSettingsFileRepository() settingsrepo.SettingsRepository {
	return &settingsFileRepository{
		scope: gap.NewScope(gap.System, "mytools"),
	}
}

func (r *settingsFileRepository) Get(dataKey string) (any, error) {
	configPath, _ := r.scope.ConfigPath(dataKey)
	var cfg any

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			cfgDir := filepath.Dir(configPath)
			if err := os.MkdirAll(cfgDir, 0755); err != nil {
				log.Fatal(err)
			}

			return cfg, err
		}
		return cfg, err
	}

	err = json.Unmarshal(data, &cfg)
	log.Default().Println(cfg)
	return cfg, err
}

func (r *settingsFileRepository) Save(dataKey string, settings any) error {
	configPath, err := r.scope.ConfigPath(dataKey)
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0644)
}
