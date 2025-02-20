package logger

import (
	"go.uber.org/zap"
)

type Logger struct {
	Log *zap.Logger
}

func New() *Logger {
	return &Logger{
		Log: zap.NewNop(),
	}
}

func (l *Logger) Init(level string) error {
	// преобразуем текстовый уровень логирования в zap.AtomicLevel
	lvl, err := zap.ParseAtomicLevel(level)
	if err != nil {
		return err
	}
	// создаём новую конфигурацию логера
	cfg := zap.NewProductionConfig()
	// устанавливаем уровень
	cfg.Level = lvl
	// создаём логер на основе конфигурации
	zl, err := cfg.Build()
	if err != nil {
		return err
	}
	// устанавливаем синглтон
	l.Log = zl
	return nil
}
