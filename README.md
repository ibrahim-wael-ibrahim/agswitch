# AGSwitch

A secure account switcher and responsive terminal dashboard for Google Antigravity.

AGSwitch stores credentials in the operating-system keyring, tracks the active account across access-token rotation, displays model quota, and switches accounts transactionally with rollback.

**Author:** Ibrahim Wael  
**Repository:** `github.com/ibrahim-wael-ibrahim/agswitch`

## Highlights

- Secure saved profiles in Secret Service on Linux.
- Stable account identity that survives renewed access tokens and expiry changes.
- Automatic synchronization of a renewed active credential back to its saved profile.
- Transactional credential activation with verification and rollback.
- Optional language-server-only hot switching that keeps the Electron UI, open files, chat and terminals alive.
- Cleanup of orphan language servers after a full application restart.
- Quota dashboard with concurrent retrieval and private caching.
- Safe auto-switch policy: stale, warned, unknown or old quota never changes accounts.
- Conservative error classification: temporary server overload does not authorize an account switch.
- Bubble Tea dashboard, JSON output, diagnostics and an upgrade-aware installer.

## Installation and updates

### Stable release

```bash
curl -fsSL https://raw.githubusercontent.com/ibrahim-wael-ibrahim/agswitch/master/scripts/install.sh | bash
```

The installer downloads a compatible release when available. Otherwise it installs the required build dependencies, builds from source and atomically replaces the existing binary. The previous binary is restored if the new binary fails validation.

### Test this development branch

```bash
curl -fsSL https://raw.githubusercontent.com/ibrahim-wael-ibrahim/agswitch/agent/safe-hot-switch/scripts/install.sh \
  | AGSWITCH_REF=agent/safe-hot-switch AGSWITCH_BUILD_FROM_SOURCE=true bash
```

Or from a local clone:

```bash
git fetch origin
git switch agent/safe-hot-switch
AGSWITCH_BUILD_FROM_SOURCE=true ./scripts/install.sh
```

Verify the installation:

```bash
agswitch version
agswitch config
agswitch doctor
```

Useful installer variables:

| Variable | Purpose |
| --- | --- |
| `AGSWITCH_VERSION` | Install a specific tagged release |
| `AGSWITCH_REF` | Source branch or tag to build; default `master` |
| `AGSWITCH_BUILD_FROM_SOURCE` | Set to `true` to bypass releases and build the selected ref |
| `BINDIR` | Binary destination; default `~/.local/bin` |

On Arch/Omarchy, the installer uses `pacman` and maintains the desktop launcher, floating Hyprland rule and `SUPER+SHIFT+CTRL+A` shortcut.

## First-time setup

```bash
agswitch doctor
agswitch migrate
agswitch list
agswitch current
```

Save the currently authenticated Antigravity account:

```bash
agswitch save work
```

Repeat after signing Antigravity into each account you own.

## Dashboard

```bash
agswitch
```

or:

```bash
agswitch tui
```

Common keys:

| Key | Action |
| --- | --- |
| `Tab`, arrows, `h`, `j`, `k`, `l` | Navigate |
| `/` | Search accounts |
| `Enter` | Select or execute |
| `r` | Refresh quota |
| `a` | Apply an auto-switch recommendation |
| `p` | Previous account |
| `d` | Diagnostics |
| `q`, `Ctrl-C` | Quit |

The dashboard follows the terminal palette and adapts to narrow and wide terminals.

## Account management

```bash
agswitch save work
agswitch update work
agswitch clone work work-backup
agswitch rename work company
agswitch info company
agswitch list
agswitch current
agswitch detect
agswitch delete company
```

## Switching modes

### Preserve launch state

```bash
agswitch use work
```

This uses the established full application transaction when Antigravity is already running.

### Full restart

```bash
agswitch use work --restart
```

AGSwitch closes Antigravity, cleans related language servers, changes the credential and starts Antigravity again.

### Switch without starting

```bash
agswitch use work --no-start
```

### Hot switch while keeping the UI open

First wait until the current response and all tool calls have finished. Then run:

```bash
agswitch use work --hot-reload --confirm-idle
```

The hot transaction is:

```text
acquire lock
→ load and back up the active credential
→ write and verify the selected credential
→ stop all matching old language-server processes
→ wait for Electron to create a new language server PID
→ commit current/previous account state
```

If backend replacement fails, AGSwitch restores the previous credential and reloads the language server again.

`--confirm-idle` is deliberately required. Current Antigravity logs do not expose a reliable marker for the end of an entire agent task; one task may contain several streamed requests and tool calls.

To select Antigravity IDE instead of standalone Antigravity, configure both paths before running AGSwitch:

```bash
export AGSWITCH_APP_PATH=/opt/antigravity-ide/antigravity-ide
export AGSWITCH_LANGUAGE_SERVER_PATH=/opt/antigravity-ide/resources/app/extensions/antigravity/bin/language_server_linux_x64
```

## Stable account detection

Antigravity renews access tokens and expiry fields. A raw hash of the complete credential therefore changes even though the Google account is unchanged.

AGSwitch now separates:

- **Payload fingerprint:** detects any credential JSON change and verifies keyring writes.
- **Identity fingerprint:** identifies the account using Google subject, then email, then a one-way hash derived from the refresh token when those claims are absent.

Raw identity material is never written to `accounts.json`. When the active payload rotates, `agswitch current` matches the stable identity and synchronizes the renewed credential into the corresponding saved profile.

Existing metadata is upgraded lazily when profiles are next read; no manual migration is required.

## Quota

```bash
agswitch quota
agswitch quota --refresh
agswitch quota work
agswitch quota --json
```

Quota retrieval provides:

- Bounded concurrent requests.
- Five-minute private cache for display.
- Stale-cache fallback for visibility when the provider is unavailable.
- Model percentage, exhaustion state and reset time when returned by Google.
- Unknown values displayed as `unknown` rather than invented percentages.

Google Cloud Code endpoints used by Antigravity are internal and may change.

### Important authentication limitation

The observed Antigravity credential format includes access and refresh tokens but does not include its OAuth client credentials. Expired inactive profiles therefore cannot currently be refreshed independently unless compatible values are explicitly supplied:

```bash
export AGSWITCH_OAUTH_CLIENT_ID=...
export AGSWITCH_OAUTH_CLIENT_SECRET=...
```

Do not extract or publish secrets embedded in third-party binaries. Prefer a supported authentication flow. Until one is available, activate an account through Antigravity so it renews its credential, then let AGSwitch synchronize that renewed payload.

## Safe auto-switch

Preview:

```bash
agswitch auto-switch --refresh --dry-run
```

Apply while Antigravity is stopped:

```bash
agswitch auto-switch --refresh --threshold 20
```

Keep the UI open after confirming the current task is idle:

```bash
agswitch auto-switch --refresh --threshold 20 --hot-reload --confirm-idle
```

Use a full application restart instead:

```bash
agswitch auto-switch --refresh --threshold 20 --force-running
```

An account is eligible only when its snapshot:

- came from `google-cloud-code`;
- has no provider warning;
- has a fetch timestamp;
- is no older than two minutes;
- contains at least one known model quota.

`cache-stale` is display-only and can never trigger switching. If the active account is unknown or lacks recent live quota, AGSwitch refuses to switch rather than guessing.

The score is the lowest known remaining percentage across the account's models. AGSwitch switches only below the threshold and only when another eligible account has a higher score.

## Error classification

The monitor classifier groups adjacent request logs conservatively:

- `server_overloaded`: temporary provider overload; retry/backoff only, never switch account.
- `resource_exhausted_ambiguous`: insufficient evidence; never switch account.
- `five_hour_quota_exhausted`, `weekly_quota_exhausted`, or explicit `quota_exhausted`: eligible to inform future automation.

The project does not intercept HTTPS requests, replace bearer tokens mid-request, or replay failed requests under another account.

An unattended daemon remains intentionally disabled until a reliable whole-task idle signal and supported live quota for inactive profiles are available.

## Process safety

A full restart now stops the Electron application and then verifies that related language-server processes are gone. This prevents an old authenticated backend from surviving as an orphan after a switch.

Hot reload stops every matching old language-server PID and succeeds only after all old PIDs disappear and a new replacement PID appears.

Display resolved process paths:

```bash
agswitch config
```

Relevant variables:

| Variable | Purpose |
| --- | --- |
| `AGSWITCH_APP_PATH` | Electron application executable |
| `AGSWITCH_LANGUAGE_SERVER_PATH` | Backend executable restarted during hot switching |
| `AGSWITCH_QUIT_COMMAND` | Optional desktop-specific graceful quit command |
| `AGSWITCH_GRACEFUL_TIMEOUT` | Shutdown timeout, for example `12s` |
| `AGSWITCH_FORCE_KILL` | Set `false` to disable forced cleanup |
| `AGSWITCH_OAUTH_CLIENT_ID` | Optional supported OAuth client ID |
| `AGSWITCH_OAUTH_CLIENT_SECRET` | Optional supported OAuth client secret |

## Storage and security

Active Antigravity credential:

```text
service=gemini
username=antigravity
```

Saved profiles:

```text
service=agswitch
username=profile:<name>
```

Private local files:

```text
~/.config/agswitch/accounts.json
~/.local/state/agswitch/state.json
~/.local/state/agswitch/antigravity.log
~/.cache/agswitch/quota.json
```

Credential payloads remain in the operating-system keyring. Never commit tokens, refresh tokens, cookies or OAuth client secrets.

## Validation after upgrading

```bash
agswitch doctor
agswitch current --json
agswitch quota --refresh --json
agswitch auto-switch --refresh --dry-run --json
```

Then perform one controlled hot switch while no task is running:

```bash
agswitch use TARGET_PROFILE --hot-reload --confirm-idle
```

Confirm that:

- `agswitch current` reports the target profile;
- the Electron window, open files, chat and terminals remain;
- a simple new request succeeds;
- the old language-server PID is gone;
- a new language-server PID exists;
- no switch is recommended from `cache-stale` quota.

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

Main packages:

```text
cmd/                 commands and dependency wiring
internal/app/        profile lifecycle and self-healing identity
internal/switcher/   transactional full and hot activation
internal/process/    process, backend reload and orphan cleanup
internal/quota/      provider, cache and freshness validation
internal/autoswitch/ conservative candidate selection
internal/monitor/    overload and quota incident classification
internal/tui/        responsive terminal dashboard
```

## Platform status

- Linux keyring, CLI, dashboard and full switching: validated.
- Language-server-only restart: validated on Omarchy/Arch with Antigravity standalone.
- Antigravity IDE backend path: configurable and requires a controlled test on each installed version.
- Windows process integration: preview.
