package log

import (
	"log"
	"os"
	"path/filepath"
	"regexp"
	"runtime"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

const (
	DebugLevel = "debug"
	InfoLevel  = "info"
	WarnLevel  = "warn"
	ErrorLevel = "error"
)

type Config struct {
	HasStdout   bool   // print to console
	Dev         bool   // development(true) or production(false)
	LogLevel    string // log level
	LogPath     string // log path
	Filename    string // log filename
	ErrFilename string // log error filename
	MaxSize     int    // the limit size of log file, Mb
	MaxBackups  int    //  the number of log file
	MaxAge      int    // days for retain the log file
	Compress    bool
}

var defaultConfig = &Config{
	HasStdout:   true,
	Dev:         false,
	LogLevel:    InfoLevel,
	LogPath:     "./logs",
	Filename:    "info.log",
	ErrFilename: "err.log",
	MaxSize:     10,
	MaxBackups:  5,
	MaxAge:      30,
	Compress:    false,
}

var Sugar *zap.SugaredLogger
var Logger *zap.Logger

func getLogLevel(str string) zapcore.Level {
	var level zapcore.Level
	switch str {
	case DebugLevel:
		level = zap.DebugLevel
	case InfoLevel:
		level = zap.InfoLevel
	case WarnLevel:
		level = zap.WarnLevel
	case ErrorLevel:
		level = zap.ErrorLevel
	default:
		level = zap.InfoLevel
	}
	return level
}

func updateDefaultConfig(config Config) {
	defaultConfig.HasStdout = config.HasStdout
	defaultConfig.Dev = config.Dev
	defaultConfig.LogLevel = config.LogLevel

	if config.MaxSize > 0 && config.MaxSize < 500 {
		defaultConfig.MaxSize = config.MaxSize
	}

	if config.MaxBackups > 0 && config.MaxBackups < 10 {
		defaultConfig.MaxBackups = config.MaxBackups
	}
	if config.MaxAge > 0 && config.MaxAge < 60 {
		defaultConfig.MaxAge = config.MaxAge
	}
	logPath := config.LogPath
	if logPath != "" && logPath != "./" {
		defaultConfig.LogPath = logPath
	}
}

func createLogPath(logPath string) error {
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		absRegexp, _ := regexp.Compile(`^(/|([a-zA-Z]:\\)).*`)
		if !absRegexp.MatchString(logPath) {
			_, currentFilePath, _, _ := runtime.Caller(1)
			workPath := filepath.Join(filepath.Dir(currentFilePath), "../")
			logPath = filepath.Join(workPath, logPath)
		}
	}
	if err := os.MkdirAll(logPath, os.ModePerm); err != nil {
		return err
	}
	return nil
}

func InitLogger(config Config) {
	updateDefaultConfig(config)

	if err := createLogPath(defaultConfig.LogPath); err != nil {
		log.Fatal("create logPath failed, err: ", err)
	}

	var writers zapcore.WriteSyncer
	var errWriters zapcore.WriteSyncer

	logWriter, errWriter := getLogWriter()
	if defaultConfig.HasStdout {
		writers = zapcore.NewMultiWriteSyncer(logWriter, zapcore.AddSync(os.Stdout))
		errWriters = zapcore.NewMultiWriteSyncer(errWriter)
	} else {
		writers = zapcore.NewMultiWriteSyncer(logWriter)
	}

	encoder := getEncoder()
	c := zapcore.NewCore(encoder, writers, zap.NewAtomicLevelAt(getLogLevel(defaultConfig.LogLevel)))
	errC := zapcore.NewCore(encoder, errWriters, zap.ErrorLevel)

	core := zapcore.NewTee(c, errC)
	opts := []zap.Option{zap.AddCaller()}
	opts = append(opts, zap.AddStacktrace(zap.ErrorLevel))
	opts = append(opts, zap.AddCallerSkip(0))
	if defaultConfig.Dev {
		opts = append(opts, zap.Development())
	}
	Logger = zap.New(core, opts...)
	zap.ReplaceGlobals(Logger)
	Sugar = Logger.Sugar()
}

func getLogWriter() (zapcore.WriteSyncer, zapcore.WriteSyncer) {
	logger := &lumberjack.Logger{
		Filename:   filepath.Join(defaultConfig.LogPath, defaultConfig.Filename),
		MaxSize:    defaultConfig.MaxSize,
		MaxBackups: defaultConfig.MaxBackups,
		MaxAge:     defaultConfig.MaxAge,
		Compress:   defaultConfig.Compress,
	}
	errLogger := &lumberjack.Logger{
		Filename:   filepath.Join(defaultConfig.LogPath, defaultConfig.ErrFilename),
		MaxSize:    defaultConfig.MaxSize,
		MaxBackups: defaultConfig.MaxBackups,
		MaxAge:     defaultConfig.MaxAge,
		Compress:   defaultConfig.Compress,
	}
	return zapcore.AddSync(logger), zapcore.AddSync(errLogger)
}

func getEncoder() zapcore.Encoder {
	encoderConfig := zap.NewProductionEncoderConfig()
	if defaultConfig.Dev {
		encoderConfig = zap.NewDevelopmentEncoderConfig()
	}
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	// NewConsoleEncoder for human readable output
	return zapcore.NewConsoleEncoder(encoderConfig)
}
