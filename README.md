# AGSwitch

A secure account switcher and responsive terminal dashboard for Google Antigravity.

AGSwitch stores saved credentials in the operating-system keyring, detects the active account by credential fingerprint, switches transactionally, displays model quota, and exposes both interactive and script-friendly workflows.

**Author:** Ibrahim Wael  
**GitHub:** `ibrahim-wael-ibrahim`  
**Repository:** `github.com/ibrahim-wael-ibrahim/agswitch`

## Features

- Responsive Bubble Tea dashboard with keyboard navigation.
- Split layout for commands, account search and quota details.
- Model quota displayed in one or two columns depending on terminal width.
- Terminal-native styling using standard ANSI attributes instead of a fixed RGB palette.
- Secure profile storage in Secret Service on Linux.
- Transactional switching with lock, verification and rollback.
- Account save, update, clone, rename, delete and migration workflows.
- Cached concurrent quota retrieval with stale-cache fallback.
- Conservative quota-based auto-switch recommendations.
- JSON output for automation and integrations.
- Environment diagnostics through `agswitch doctor`.
- Tagged release binaries and a dependency-aware installer.

## Dashboard

Launch the dashboard with:

```bash
agswitch
```

or explicitly:

```bash
agswitch tui
```

The upper half contains two responsive panels:

1. **Program & Commands** — choose an operation and execute it inside AGSwitch.
2. **Search & Accounts** — filter, select and inspect saved accounts.

The lower half displays quota for the selected account. On wide terminals, models are displayed in two columns. On narrow terminals, panels and quota rows stack automatically.

### Keyboard controls

| Key | Action |
| --- | --- |
| `Tab`, `Left`, `Right`, `h`, `l` | Move between command and account panels |
| `Up`, `Down`, `j`, `k` | Move inside the focused panel |
| `/` | Search accounts |
| `Enter` | Select an account or execute the selected command |
| `r` | Refresh live quota |
| `a` | Apply the auto-switch recommendation |
| `p` | Switch to the previous account |
| `d` | Run diagnostics |
| `q`, `Ctrl-C` | Quit |

The dashboard deliberately uses terminal defaults: bold, dim and reverse-video attributes. It does not impose a hard-coded color scheme, so it follows the palette configured in the terminal emulator.

## Dashboard commands

The command panel provides:

- **Switch + launch** — activate the selected profile and start Antigravity.
- **Switch only** — activate the selected profile without launching Antigravity.
- **Update profile** — save the current Antigravity credential into the selected profile.
- **Refresh quota** — bypass the five-minute quota cache.
- **Auto switch** — apply the conservative recommendation.
- **Previous account** — return to the previously active account.
- **Run doctor** — inspect dependencies, storage and process state.
- **Quit dashboard** — exit without changing accounts.

## Installation

### One-command installer

From a cloned repository:

```bash
./scripts/install.sh
```

Remote installer after the project is published on the default branch:

```bash
curl -fsSL https://raw.githubusercontent.com/ibrahim-wael-ibrahim/agswitch/master/scripts/install.sh | bash
```

The installer first attempts to download the matching release binary. If no compatible binary exists, it installs build dependencies and compiles from source.

Override the installation location:

```bash
BINDIR="$HOME/bin" ./scripts/install.sh
```

Install a specific release:

```bash
AGSWITCH_VERSION=v0.1.0 ./scripts/install.sh
```

### Arch Linux

The installer uses `pacman` and installs:

```text
ca-certificates curl git go libsecret procps-ng systemd
```

Manual setup:

```bash
sudo pacman -Sy --needed ca-certificates curl git go libsecret procps-ng systemd
make check
make install
```

### Fedora

The installer uses `dnf` and installs:

```text
ca-certificates curl git golang libsecret procps-ng systemd
```

Manual setup:

```bash
sudo dnf install ca-certificates curl git golang libsecret procps-ng systemd
make check
make install
```

### Debian and Ubuntu

The installer uses `apt-get` and installs:

```text
ca-certificates curl git golang-go libsecret-tools procps systemd
```

Manual setup:

```bash
sudo apt-get update
sudo apt-get install ca-certificates curl git golang-go libsecret-tools procps systemd
make check
make install
```

### Windows

The shell installer recognises Git Bash, MSYS2 and Cygwin. It can use `winget` or Chocolatey to install Git and Go, and it looks for a Windows release binary named:

```text
agswitch_windows_amd64.exe
```

Run from Git Bash:

```bash
./scripts/install.sh
```

**Support status:** the dashboard, account metadata, quota client and Windows Credential Manager dependency can be built through Go, but the Antigravity process-control adapter must be validated against the real Windows Antigravity executable before Windows is considered stable. Linux is the currently validated platform.

## First-time setup

Run diagnostics:

```bash
agswitch doctor
```

Import legacy JSON profiles:

```bash
agswitch migrate
agswitch list
agswitch current
```

Do not delete source files until every migrated account has been tested. After validation:

```bash
agswitch migrate --force --delete-source
```

## Account management

```bash
agswitch save work
agswitch update work
agswitch clone work work-backup
agswitch rename work company
agswitch info company
agswitch info company --json
agswitch list
agswitch list --json
agswitch current
agswitch current --json
agswitch detect
agswitch delete company
```

### Switching

Preserve the current launch state:

```bash
agswitch use work
```

Always start Antigravity:

```bash
agswitch use work --restart
```

Leave Antigravity stopped:

```bash
agswitch use work --no-start
```

Return to the previous profile:

```bash
agswitch previous
```

## Transaction model

A switch is performed as a transaction:

```text
acquire operation lock
→ validate selected profile
→ back up the active credential
→ detect application state
→ stop Antigravity when required
→ write the selected credential
→ read it back and verify the fingerprint
→ restore the requested launch state
→ verify startup
→ commit current/previous state
```

If activation or startup fails, AGSwitch attempts to restore the previous credential and process state.

## Quota

```bash
agswitch quota
agswitch quota --refresh
agswitch quota work
agswitch quota --json
```

Quota retrieval includes:

- Bounded concurrent account requests.
- Five-minute private cache.
- Stale-cache fallback when the provider is unavailable.
- Access-token usage and refresh support when compatible OAuth client data exists.
- Endpoint fallback and response-size limits.
- Unknown quota displayed as `unknown`, never invented as `0%`.
- Duplicate display names grouped into variants using the lowest known quota.

Google Cloud Code quota endpoints are internal and may change without notice. The provider is isolated from the rest of the application so it can be updated independently.

When Antigravity writes a renewed access token, update the saved profile:

```bash
agswitch use work --restart
agswitch update work
agswitch quota work --refresh
```

Do not commit tokens, refresh tokens or OAuth client secrets.

## Auto switch

Preview the decision:

```bash
agswitch auto-switch --dry-run --refresh
```

Choose a threshold:

```bash
agswitch auto-switch --threshold 30 --dry-run
```

Apply the recommendation while Antigravity is stopped:

```bash
agswitch auto-switch --threshold 30 --refresh
```

Explicitly allow interruption when the application is running:

```bash
agswitch auto-switch --threshold 30 --refresh --force-running
```

The score for each account is its lowest known model quota. Accounts with failed or completely unknown quota are ignored. AGSwitch does not switch when the current account is above the threshold or when no better candidate exists.

## Storage and security

### Active Antigravity credential

```text
service=gemini
username=antigravity
```

### Saved profiles

```text
service=agswitch
username=profile:<name>
```

### Non-secret local files

```text
~/.config/agswitch/accounts.json
~/.local/state/agswitch/state.json
~/.local/state/agswitch/antigravity.log
~/.cache/agswitch/quota.json
```

Only metadata, state, logs and quota summaries are stored on disk. Saved credential payloads remain in the operating-system keyring.

## Process shutdown

Configure a tested desktop-specific graceful quit command when required:

```bash
export AGSWITCH_QUIT_COMMAND="$HOME/.config/agswitch/quit-antigravity"
```

On Linux, the fallback detects the Antigravity process, sends `SIGTERM`, waits for shutdown and optionally uses `SIGKILL`. D-Bus tray services are diagnosed, but the exact Antigravity tray action remains desktop-specific.

## Configuration

Display resolved paths and settings:

```bash
agswitch config
```

Useful environment variables:

| Variable | Purpose |
| --- | --- |
| `AGSWITCH_APP_PATH` | Antigravity executable path |
| `AGSWITCH_QUIT_COMMAND` | Custom graceful shutdown command |
| `AGSWITCH_GRACEFUL_TIMEOUT` | Shutdown timeout, such as `12s` |
| `AGSWITCH_FORCE_KILL` | Set to `false` to disable forced termination |
| `AGSWITCH_VERSION` | Release selected by the installer |
| `BINDIR` | Installer/build destination |

## Version and project identity

```bash
agswitch version
agswitch version --json
```

The output includes the resolved version, author, repository and Go runtime version.

## Development

```bash
make fmt
make tidy
make vet
make test
make race
make build
make check
```

Run the dashboard from source:

```bash
go run .
```

The main packages are separated by responsibility:

```text
cmd/                 Cobra commands and dependency wiring
internal/app/        profile lifecycle operations
internal/switcher/   transactional activation
internal/process/    platform process adapters
internal/keyring/    secure credential stores
internal/quota/      provider, cache and formatting
internal/autoswitch/ conservative selection policy
internal/tui/        responsive Bubble Tea dashboard
```

## Release

Create a semantic-version tag:

```bash
git tag v0.1.0
git push origin v0.1.0
```

The release workflow runs tests, builds supported architecture binaries, creates SHA-256 checksums and publishes GitHub release assets.

## Uninstall

```bash
make uninstall
```

or:

```bash
./scripts/uninstall.sh
```

Uninstalling the binary does not automatically delete keyring profiles or user state.

## Project status

- Linux CLI, keyring storage, switching, migration and quota: validated.
- Responsive interactive dashboard: implemented and covered by repository validation.
- Arch, Fedora, Debian/Ubuntu installer flows: implemented.
- Windows installer detection: implemented.
- Windows Antigravity process integration: preview until validated on a real Windows installation.
