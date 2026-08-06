package cmd

import (
	"time"

	"github.com/ibrahim-wael/agswitch/internal/account"
	appsvc "github.com/ibrahim-wael/agswitch/internal/app"
	"github.com/ibrahim-wael/agswitch/internal/config"
	"github.com/ibrahim-wael/agswitch/internal/doctor"
	keyringpkg "github.com/ibrahim-wael/agswitch/internal/keyring"
	lockpkg "github.com/ibrahim-wael/agswitch/internal/lock"
	processpkg "github.com/ibrahim-wael/agswitch/internal/process"
	"github.com/ibrahim-wael/agswitch/internal/quota"
	statepkg "github.com/ibrahim-wael/agswitch/internal/state"
	"github.com/ibrahim-wael/agswitch/internal/switcher"
	"github.com/spf13/cobra"
)

type dependencies struct {
	config  config.Config
	app     *appsvc.Service
	quota   *quota.Service
	doctor  doctor.Service
	process *processpkg.Manager
	state   *statepkg.Store
}

func Execute() error {
	return newRootCommand(buildDependencies()).Execute()
}

func buildDependencies() *dependencies {
	cfg := config.Default()
	accounts := account.NewFileRepository(cfg.AccountsPath)
	active := keyringpkg.NewActiveStore()
	profiles := keyringpkg.NewProfileStore()
	locker := lockpkg.New(cfg.LockPath)
	detector := processpkg.LinuxDetector{}
	processManager := &processpkg.Manager{
		Executable: cfg.AppPath,
		Detector:   detector,
		Launcher: processpkg.LinuxLauncher{
			LogPath:     cfg.LogPath,
			MaxLogBytes: 5 << 20,
		},
		Quitter: processpkg.CommandThenSignalQuitter{
			Command:  cfg.QuitCommand,
			Detector: detector,
			Timeout:  4 * time.Second,
			Fallback: processpkg.SignalQuitter{
				Detector: detector,
				Timeout:  cfg.GracefulTimeout,
				Force:    cfg.ForceKill,
			},
		},
		Backend: processpkg.LanguageServerReloader{
			Executable: cfg.LanguageServerPath,
			Timeout:    12 * time.Second,
			Force:      cfg.ForceKill,
		},
	}
	stateStore := statepkg.New(cfg.StatePath)
	switchService := &switcher.Service{
		ActiveStore:  active,
		ProfileStore: profiles,
		Process:      processManager,
		State:        stateStore,
		Locker:       locker,
	}
	appService := &appsvc.Service{
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
			return runTUI(command, dependencies, true)
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
