package config

import (
	"log/slog"
	"os"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	SSHPrivateKeyPath string `toml:"ssh_private_key_path" mapstructure:"ssh_private_key_path"`
	SSHPort           int    `toml:"ssh_port" mapstructure:"ssh_port"`
}

var C *Config

func InitConfig() {
	if C != nil {
		return
	}
	viper.SetConfigFile("config.toml")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	if err := viper.ReadInConfig(); err != nil {
		slog.Error("failed to read config file", "err", err)
		os.Exit(1)
	}
	C = &Config{}
	if err := viper.Unmarshal(C); err != nil {
		slog.Error("failed to unmarshal config", "err", err)
		os.Exit(1)
	}
	slog.Debug("config loaded", "ssh_private_key_path", C.SSHPrivateKeyPath, "ssh_port", C.SSHPort)
}
