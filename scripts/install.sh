#!/usr/bin/env bash

set -Eeuo pipefail

ROOT="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
BINDIR="${BINDIR:-$HOME/.local/bin}"
OUTPUT="$ROOT/bin/agswitch"

mkdir -p "$ROOT/bin"
go build -trimpath -o "$OUTPUT" "$ROOT"
install -d -m 0755 "$BINDIR"
install -m 0755 "$OUTPUT" "$BINDIR/agswitch"
printf 'Installed: %s/agswitch\n' "$BINDIR"
