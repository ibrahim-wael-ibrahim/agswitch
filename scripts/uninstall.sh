#!/usr/bin/env bash
set -Eeuo pipefail

BINDIR="${BINDIR:-$HOME/.local/bin}"
BINARY="$BINDIR/agswitch"

if [[ -e "$BINARY" ]]; then
  rm -f -- "$BINARY"
  printf 'Removed: %s\n' "$BINARY"
else
  printf 'Not installed: %s\n' "$BINARY"
fi

cat <<'EOF'
Credentials and metadata were kept.
Remove them manually only when you are sure:
  ~/.config/agswitch
  ~/.local/state/agswitch
  ~/.cache/agswitch
Saved profile secrets remain in the system keyring under service=agswitch.
EOF
