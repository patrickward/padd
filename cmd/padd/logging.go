package main

import (
	"io"
	"log"
	"os"
	"path/filepath"

	"gopkg.in/natefinch/lumberjack.v2"
)

type LogConfig struct {
	LogFile    string // Log file path
	MaxSize    int    // Max size in megabytes
	MaxBackups int    // Max number of backups
	MaxAge     int    // Max age in days
	Compress   bool   // Compress backups
}

func DefaultLogConfig(dataDir string) LogConfig {
	return LogConfig{
		LogFile:    filepath.Join(dataDir, "service", "padd.log"),
		MaxSize:    100,
		MaxBackups: 3,
		MaxAge:     30,
		Compress:   true,
	}
}

func SetupLogging(config LogConfig) error {
	if err := os.MkdirAll(filepath.Dir(config.LogFile), 0755); err != nil {
		return err
	}

	logger := &lumberjack.Logger{
		Filename:   config.LogFile,
		MaxSize:    config.MaxSize,
		MaxBackups: config.MaxBackups,
		MaxAge:     config.MaxAge,
		Compress:   config.Compress,
	}

	multiWriter := io.MultiWriter(os.Stdout, logger)
	log.SetOutput(multiWriter)

	return nil
}
