#!/usr/bin/env bash
set -Eeuo pipefail
ROOT="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
BINDIR="${BINDIR:-$HOME/.local/bin}"
go build -trimpath -o "$ROOT/agswitch" "$ROOT"
install -d -m 0755 "$BINDIR"
install -m 0755 "$ROOT/agswitch" "$BINDIR/agswitch"
rm -f "$ROOT/agswitch"
printf 'Installed: %s/agswitch\n' "$BINDIR"
