package logger

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"time"

	"mergenator/internal/config"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/pkgerrors"
	"gopkg.in/natefinch/lumberjack.v2"
)

var log zerolog.Logger

func Init(cfg *config.Config) (io.Closer, error) {
	logDir := filepath.Dir(cfg.LogFile)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, err
	}

	// Настройка ротации логов
	rotatingWriter := &lumberjack.Logger{
		Filename:   cfg.LogFile,
		MaxSize:    cfg.LogMaxSize,
		MaxBackups: cfg.LogMaxBackups,
		MaxAge:     cfg.LogMaxAge,
		Compress:   cfg.LogCompress,
		LocalTime:  true,
	}

	// Настраиваем уровень логирования
	level, err := zerolog.ParseLevel(cfg.LogLevel)
	if err != nil {
		level = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(level)

	// Настраиваем обработку ошибок со стектрейсом
	zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack

	// Настраиваем формат времени
	zerolog.TimeFieldFormat = time.RFC3339

	var writers []io.Writer
	writers = append(writers, rotatingWriter)

	// Если нужен вывод в консоль
	if cfg.LogConsole {
		consoleWriter := zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: "15:04:05",
			NoColor:    false,
		}
		writers = append(writers, consoleWriter)
	}

	multiWriter := zerolog.MultiLevelWriter(writers...)

	log = zerolog.New(multiWriter).
		With().
		Timestamp().
		Caller().
		Str("service", "mergenator").
		Logger()

	// Логируем успешный запуск
	log.Info().
		Str("log_file", cfg.LogFile).
		Str("log_level", cfg.LogLevel).
		Msg("Logger initialized")

	return rotatingWriter, nil
}

// Глобальный логгер
func Get() *zerolog.Logger {
	return &log
}

// Convenience functions

func Debug() *zerolog.Event {
	return log.Debug()
}

func Info() *zerolog.Event {
	return log.Info()
}

func Warn() *zerolog.Event {
	return log.Warn()
}

func Error() *zerolog.Event {
	return log.Error()
}

func Fatal() *zerolog.Event {
	return log.Fatal()
}

func Panic() *zerolog.Event {
	return log.Panic()
}

func WithContext(ctx context.Context) *zerolog.Logger {
	return zerolog.Ctx(ctx)
}
