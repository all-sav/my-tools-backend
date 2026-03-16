package settings

type SettingsService interface {
	Get(module string) (any, error)
	Update(module string, data any) error
	HasModule(module string) bool
}
