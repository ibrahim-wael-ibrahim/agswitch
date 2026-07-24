package config

import (
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Config struct {
	BaseDir         string
	StateDir        string
	CacheDir        string
	AccountsPath    string
	StatePath       string
	LockPath        string
	AppPath         string
	LogPath         string
	QuotaCache      string
	QuitCommand     []string
	GracefulTimeout time.Duration
	ForceKill       bool
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

	appPath := os.Getenv("AGSWITCH_APP_PATH")
	if appPath == "" {
		appPath = "/opt/Antigravity/antigravity"
	}
	timeout := 8 * time.Second
	if value := os.Getenv("AGSWITCH_GRACEFUL_TIMEOUT"); value != "" {
		if parsed, parseErr := time.ParseDuration(value); parseErr == nil && parsed > 0 {
			timeout = parsed
		}
	}
	forceKill := !strings.EqualFold(os.Getenv("AGSWITCH_FORCE_KILL"), "false")
	var quitCommand []string
	if value := strings.TrimSpace(os.Getenv("AGSWITCH_QUIT_COMMAND")); value != "" {
		quitCommand = strings.Fields(value)
	}

	return Config{
		BaseDir:         baseDir,
		StateDir:        stateDir,
		CacheDir:        cacheDir,
		AccountsPath:    filepath.Join(baseDir, "accounts.json"),
		StatePath:       filepath.Join(stateDir, "state.json"),
		LockPath:        filepath.Join(stateDir, "agswitch.lock"),
		AppPath:         appPath,
		LogPath:         filepath.Join(stateDir, "antigravity.log"),
		QuotaCache:      filepath.Join(cacheDir, "quota.json"),
		QuitCommand:     quitCommand,
		GracefulTimeout: timeout,
		ForceKill:       forceKill,
	}
}

func envOrJoin(name string, home string, parts ...string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	items := append([]string{home}, parts...)
	return filepath.Join(items...)
}
