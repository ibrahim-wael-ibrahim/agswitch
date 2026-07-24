# agswitch

Linux CLI and TUI for switching Antigravity accounts safely.

## Features

- Profile credentials stored in Secret Service, not plaintext files.
- Active account detection by SHA-256 credential fingerprint.
- Transactional switch with verification and rollback.
- Preserves whether Antigravity was running; supports `--restart` and `--no-start`.
- Legacy JSON migration with verification before optional deletion.
- TUI account picker that exits after successful launch.
- `doctor` diagnostics for keyring, process, D-Bus, tray services, paths, and permissions.

Live quota is intentionally deferred because the known Google endpoints are internal and unstable.

## Install

```bash
make test
make install
agswitch doctor
```

## Usage

```bash
agswitch                         # open TUI
agswitch tui --stay
agswitch save work
agswitch list
agswitch current
agswitch use work
agswitch use work --restart
agswitch use work --no-start
agswitch delete work
agswitch migrate
agswitch migrate --force --delete-source
agswitch doctor
agswitch config
```

The default active keyring entry is `service=gemini`, `username=antigravity`. Saved profiles use `service=agswitch`, `username=profile:<name>`.

Metadata is stored at `~/.config/agswitch/accounts.json`; runtime state and logs are under `~/.local/state/agswitch/`.

For tray-only graceful exit, set a tested helper command:

```bash
export AGSWITCH_QUIT_COMMAND="$HOME/.config/agswitch/quit-antigravity"
```

The fallback sends SIGTERM to the oldest matching main process, waits, then optionally SIGKILLs remaining matching processes.
