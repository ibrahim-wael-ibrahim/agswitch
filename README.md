# agswitch

Linux CLI and terminal dashboard for switching Antigravity accounts safely and viewing per-account model quota.

## Highlights

- Profile credentials are stored in Secret Service, never plaintext metadata files.
- The active account is detected using a SHA-256 credential fingerprint.
- Switching is transactional: lock, backup, stop, store, verify, launch, commit, rollback on failure.
- Antigravity launch state is preserved by default, with `--restart` and `--no-start` overrides.
- Legacy JSON profiles can be migrated and verified before optional deletion.
- The default `fzf` dashboard clears the terminal, loads quota for every account, shows colored progress bars, and switches on Enter.
- Quota requests use a bounded concurrent worker pool, five-minute private cache, stale-cache fallback, timeout, and token redaction.
- Diagnostics cover keyring dependencies, paths, permissions, D-Bus, tray availability, quota cache, logs, and process state.
- Tagged releases build Linux amd64 and arm64 binaries with SHA-256 checksums.

## Requirements

- Linux with Secret Service and `secret-tool`
- `pgrep`
- `fzf` for the interactive dashboard
- `busctl` is optional but recommended for tray diagnostics

## Build and install

```bash
make check
make install
hash -r
agswitch doctor
```

The binary is installed to `~/.local/bin/agswitch` by default. Override with `BINDIR=/custom/path make install`.

## First migration

```bash
agswitch migrate
agswitch list
agswitch current
```

Only after testing every migrated profile:

```bash
agswitch migrate --force --delete-source
```

## Dashboard

```bash
agswitch
# or
agswitch tui
```

Keys:

- `Enter`: switch to the selected account and launch Antigravity
- `Ctrl-R`: bypass the quota cache and refresh live data
- `Esc`: exit

Use `agswitch tui --stay` to keep reopening the dashboard after a successful switch.

## Account commands

```bash
agswitch save work
agswitch update work
agswitch clone work work-backup
agswitch rename work company
agswitch info company
agswitch list
agswitch current
agswitch detect
agswitch use company
agswitch use company --restart
agswitch use company --no-start
agswitch previous
agswitch status
agswitch status --json
agswitch delete company
```

## Quota

```bash
agswitch quota
agswitch quota --refresh
agswitch quota work
agswitch quota --json
```

Quota is read from Google Cloud Code internal endpoints. These are not a public stable API and may change without notice. `agswitch` isolates the implementation behind a provider, never stores tokens in the cache, and shows unavailable/stale status instead of inventing `0%` data.

When an expired access token must be refreshed and the saved credential does not contain OAuth client information, configure:

```bash
export AGSWITCH_OAUTH_CLIENT_ID='...'
export AGSWITCH_OAUTH_CLIENT_SECRET='...'
```

Never place tokens or client secrets in the repository.

## Storage

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

Non-secret files:

```text
~/.config/agswitch/accounts.json
~/.local/state/agswitch/state.json
~/.local/state/agswitch/antigravity.log
~/.cache/agswitch/quota/
```

## Process shutdown

Set a tested custom command when your desktop requires the tray `Quit` action:

```bash
export AGSWITCH_QUIT_COMMAND="$HOME/.config/agswitch/quit-antigravity"
```

The fallback targets the detected main process with SIGTERM, waits for shutdown, then optionally uses SIGKILL. Generic tray services are diagnosed, but the exact Antigravity D-Bus Quit action must be verified on the target desktop session.

## Development

```bash
make fmt
make tidy
make vet
make test
make race
make build
make release VERSION=v0.1.0
```

## Release

Push a semantic version tag:

```bash
git tag v0.1.0
git push origin v0.1.0
```

GitHub Actions publishes Linux amd64/arm64 binaries and checksums.

## Uninstall

```bash
make uninstall
# or
./scripts/uninstall.sh
```
