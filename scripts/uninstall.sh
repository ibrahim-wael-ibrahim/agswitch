#!/usr/bin/env bash
set -Eeuo pipefail

BINDIR="${BINDIR:-$HOME/.local/bin}"
DATA_HOME="${XDG_DATA_HOME:-$HOME/.local/share}"
CONFIG_HOME="${XDG_CONFIG_HOME:-$HOME/.config}"
BINARY="$BINDIR/agswitch"
DESKTOP_LAUNCHER="$BINDIR/agswitch-desktop"
DESKTOP_FILE="$DATA_HOME/applications/agswitch.desktop"
ICON_FILE="$DATA_HOME/icons/hicolor/scalable/apps/agswitch.svg"
HYPR_APP_CONFIG="$CONFIG_HOME/hypr/agswitch.conf"
HYPR_MAIN_CONFIG="$CONFIG_HOME/hypr/hyprland.conf"
HYPR_SOURCE_LINE="source = $HYPR_APP_CONFIG"

remove_file() {
  local path="$1"
  if [[ -e "$path" ]]; then
    rm -f -- "$path"
    printf 'Removed: %s\n' "$path"
  fi
}

remove_file "$BINARY"
remove_file "$DESKTOP_LAUNCHER"
remove_file "$DESKTOP_FILE"
remove_file "$ICON_FILE"
remove_file "$HYPR_APP_CONFIG"

if [[ -f "$HYPR_MAIN_CONFIG" ]] && grep -Fqx "$HYPR_SOURCE_LINE" "$HYPR_MAIN_CONFIG"; then
  tmp="$(mktemp)"
  grep -Fvx "$HYPR_SOURCE_LINE" "$HYPR_MAIN_CONFIG" | sed '/^# AGSwitch integration$/d' > "$tmp"
  mv "$tmp" "$HYPR_MAIN_CONFIG"
  printf 'Removed Hyprland source line: %s\n' "$HYPR_MAIN_CONFIG"
fi

if command -v update-desktop-database >/dev/null 2>&1; then
  update-desktop-database "$DATA_HOME/applications" >/dev/null 2>&1 || true
fi
if command -v hyprctl >/dev/null 2>&1 && [[ -n "${HYPRLAND_INSTANCE_SIGNATURE:-}" ]]; then
  hyprctl reload >/dev/null 2>&1 || true
fi

cat <<'EOF'
Credentials and metadata were kept.
Remove them manually only when you are sure:
  ~/.config/agswitch
  ~/.local/state/agswitch
  ~/.cache/agswitch
Saved profile secrets remain in the system keyring under service=agswitch.
EOF
