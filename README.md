# AGSwitch

A secure account switcher and realtime terminal dashboard for Google Antigravity.

**Current release:** `v1.1.0`  
**Author:** Ibrahim Wael  
**Repository:** `github.com/ibrahim-wael-ibrahim/agswitch`

AGSwitch keeps saved Antigravity profiles in the operating-system keyring, tracks the active account across access-token rotation, displays live quota, and switches profiles transactionally with rollback.

## v1.1.0 highlights

- Responsive Bubble Tea v2 + Lip Gloss v2 dashboard.
- Semantic colors with dark/light modes and `NO_COLOR` support.
- Quota-first account list with the current five-hour window emphasized.
- Live quota sweep for all saved profiles on startup and every 60 seconds by default.
- Bounded retry/backoff for temporary provider failures.
- Explicit `LIVE`, `AUTH`, `RETRY`, `STALE`, `OLD`, and `UNKNOWN` states.
- Five-hour and long/weekly reset windows derived from provider `resetTime`; no invented weekly percentage.
- Compact model summary with optional full model details.
- Stable profile identity that survives access-token rotation.
- Language-server-only hot switching that preserves the Electron UI, files, chat, and terminals.
- Full-restart recovery if a language-server hot reload cannot recover.
- Safe OAuth diagnostics with `agswitch auth doctor`.
- Antigravity-assisted credential renewal with `agswitch auth refresh-via-antigravity`.
- Dynamic build versioning: `master` builds use `VERSION`, feature branches use `dev+<sha>`.
- Installer works both as a local script and through `curl | bash`.

## Install or update

Install the current `master` release from source:

```bash
curl -fsSL \
  https://raw.githubusercontent.com/ibrahim-wael-ibrahim/agswitch/master/scripts/install.sh \
  | AGSWITCH_REF=master \
    AGSWITCH_BUILD_FROM_SOURCE=true \
    bash
```

Then verify:

```bash
agswitch version
agswitch doctor
agswitch current --json
```

A `master` source build reads the repository `VERSION` file and should report:

```text
agswitch v1.1.0
```

Feature-branch builds remain identifiable, for example:

```text
agswitch v0.0.0-dev+794527a4
```

The installer validates the new binary and atomically replaces the old one. On Arch/Omarchy it also maintains the desktop launcher, floating Hyprland rule, and `SUPER+SHIFT+CTRL+A` shortcut.

## Dashboard

Open AGSwitch:

```bash
agswitch
```

or:

```bash
agswitch tui
```

The dashboard immediately requests quota for every saved profile and refreshes all profiles every minute by default.

Typical account row:

```text
● ibrahim-wael-2       [LIVE]  5h 83% · 48m
○ work                 [AUTH]
○ backup               [RETRY]
```

The selected-account panel shows the email, data age, model count, quota source, five-hour window, and a long/weekly window only when the provider actually exposes one.

### Quota states

| State | Meaning |
| --- | --- |
| `LIVE` | Recent Google Cloud Code quota; safe for recommendations |
| `AUTH` | Saved credential cannot currently obtain live quota |
| `RETRY` | Temporary provider/network failure; retry automatically |
| `STALE` | Cached/provider-warning data; display-only |
| `OLD` | Previously live snapshot beyond the safe freshness limit |
| `UNKNOWN` | No trustworthy percentage is available |

Cached, warned, old, or unknown quota can never authorize auto-switching.

### Five-hour and weekly display

AGSwitch uses actual provider model fields such as `remainingFraction`, `resetTime`, and `isExhausted`.

- reset within roughly six hours → **5 HOUR** window;
- farther reset up to roughly eight days → **WEEKLY / long** window;
- no long reset → the UI says the provider did not expose a separate weekly window.

The account-level percentage is the most constrained known model in that window, which makes it useful for identifying the account that is currently exhausted or closest to exhaustion.

### Model display

Press `m` for a compact grouped summary. Press `M` for all model rows.

For non-live profiles, cached model percentages are marked as display-only and are not presented as current quota.

### Keyboard shortcuts

| Key | Action |
| --- | --- |
| `↑`, `↓`, `j`, `k` | Select account |
| `Enter`, `s` | Hot switch selected account |
| `f` | Full restart with selected account |
| `u` | Sync renewed active credential into the selected saved profile |
| `o` | Activate selected credential without restarting Antigravity |
| `r` | Refresh quota for all accounts |
| `a` | Preview/confirm automatic recommendation |
| `/` | Search profiles |
| `m` | Compact model summary |
| `M` | Full model details |
| `q`, `Ctrl-C` | Quit |

## Realtime quota for inactive accounts

Antigravity credentials may contain an access token and refresh token without the OAuth client secret required for AGSwitch to independently refresh inactive accounts.

Use the diagnostics command first:

```bash
agswitch auth doctor
```

Sanitized JSON output:

```bash
agswitch auth doctor --json > ~/Desktop/agswitch-auth.json
```

The report does **not** print access tokens, refresh tokens, or the client secret. It can show token client information, scopes, direct refresh status, and whether the current access token can call the quota endpoint.

### Recommended renewal path: let Antigravity refresh the credential

When direct OAuth refresh is unavailable, AGSwitch can temporarily activate a saved profile and let Antigravity renew it using Antigravity's own authenticated flow:

```bash
agswitch auth refresh-via-antigravity work \
  --confirm-idle \
  --timeout 45s
```

For all saved quota-enabled profiles:

```bash
agswitch auth refresh-via-antigravity \
  --all \
  --confirm-idle \
  --timeout 45s
```

Machine-readable report:

```bash
agswitch auth refresh-via-antigravity \
  --all \
  --confirm-idle \
  --timeout 45s \
  --json \
  > ~/Desktop/agswitch-antigravity-refresh.json
```

The operation is intentionally opt-in because it restarts the Antigravity backend for each target profile.

The flow is:

```text
remember original profile
→ skip profiles that already have recent live quota
→ activate target credential
→ restart the language server
→ wait for Antigravity to publish a renewed credential
→ synchronize the renewed credential into the saved profile
→ request live quota
→ continue to the next profile
→ restore the original profile
```

If language-server-only reload fails, AGSwitch falls back to a full application restart for that switch and for restoration of the original profile.

Only run `--confirm-idle` after the current response and tool calls have finished.

A successful renewal looks like:

```text
[SWITCH] work · restarting language server only
[LIVE] work renewed via Antigravity · 24 models · min 100%
[RESTORE] original-profile
```

A real quota result of `0%` remains `0%`; AGSwitch does not hide exhaustion.

## Switching

### Hot switch

Recommended while Antigravity is open and idle:

```bash
agswitch use work --hot-reload --confirm-idle
```

This writes and verifies the selected credential, replaces the language-server generation, and keeps the Electron UI open.

### Full restart

```bash
agswitch use work --restart
```

Use this to recover when the backend is not running or when a hot reload cannot replace the language server.

### Preserve current launch state

```bash
agswitch use work
```

### Do not start Antigravity

```bash
agswitch use work --no-start
```

## Quota commands

```bash
agswitch quota
agswitch quota --refresh
agswitch quota work --refresh
agswitch quota --json
```

Quota retrieval provides bounded concurrency, a private local cache, stale fallback for visibility only, model percentages, reset times, and explicit unknown values instead of fabricated data.

## Safe auto-switch

Preview only:

```bash
agswitch auto-switch --refresh --dry-run
```

Hot switch after confirming the current work is idle:

```bash
agswitch auto-switch \
  --refresh \
  --threshold 20 \
  --hot-reload \
  --confirm-idle
```

Only recent live Google Cloud Code snapshots without warnings are eligible. The current profile must be at or below the threshold, and another eligible profile must have a higher minimum known remaining percentage.

`cache-stale` and ambiguous provider overloads never authorize a switch.

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

Stable identity uses Google subject when available, then email, then a one-way identity derived from the refresh token. Raw identity material is not written to `accounts.json`.

## Themes

Dark:

```bash
AGSWITCH_THEME=dark agswitch
```

Light:

```bash
AGSWITCH_THEME=light agswitch
```

No color:

```bash
NO_COLOR=1 agswitch
```

## Configuration

Useful variables:

| Variable | Purpose |
| --- | --- |
| `AGSWITCH_APP_PATH` | Antigravity executable |
| `AGSWITCH_LANGUAGE_SERVER_PATH` | Backend executable used for hot reload |
| `AGSWITCH_QUIT_COMMAND` | Optional graceful quit command |
| `AGSWITCH_GRACEFUL_TIMEOUT` | Process shutdown timeout |
| `AGSWITCH_FORCE_KILL` | Disable forced cleanup when set to `false` |
| `AGSWITCH_OAUTH_CLIENT_ID` | Optional supported OAuth client ID |
| `AGSWITCH_OAUTH_CLIENT_SECRET` | Optional supported OAuth client secret |
| `AGSWITCH_THEME` | `dark` or `light` |
| `NO_COLOR` | Disable colors |

Do not extract, publish, or commit third-party OAuth client secrets. Prefer `refresh-via-antigravity` when the saved profile can be legitimately renewed by the installed Antigravity application.

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

Private local state:

```text
~/.config/agswitch/accounts.json
~/.local/state/agswitch/state.json
~/.local/state/agswitch/antigravity.log
~/.cache/agswitch/quota.json
```

Credential payloads remain in the operating-system keyring. Never commit access tokens, refresh tokens, cookies, or OAuth client secrets.

## Validation after upgrading

```bash
agswitch version
agswitch doctor
agswitch current --json
agswitch auth doctor
agswitch quota --refresh --json
agswitch auto-switch --refresh --dry-run --json
```

Then open:

```bash
agswitch
```

Verify that the dashboard reports the live-account count, current five-hour quota, reset countdown, and explicit authentication state for every saved profile.

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
cmd/                 CLI and dependency wiring
internal/app/        profile lifecycle and stable identity
internal/switcher/   transactional full/hot activation
internal/process/    backend reload, restart, and orphan cleanup
internal/quota/      provider, auth diagnostics, cache, reset windows
internal/autoswitch/ conservative live-quota selection
internal/monitor/    overload/quota incident classification
internal/tui/        responsive Bubble Tea + Lip Gloss dashboard
```

## Platform status

- Linux keyring, CLI, dashboard, quota, and switching: validated.
- Standalone Antigravity language-server hot reload: validated on Omarchy/Arch.
- Antigravity-assisted renewal: validated on multiple saved profiles; fallback recovery remains guarded by `--confirm-idle`.
- Antigravity IDE backend path: configurable, but should be tested per installed IDE version.
- Windows process integration: preview.
