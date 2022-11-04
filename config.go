package main

import (
	"github.com/knadh/koanf"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/providers/structs"
	"github.com/sirupsen/logrus"
	"time"
)

type Config struct {
	ServerTimeout time.Duration `koanf:"server_timeout"`
	APIKey        string        `koanf:"api_key"`
	ListenPort    int           `koanf:"listen_port"`

	DatabasePath          string `koanf:"database_path"`
	DatabaseEncryptionKey string `koanf:"database_encryption_key"`

	TelegramAppID   int    `koanf:"telegram_app_id"`
	TelegramAppHash string `koanf:"telegram_app_hash"`
}

var defaultConfig = &Config{
	ServerTimeout:         10 * time.Second,
	APIKey:                "some_random_strong_shit",
	ListenPort:            1455,
	DatabasePath:          "./data",
	DatabaseEncryptionKey: "twenty__four__char__shit",
	TelegramAppID:         0,
	TelegramAppHash:       "",
}

func InitConfig() *Config {
	var cfg Config

	k := koanf.New(".")

	if err := k.Load(structs.Provider(defaultConfig, "koanf"), nil); err != nil {
		logrus.Fatalf("error loading default: %s", err)
	}

	if err := k.Load(file.Provider("config.yaml"), yaml.Parser()); err != nil {
		logrus.Errorf("error loading config.yml: %s", err)
	}

	if err := k.Unmarshal("", &cfg); err != nil {
		logrus.Fatalf("error unmarshalling config: %s", err)
	}

	logrus.Infof("loaded config: %+v", cfg)

	return &cfg
}
