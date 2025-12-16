package logger

import (
	"context"
	"os"
	"runtime"
	"strings"

	"github.com/sirupsen/logrus"
)

type Logger interface {
	Warn(ctx context.Context, msg string, args ...any)
	Error(ctx context.Context, msg string, args ...any)
	Info(ctx context.Context, msg string, args ...any)
	Debug(ctx context.Context, msg string, args ...any)
}

type logrusLogger struct {
	logger *logrus.Logger
}

// getCallerFunctionName 获取调用者的函数名
func getCallerFunctionName() string {
	pc := make([]uintptr, 10)
	runtime.Callers(3, pc)
	funcName := runtime.FuncForPC(pc[0]).Name()
	// 提取最后一个点之后的部分作为函数名
	parts := strings.Split(funcName, ".")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return "unknown"
}

func (l *logrusLogger) Warn(ctx context.Context, msg string, args ...any) {
	args = append([]any{getCallerFunctionName()}, args...)
	l.logger.WithContext(ctx).Warnf("[%s] "+msg, args...)
}

func (l *logrusLogger) Error(ctx context.Context, msg string, args ...any) {
	args = append([]any{getCallerFunctionName()}, args...)
	l.logger.WithContext(ctx).Errorf("[%s] "+msg, args...)
}

func (l *logrusLogger) Info(ctx context.Context, msg string, args ...any) {
	args = append([]any{getCallerFunctionName()}, args...)
	l.logger.WithContext(ctx).Infof("[%s] "+msg, args...)
}

func (l *logrusLogger) Debug(ctx context.Context, msg string, args ...any) {
	args = append([]any{getCallerFunctionName()}, args...)
	l.logger.WithContext(ctx).Debugf("[%s] "+msg, args...)
}

var defaultLogger Logger

type LoggerConfig struct {
	Level string `json:"level,omitempty" yaml:"level,omitempty" toml:"level,omitempty"`
	File  string `json:"file,omitempty" yaml:"file,omitempty" toml:"file,omitempty"`
}

func InitLogger(cfg *LoggerConfig) {
	log := logrus.New()
	level, err := logrus.ParseLevel(cfg.Level)
	if err != nil {
		level = logrus.InfoLevel
	}
	log.SetLevel(level)
	if cfg.File != "" {
		fileWriter, err := os.OpenFile(cfg.File, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if err != nil {
			log.Fatalf("Failed to open log file %s: %v", cfg.File, err)
		}
		log.SetOutput(fileWriter)
	}

	defaultLogger = &logrusLogger{logger: log}
}

func Warn(ctx context.Context, msg string, args ...any) {
	defaultLogger.Warn(ctx, msg, args...)
}

func Error(ctx context.Context, msg string, args ...any) {
	defaultLogger.Error(ctx, msg, args...)
}

func Info(ctx context.Context, msg string, args ...any) {
	defaultLogger.Info(ctx, msg, args...)
}

func Debug(ctx context.Context, msg string, args ...any) {
	defaultLogger.Debug(ctx, msg, args...)
}
