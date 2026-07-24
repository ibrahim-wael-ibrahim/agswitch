package cmd

import (
	"github.com/ibrahim-wael/agswitch/internal/account"
	agsapp "github.com/ibrahim-wael/agswitch/internal/app"
	"github.com/ibrahim-wael/agswitch/internal/config"
	"github.com/ibrahim-wael/agswitch/internal/doctor"
	agskeyring "github.com/ibrahim-wael/agswitch/internal/keyring"
	agslock "github.com/ibrahim-wael/agswitch/internal/lock"
	agsprocess "github.com/ibrahim-wael/agswitch/internal/process"
	agsstate "github.com/ibrahim-wael/agswitch/internal/state"
	"github.com/ibrahim-wael/agswitch/internal/switcher"
	"github.com/spf13/cobra"
	"time"
)

type dependencies struct {
	config config.Config
	app    *agsapp.Service
	doctor doctor.Service
}

func Execute() error { return newRootCommand(buildDependencies()).Execute() }
func buildDependencies() *dependencies {
	cfg := config.Default()
	a := account.NewFileRepository(cfg.AccountsPath)
	active := agskeyring.NewActiveStore()
	profiles := agskeyring.NewProfileStore()
	l := agslock.New(cfg.LockPath)
	det := agsprocess.LinuxDetector{}
	p := &agsprocess.Manager{Executable: cfg.AppPath, Detector: det, Launcher: agsprocess.LinuxLauncher{LogPath: cfg.LogPath}, Quitter: agsprocess.CommandThenSignalQuitter{Command: cfg.QuitCommand, Detector: det, Timeout: 4 * time.Second, Fallback: agsprocess.SignalQuitter{Detector: det, Timeout: cfg.GracefulTimeout, Force: cfg.ForceKill}}}
	sw := &switcher.Service{ActiveStore: active, ProfileStore: profiles, Process: p, State: agsstate.New(cfg.StatePath), Locker: l}
	app := &agsapp.Service{Active: active, Profiles: profiles, Accounts: a, Switcher: sw, Locker: l}
	return &dependencies{cfg, app, doctor.Service{Config: cfg, Active: active, Accounts: a, Process: p}}
}
func newRootCommand(d *dependencies) *cobra.Command {
	r := &cobra.Command{Use: "agswitch", Short: "Switch Antigravity accounts safely", SilenceUsage: true, SilenceErrors: true, RunE: func(c *cobra.Command, _ []string) error { return runTUI(c, d, false) }}
	r.AddCommand(newTUICommand(d), newUseCommand(d), newSaveCommand(d), newListCommand(d), newCurrentCommand(d), newDeleteCommand(d), newMigrateCommand(d), newQuotaCommand(), newDoctorCommand(d), newConfigCommand(d))
	return r
}
