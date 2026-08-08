# AGSwitch

A secure account switcher and responsive terminal dashboard for Google Antigravity.

AGSwitch stores credentials in the operating-system keyring, tracks the active account across access-token rotation, displays model quota, and switches accounts transactionally with rollback.

**Author:** Ibrahim Wael  
**Repository:** `github.com/ibrahim-wael-ibrahim/agswitch`

## Highlights

- Secure saved profiles in Secret Service on Linux.
- Stable account identity that survives renewed access tokens and expiry changes.
- Automatic synchronization of a renewed active credential back to its saved profile.
- Transactional activation with verification and rollback.
- Language-server-only hot switching that keeps the Electron UI, open files, chat and terminals alive.
- Cleanup of orphan language servers after full application restarts.
- Safe quota-based recommendations: stale, warned, unknown or old quota never changes accounts.
- Conservative overload classification: temporary provider overload never authorizes an account switch.
- Bubble Tea v2 dashboard styled with Lip Gloss v2.
- Semantic color states, dark/light themes and `NO_COLOR` support.
- JSON output, diagnostics and an upgrade-aware installer.

## Installation and updates

Install or update from `master`:

```bash
curl -fsSL https://raw.githubusercontent.com/ibrahim-wael-ibrahim/agswitch/master/scripts/install.sh \
  | AGSWITCH_REF=master AGSWITCH_BUILD_FROM_SOURCE=true bash
```

The installer builds the selected ref, validates the new binary and atomically replaces the existing installation. The previous binary is restored if validation fails.

Useful installer variables:

| Variable | Purpose |
| --- | --- |
| `AGSWITCH_VERSION` | Install a specific tagged release |
| `AGSWITCH_REF` | Source branch or tag to build; default `master` |
| `AGSWITCH_BUILD_FROM_SOURCE` | Set `true` to bypass releases and build the selected ref |
| `BINDIR` | Binary destination; default `~/.local/bin` |

On Arch/Omarchy, the installer also maintains the desktop launcher, floating Hyprland rule and `SUPER+SHIFT+CTRL+A` shortcut.

Verify after updating:

```bash
agswitch version
agswitch config
agswitch doctor
agswitch current --json
```

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

Open the dashboard with:

```bash
agswitch
```

or:

```bash
agswitch tui
```

The UI uses Bubble Tea for state and events and Lip Gloss for responsive layout, borders and semantic colors.

### Dashboard flow

The normal workflow is intentionally short:

```text
select account
→ press s for Hot switch
→ confirm Antigravity is idle
→ language server restarts
→ Electron UI stays open
```

The dashboard separates quota freshness from quota percentage. Old cached values remain visible for diagnostics, but they are clearly marked and cannot drive auto-switching.

Account badges:

| Badge | Meaning |
| --- | --- |
| `LIVE` | Recent Google Cloud Code quota; eligible for recommendations |
| `STALE` | Fallback cache or provider warning; display-only |
| `OLD` | Previously live snapshot older than the safe age limit; display-only |
| `ERROR` | Quota fetch failed |
| `UNKNOWN` | No trustworthy quota value is available |

Color semantics:

| Color | Meaning |
| --- | --- |
| Green | Healthy/live state or high remaining quota |
| Amber | Warning, stale data or medium remaining quota |
| Red | Error or low remaining quota |
| Purple | Navigation, focus and selected actions |
| Muted gray | Secondary or display-only information |

Quota bars use the same semantics: `<=20%` red, `21–50%` amber and `>50%` green.

### Keyboard shortcuts

| Key | Action |
| --- | --- |
| `Tab`, `Shift-Tab`, arrows, `h/j/k/l` | Navigate panels and rows |
| `s` | Hot switch selected account |
| `Enter` | Select or run focused action |
| `/` | Search accounts |
| `r` | Refresh quota now |
| `a` | Preview/confirm auto hot-switch recommendation |
| `A` | Edit auto-switch threshold |
| `R` | Edit automatic refresh interval |
| `p` | Previous account |
| `d` | Run doctor |
| `q`, `Ctrl-C` | Quit |

Hot switch and auto hot-switch always show a safety confirmation. Continue only after the current response and tool calls have finished.

### Themes and colors

The default theme is optimized for dark terminals.

Force dark mode:

```bash
AGSWITCH_THEME=dark agswitch
```

Force light mode:

```bash
AGSWITCH_THEME=light agswitch
```

Disable colors while keeping the same layout:

```bash
NO_COLOR=1 agswitch
```

Lip Gloss automatically reduces colors when the terminal supports a smaller color profile.

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

### Hot switch — recommended while Antigravity is open

Wait until the current response and all tool calls have finished, then:

```bash
agswitch use work --hot-reload --confirm-idle
```

The transaction is:

```text
acquire operation lock
→ back up active credential
→ write and verify selected credential
→ stop old language-server generation
→ wait for a new language-server PID
→ commit current/previous account state
```

If backend replacement fails, AGSwitch restores the previous credential and reloads the backend again.

The same path is available directly from the dashboard with `s`.

### Full restart

```bash
agswitch use work --restart
```

AGSwitch closes Antigravity, removes related stale language-server processes, changes the credential and starts Antigravity again.

### Preserve launch state

```bash
agswitch use work
```

### Switch without starting Antigravity

```bash
agswitch use work --no-start
```

To target Antigravity IDE instead of standalone Antigravity, configure both paths before running AGSwitch:

```bash
export AGSWITCH_APP_PATH=/opt/antigravity-ide/antigravity-ide
export AGSWITCH_LANGUAGE_SERVER_PATH=/opt/antigravity-ide/resources/app/extensions/antigravity/bin/language_server_linux_x64
```

The standalone backend path has been validated on Omarchy/Arch. IDE hot reload should be tested against each installed IDE version because it can run multiple language-server processes.

## Stable account detection

Antigravity renews access tokens and expiry fields. A raw hash of the complete credential therefore changes even when the Google account is unchanged.

AGSwitch separates:

- **Payload fingerprint:** verifies exact keyring writes and detects credential JSON changes.
- **Identity fingerprint:** identifies the account using Google subject, then email, then a one-way value derived from the refresh token when those claims are absent.

Raw identity material is never written to `accounts.json`. When the active payload rotates, `agswitch current` matches the stable identity and synchronizes the renewed credential into the saved profile.

Existing metadata upgrades lazily; no manual migration is required.

## Quota

```bash
agswitch quota
agswitch quota --refresh
agswitch quota work
agswitch quota --json
```

Quota retrieval provides:

- bounded concurrent requests;
- private local caching;
- stale-cache fallback for visibility only;
- model percentage, exhaustion state and reset time when returned by Google;
- `unknown` rather than an invented percentage when quota cannot be trusted.

Google Cloud Code endpoints used by Antigravity are internal and may change.

### Authentication limitation

Observed Antigravity credentials include access and refresh tokens but do not include the OAuth client credentials required to independently refresh every inactive profile.

Compatible client values may be supplied explicitly when obtained through a supported authentication flow:

```bash
export AGSWITCH_OAUTH_CLIENT_ID=...
export AGSWITCH_OAUTH_CLIENT_SECRET=...
```

Do not extract or publish secrets embedded in third-party binaries. Until a supported inactive-profile flow is available, activate an account through Antigravity so it renews the credential, then let AGSwitch synchronize the renewed payload.

## Safe auto-switch

Preview:

```bash
agswitch auto-switch --refresh --dry-run
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

The candidate score is the lowest known remaining percentage across the account's models. AGSwitch switches only when the current account is at or below the configured threshold and another eligible account has a higher score.

## Error classification

The monitor classifier groups adjacent request logs conservatively:

- `server_overloaded`: temporary provider overload; retry/backoff only, never switch account.
- `resource_exhausted_ambiguous`: insufficient evidence; never switch account.
- explicit quota exhaustion classes can inform future automation only when quota evidence is specific enough.

The project does not intercept HTTPS requests, replace bearer tokens mid-request, or replay a failed request under another account.

An unattended daemon remains intentionally disabled until a reliable whole-task idle signal and supported live quota for inactive profiles are available.

## Process safety

A full restart verifies that related old language-server processes are gone before considering the transaction complete.

Hot reload stops the old backend generation and succeeds only after the old PIDs disappear and a replacement language-server PID appears.

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
| `AGSWITCH_THEME` | `dark` or `light` dashboard palette |
| `NO_COLOR` | Disable dashboard colors when set |

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

Then, while no task is running, open the dashboard:

```bash
agswitch
```

Select a real profile and press `s`. Confirm that:

- `agswitch current` reports the target profile;
- the Electron window, open files, chat and terminals remain;
- a simple new request succeeds;
- the old language-server PID is gone;
- a new language-server PID exists;
- stale quota is visibly marked and never recommended.

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
internal/tui/        Bubble Tea + Lip Gloss dashboard
```

## Platform status

- Linux keyring, CLI, dashboard and full switching: validated.
- Standalone language-server-only restart: validated on Omarchy/Arch.
- Antigravity IDE backend path: configurable, but hot reload requires controlled testing per installed IDE version.
- Windows process integration: preview.
