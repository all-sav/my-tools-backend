package settings

type SettingsService interface {
	Get(module string) (any, error)
	Update()
}
