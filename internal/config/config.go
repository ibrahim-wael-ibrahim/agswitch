package config

import (
	"os"
	"path/filepath"
)

type Config struct {
	BaseDir    string
	StateDir   string
	AppPath    string
	LogPath    string
	QuotaCache string
}

func Default() Config {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}

	configHome := envOrJoin("XDG_CONFIG_HOME", home, ".config")
	stateHome := envOrJoin("XDG_STATE_HOME", home, ".local", "state")
	cacheHome := envOrJoin("XDG_CACHE_HOME", home, ".cache")

	baseDir := filepath.Join(configHome, "agswitch")
	stateDir := filepath.Join(stateHome, "agswitch")
	cacheDir := filepath.Join(cacheHome, "agswitch")

	return Config{
		BaseDir:    baseDir,
		StateDir:   stateDir,
		AppPath:    "/opt/Antigravity/antigravity",
		LogPath:    filepath.Join(stateDir, "agswitch.log"),
		QuotaCache: filepath.Join(cacheDir, "quota.json"),
	}
}

func envOrJoin(name string, home string, parts ...string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}

	items := append([]string{home}, parts...)
	return filepath.Join(items...)
}
