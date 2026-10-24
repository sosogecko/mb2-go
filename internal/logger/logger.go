package logger

import (
	"os"
	"path/filepath"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"

	"mb2-go/internal/config"
)

type Loggers struct {
	Access *zap.Logger
	App    *zap.Logger
}

func New(cfg config.LogConfig, debug bool) (*Loggers, error) {
	if err := os.MkdirAll(cfg.Dir, 0o755); err != nil {
		return nil, err
	}

	encoderCfg := zap.NewProductionEncoderConfig()
	encoderCfg.TimeKey = "time"
	encoderCfg.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderCfg.EncodeDuration = zapcore.StringDurationEncoder
	jsonEnc := zapcore.NewJSONEncoder(encoderCfg)

	consoleCfg := encoderCfg
	consoleCfg.EncodeLevel = zapcore.CapitalColorLevelEncoder
	consoleEnc := zapcore.NewConsoleEncoder(consoleCfg)

	accessWS := zapcore.AddSync(&lumberjack.Logger{
		Filename:   filepath.Join(cfg.Dir, "access.log"),
		MaxSize:    cfg.MaxSizeMB,
		MaxAge:     cfg.MaxAgeDays,
		MaxBackups: cfg.MaxAgeDays,
		Compress:   cfg.Compress,
		LocalTime:  true,
	})
	errorWS := zapcore.AddSync(&lumberjack.Logger{
		Filename:   filepath.Join(cfg.Dir, "error.log"),
		MaxSize:    cfg.MaxSizeMB,
		MaxAge:     cfg.MaxAgeDays,
		MaxBackups: cfg.MaxAgeDays,
		Compress:   cfg.Compress,
		LocalTime:  true,
	})
	appWS := zapcore.AddSync(&lumberjack.Logger{
		Filename:   filepath.Join(cfg.Dir, "app.log"),
		MaxSize:    cfg.MaxSizeMB,
		MaxAge:     cfg.MaxAgeDays,
		MaxBackups: cfg.MaxAgeDays,
		Compress:   cfg.Compress,
		LocalTime:  true,
	})

	level := zapcore.InfoLevel
	if debug {
		level = zapcore.DebugLevel
	}

	console := zapcore.Lock(os.Stdout)
	accessCore := zapcore.NewTee(
		zapcore.NewCore(jsonEnc, accessWS, zap.NewAtomicLevelAt(zapcore.InfoLevel)),
		zapcore.NewCore(consoleEnc, console, zap.NewAtomicLevelAt(level)),
	)
	appCore := zapcore.NewTee(
		zapcore.NewCore(jsonEnc, appWS, zap.NewAtomicLevelAt(level)),
		zapcore.NewCore(jsonEnc, errorWS, zap.NewAtomicLevelAt(zapcore.ErrorLevel)),
		zapcore.NewCore(consoleEnc, console, zap.NewAtomicLevelAt(level)),
	)

	return &Loggers{
		Access: zap.New(accessCore),
		App:    zap.New(appCore, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel)),
	}, nil
}
