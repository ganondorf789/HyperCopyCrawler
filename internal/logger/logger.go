package logger

import (
	"os"
	"path/filepath"
	"strconv"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// Init configures zap with lumberjack file rotation and redirects std log output.
func Init(appName string) (*zap.Logger, func(), error) {
	logFile := getEnv("LOG_FILE", filepath.Join("logs", appName+".log"))
	if err := os.MkdirAll(filepath.Dir(logFile), 0o755); err != nil {
		return nil, nil, err
	}

	level := zapcore.InfoLevel
	if err := level.UnmarshalText([]byte(getEnv("LOG_LEVEL", "info"))); err != nil {
		level = zapcore.InfoLevel
	}

	rotator := &lumberjack.Logger{
		Filename:   logFile,
		MaxSize:    getEnvInt("LOG_MAX_SIZE_MB", 100),
		MaxBackups: getEnvInt("LOG_MAX_BACKUPS", 7),
		MaxAge:     getEnvInt("LOG_MAX_AGE_DAYS", 30),
		Compress:   getEnvBool("LOG_COMPRESS", true),
	}

	encoderCfg := zap.NewProductionEncoderConfig()
	encoderCfg.EncodeTime = zapcore.ISO8601TimeEncoder

	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderCfg),
		zapcore.NewMultiWriteSyncer(zapcore.AddSync(os.Stdout), zapcore.AddSync(rotator)),
		level,
	)

	l := zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))
	zap.ReplaceGlobals(l)
	undoStdLog := zap.RedirectStdLog(l)

	cleanup := func() {
		undoStdLog()
		_ = l.Sync()
		_ = rotator.Close()
	}

	return l, cleanup, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return parsed
}

func getEnvBool(key string, fallback bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}
	return parsed
}
