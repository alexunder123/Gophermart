package config

import (
	"errors"
	"flag"

	"github.com/caarlos0/env/v6"
)

type Config struct {
	RunAddress  string `env:"RUN_ADDRESS"`
	DatabaseURI string `env:"DATABASE_URI"`
	AccuralSystemAddress string `env:"ACCRUAL_SYSTEM_ADDRESS"`
}

func NewConfig() (*Config, error) {
	var config Config

	err := env.Parse(&config)
	if err != nil {
		return nil, err
	}

	if config.RunAddress == "" {
		flag.StringVar(&config.RunAddress, "a", "127.0.0.1:8080", "Адрес запускаемого сервера")
	}
	if config.DatabaseURI == "" {
		flag.StringVar(&config.DatabaseURI, "d", "", "База данных SQL")
	}
	if config.AccuralSystemAddress == "" {
		flag.StringVar(&config.AccuralSystemAddress, "r", "", "Сервер расчета начислений")
	}

	flag.Parse()

	if config.RunAddress == "" {
		return nil, errors.New("server address not provided")
	}
	if config.DatabaseURI == "" {
		return nil, errors.New("storage address not provided")
	}
	if config.AccuralSystemAddress == "" {
		return nil, errors.New("accural address not provided")
	}

	return &config, nil
}
