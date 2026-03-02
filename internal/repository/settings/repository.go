package settingsrepo

type SettingsRepository interface {
	Get(dataKey string) (any, error)
	Save(dataKey string, settings any) error
}
