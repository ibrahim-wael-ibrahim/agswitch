#!/usr/bin/env bash
set -Eeuo pipefail
if command -v agswitch >/dev/null 2>&1; then exec agswitch migrate "$@"; fi
ROOT="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
exec go run "$ROOT" migrate "$@"
