package logger

import (
	"context"
	"fmt"

	"gopher-equalizer/config"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	RequestID = "RequestID"
	LoggerKey = "logger"
)

type Logger struct {
	l *zap.Logger
}

func New(ctx context.Context, cfg *config.Config) (context.Context, error) {
	// Добавляем энкодер времени вручную
	// Это функция, поэтому из yml его не достать
	cfg.Logger.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	logger, err := cfg.Logger.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to create logger: %w", err)
	}

	ctx = context.WithValue(ctx, LoggerKey, &Logger{l: logger})

	return ctx, nil
}

func GetLoggerFromCtx(ctx context.Context) *Logger {
	return ctx.Value(LoggerKey).(*Logger)
}

func (l *Logger) Info(ctx context.Context, msg string, fields ...zap.Field) {
	if ctx.Value(RequestID) != nil {
		fields = append(fields, zap.String(RequestID, ctx.Value(RequestID).(string)))
	}

	l.l.Info(msg, fields...)
}

func (l *Logger) Debug(ctx context.Context, msg string, fields ...zap.Field) {
	if ctx.Value(RequestID) != nil {
		fields = append(fields, zap.String(RequestID, ctx.Value(RequestID).(string)))
	}

	l.l.Debug(msg, fields...)
}

func (l *Logger) Error(ctx context.Context, msg string, fields ...zap.Field) {
	if ctx.Value(RequestID) != nil {
		fields = append(fields, zap.String(RequestID, ctx.Value(RequestID).(string)))
	}

	l.l.Error(msg, fields...)
}
