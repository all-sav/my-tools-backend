package dbrepo

import (
	"encoding/json"
	"fmt"
	settingsrepo "mergenator/internal/repository/settings"
	"os"
	"path/filepath"
	"time"

	"go.etcd.io/bbolt"
)

const (
	bucketName = "settings"
	// settingsKey = "mergenator"
)

type settingsDBRepository struct {
	db     *bbolt.DB
	dbPath string
}

type Options struct {
	// Можно указать конкретный путь, иначе будет определен автоматически
	DataDir string
	// Имя файла базы данных
	DBName string
}

func NewSettingsDBRepository(opts ...Options) (settingsrepo.SettingsRepository, error) {
	var opt Options
	if len(opts) > 0 {
		opt = opts[0]
	}

	// Определяем путь к данным
	dataDir := opt.DataDir
	if dataDir == "" {
		dataDir = "../../data"
	}

	// Создаем директорию если её нет
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory %s: %w", dataDir, err)
	}

	// Определяем имя файла БД
	dbName := opt.DBName
	if dbName == "" {
		dbName = "mytools.db"
	}
	dbPath := filepath.Join(dataDir, dbName)

	// Открываем БД
	db, err := bbolt.Open(dbPath, 0600, &bbolt.Options{
		Timeout: 1 * time.Second,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to open bbolt database at %s: %w", dbPath, err)
	}

	// Создаем bucket для настроек если его нет
	err = db.Update(func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(bucketName))
		return err
	})
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create settings bucket: %w", err)
	}

	// defer db.Close()

	return &settingsDBRepository{
		db:     db,
		dbPath: dbPath,
	}, nil
}

func (r *settingsDBRepository) Get(dataKey string) (any, error) {
	var settings any

	err := r.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketName))
		if bucket == nil {
			return fmt.Errorf("settings bucket not found")
		}

		data := bucket.Get([]byte(dataKey))
		if data == nil {
			// todo: что-то придумать если настройки не найдены?
			return nil
		}

		return json.Unmarshal(data, &settings)
	})

	if err != nil {
		return settings, fmt.Errorf("failed to read settings: %w", err)
	}

	return settings, nil
}

func (r *settingsDBRepository) Save(dataKey string, settings any) error {
	// todo: добавить вавалидацию

	data, err := json.Marshal(settings)
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %w", err)
	}

	err = r.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketName))
		if bucket == nil {
			return fmt.Errorf("settings bucket not found")
		}

		return bucket.Put([]byte(dataKey), data)
	})

	if err != nil {
		return fmt.Errorf("failed to save settings: %w", err)
	}

	return nil
}

func (r *settingsDBRepository) Close() error {
	if r.db != nil {
		return r.db.Close()
	}
	return nil
}
