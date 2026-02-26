package db

import (
	"database/sql"
	"log"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type MergenatorSettings struct {
	StandID             int    `json:"stand_id"`
	BackendProjectID    string `json:"backend_project_id"`
	FrontendProjectID   string `json:"frontend_project_id"`
	BackendStandBranch  string `json:"backend_stand_branch"`
	FrontendStandBranch string `json:"frontend_stand_branch"`
	CIMainBranch        string `json:"ci_main_branch"`
	RequiredPrefix      string `json:"required_prefix"`
	Prefix              string `json:"prefix"`
	CIPrefix            string `json:"ci_prefix"`
	UpdatedAt           int64  `json:"updated_at"`
}

var db *sql.DB

func InitDB() {
	var err error
	db, err = sql.Open("sqlite3", "./mytools.db")
	if err != nil {
		log.Fatal("Failed to open database:", err)
	}

	createTableSQL := `
    CREATE TABLE IF NOT EXISTS mergenator_settings (
        stand_id INTEGER PRIMARY KEY,
        backend_project_id TEXT NOT NULL DEFAULT '',
        frontend_project_id TEXT NOT NULL DEFAULT '',
        backend_stand_branch TEXT NOT NULL DEFAULT '',
        frontend_stand_branch TEXT NOT NULL DEFAULT '',
        ci_main_branch TEXT NOT NULL DEFAULT '',
        required_prefix TEXT NOT NULL DEFAULT '',
        prefix TEXT NOT NULL DEFAULT '',
        ci_prefix TEXT NOT NULL DEFAULT '',
        updated_at INTEGER NOT NULL
    );`

	_, err = db.Exec(createTableSQL)
	if err != nil {
		log.Fatal("Failed to create table:", err)
	}

	log.Println("Database initialized successfully")
}

func CloseDB() {
	if db != nil {
		db.Close()
	}
}

// Сохранить или обновить настройки пользователя
func SaveMergenatorSettings(settings MergenatorSettings) error {
	settings.UpdatedAt = time.Now().Unix()

	query := `
    INSERT OR REPLACE INTO mergenator_settings (
        stand_id, backend_project_id, frontend_project_id,
        backend_stand_branch, frontend_stand_branch, ci_main_branch,
        required_prefix, prefix, ci_prefix, updated_at
    ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := db.Exec(query,
		settings.StandID,
		settings.BackendProjectID,
		settings.FrontendProjectID,
		settings.BackendStandBranch,
		settings.FrontendStandBranch,
		settings.CIMainBranch,
		settings.RequiredPrefix,
		settings.Prefix,
		settings.CIPrefix,
		settings.UpdatedAt,
	)
	return err
}

// Получить настройки мерженатора
func GetMergenatorSettings() (*MergenatorSettings, error) {
	query := `
    SELECT stand_id, backend_project_id, frontend_project_id,
           backend_stand_branch, frontend_stand_branch, ci_main_branch,
           required_prefix, prefix, ci_prefix, updated_at
    FROM mergenator_settings LIMIT 1`

	var settings MergenatorSettings
	err := db.QueryRow(query).Scan(
		&settings.StandID,
		&settings.BackendProjectID,
		&settings.FrontendProjectID,
		&settings.BackendStandBranch,
		&settings.FrontendStandBranch,
		&settings.CIMainBranch,
		&settings.RequiredPrefix,
		&settings.Prefix,
		&settings.CIPrefix,
		&settings.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil // Настроек нет
	}
	if err != nil {
		return nil, err
	}
	return &settings, nil
}

// Удалить настройки пользователя
func DeleteUserSettings(gitlabUserID int) error {
	query := "DELETE FROM user_settings WHERE gitlab_user_id = ?"
	_, err := db.Exec(query, gitlabUserID)
	return err
}
