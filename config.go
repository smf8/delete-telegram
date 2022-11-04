package main

import "time"

type Config struct {
	ServerTimeout time.Duration
	APIKey        string
	ListenPort    int

	DatabasePath          string
	DatabaseEncryptionKey string

	TelegramAppID   int
	TelegramAppHash string
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
