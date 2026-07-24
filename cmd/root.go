package cmd

import (
	"time"

	"github.com/ibrahim-wael/agswitch/internal/account"
	agsapp "github.com/ibrahim-wael/agswitch/internal/app"
	"github.com/ibrahim-wael/agswitch/internal/config"
	"github.com/ibrahim-wael/agswitch/internal/doctor"
	agskeyring "github.com/ibrahim-wael/agswitch/internal/keyring"
	agslock "github.com/ibrahim-wael/agswitch/internal/lock"
	agsprocess "github.com/ibrahim-wael/agswitch/internal/process"
	"github.com/ibrahim-wael/agswitch/internal/quota"
	agsstate "github.com/ibrahim-wael/agswitch/internal/state"
	"github.com/ibrahim-wael/agswitch/internal/switcher"
	"github.com/spf13/cobra"
)

type dependencies struct {
	config  config.Config
	app     *agsapp.Service
	quota   *quota.Service
	doctor  doctor.Service
	process *agsprocess.Manager
	state   *agsstate.Store
}

func Execute() error {
	return newRootCommand(buildDependencies()).Execute()
}

func buildDependencies() *dependencies {
	cfg := config.Default()
	accounts := account.NewFileRepository(cfg.AccountsPath)
	active := agskeyring.NewActiveStore()
	profiles := agskeyring.NewProfileStore()
	locker := agslock.New(cfg.LockPath)
	detector := agsprocess.LinuxDetector{}
	processManager := &agsprocess.Manager{
		Executable: cfg.AppPath,
		Detector:   detector,
		Launcher: agsprocess.LinuxLauncher{
			LogPath:     cfg.LogPath,
			MaxLogBytes: 5 << 20,
		},
		Quitter: agsprocess.CommandThenSignalQuitter{
			Command:  cfg.QuitCommand,
			Detector: detector,
			Timeout:  4 * time.Second,
			Fallback: agsprocess.SignalQuitter{
				Detector: detector,
				Timeout:  cfg.GracefulTimeout,
				Force:    cfg.ForceKill,
			},
		},
	}
	stateStore := agsstate.New(cfg.StatePath)
	switchService := &switcher.Service{
		ActiveStore:  active,
		ProfileStore: profiles,
		Process:      processManager,
		State:        stateStore,
		Locker:       locker,
	}
	appService := &agsapp.Service{
		Active:   active,
		Profiles: profiles,
		Accounts: accounts,
		Switcher: switchService,
		Locker:   locker,
	}
	quotaService := &quota.Service{
		Profiles: profiles,
		Provider: quota.NewGoogleProvider(),
		Cache: &quota.FileCache{
			Path: cfg.QuotaCache,
			TTL:  5 * time.Minute,
		},
		Concurrency: 4,
	}
	return &dependencies{
		config:  cfg,
		app:     appService,
		quota:   quotaService,
		process: processManager,
		state:   stateStore,
		doctor: doctor.Service{
			Config:   cfg,
			Active:   active,
			Accounts: accounts,
			Process:  processManager,
		},
	}
}

func newRootCommand(dependencies *dependencies) *cobra.Command {
	root := &cobra.Command{
		Use:           "agswitch",
		Short:         "Switch Antigravity accounts safely",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(command *cobra.Command, _ []string) error {
			return runTUI(command, dependencies, false)
		},
	}
	root.AddCommand(
		newTUICommand(dependencies),
		newUseCommand(dependencies),
		newAutoSwitchCommand(dependencies),
		newPreviousCommand(dependencies),
		newSaveCommand(dependencies),
		newUpdateCommand(dependencies),
		newCloneCommand(dependencies),
		newRenameCommand(dependencies),
		newInfoCommand(dependencies),
		newDetectCommand(dependencies),
		newListCommand(dependencies),
		newCurrentCommand(dependencies),
		newStatusCommand(dependencies),
		newDeleteCommand(dependencies),
		newMigrateCommand(dependencies),
		newQuotaCommand(dependencies),
		newDoctorCommand(dependencies),
		newConfigCommand(dependencies),
		newVersionCommand(),
	)
	return root
}
