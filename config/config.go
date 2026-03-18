package config

import (
	"log"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Bot struct {
		Token    string `toml:"token"`
		Username string `toml:"username"`
		OwnerId  int64  `toml:"owner_id"`
	}
	Settings struct {
		MetricsEnabled bool `toml:"metrics_enabled"`
	}
}

var Cfg Config

func LoadConfig(path string) {
	if _, err := toml.DecodeFile(path, &Cfg); err != nil {
		log.Fatalf("❌ Не вдалося прочитати конфіг: %s", err)
	}
}
