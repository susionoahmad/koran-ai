package logger

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Logger interface {
	Debug(msg string, fields ...zap.Field)
	Info(msg string, fields ...zap.Field)
	Warn(msg string, fields ...zap.Field)
	Error(msg string, fields ...zap.Field)
	Fatal(msg string, fields ...zap.Field)
	Sync() error
	GetZapLogger() *zap.Logger
}

type zapLogger struct {
	log *zap.Logger
}

// NewLogger creates a new structured Zap logger based on env and level settings.
func NewLogger(env string, levelStr string) (Logger, error) {
	var level zapcore.Level
	if err := level.UnmarshalText([]byte(levelStr)); err != nil {
		level = zapcore.DebugLevel
	}

	var encoderConfig zapcore.EncoderConfig
	var encoder zapcore.Encoder

	if env == "production" {
		encoderConfig = zap.NewProductionEncoderConfig()
		encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	} else {
		encoderConfig = zap.NewDevelopmentEncoderConfig()
		encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		encoderConfig.EncodeTime = zapcore.TimeEncoderOfLayout("2006-01-02 15:04:05.000")
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	}

	core := zapcore.NewCore(
		encoder,
		zapcore.Lock(os.Stdout),
		level,
	)

	log := zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))

	return &zapLogger{log: log}, nil
}

func (l *zapLogger) Debug(msg string, fields ...zap.Field) {
	l.log.Debug(msg, fields...)
}

func (l *zapLogger) Info(msg string, fields ...zap.Field) {
	l.log.Info(msg, fields...)
}

func (l *zapLogger) Warn(msg string, fields ...zap.Field) {
	l.log.Warn(msg, fields...)
}

func (l *zapLogger) Error(msg string, fields ...zap.Field) {
	l.log.Error(msg, fields...)
}

func (l *zapLogger) Fatal(msg string, fields ...zap.Field) {
	l.log.Fatal(msg, fields...)
}

func (l *zapLogger) Sync() error {
	return l.log.Sync()
}

func (l *zapLogger) GetZapLogger() *zap.Logger {
	return l.log
}
