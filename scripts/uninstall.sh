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
HYPR_BINDINGS_CONFIG="$CONFIG_HOME/hypr/bindings.conf"
HYPR_SOURCE_LINE="source = $HYPR_APP_CONFIG"
HYPR_BIND_START="# BEGIN AGSwitch managed binding"
HYPR_BIND_END="# END AGSwitch managed binding"

remove_file() {
  local path="$1"
  if [[ -e "$path" ]]; then
    rm -f -- "$path"
    printf 'Removed: %s\n' "$path"
  fi
}

remove_managed_block() {
  local file="$1" start_marker="$2" end_marker="$3" output
  [[ -f "$file" ]] || return 0
  output="$(mktemp)"
  awk -v start="$start_marker" -v end="$end_marker" '
    $0 == start { skipping = 1; next }
    $0 == end { skipping = 0; next }
    !skipping { print }
  ' "$file" > "$output"
  mv "$output" "$file"
}

remove_file "$BINARY"
remove_file "$DESKTOP_LAUNCHER"
remove_file "$DESKTOP_FILE"
remove_file "$ICON_FILE"
remove_file "$HYPR_APP_CONFIG"

if [[ -f "$HYPR_MAIN_CONFIG" ]] && grep -Fqx "$HYPR_SOURCE_LINE" "$HYPR_MAIN_CONFIG"; then
  tmp="$(mktemp)"
  grep -Fvx "$HYPR_SOURCE_LINE" "$HYPR_MAIN_CONFIG" |
    sed '/^# AGSwitch integration$/d; /^# AGSwitch window rule$/d' > "$tmp"
  mv "$tmp" "$HYPR_MAIN_CONFIG"
  printf 'Removed Hyprland source line: %s\n' "$HYPR_MAIN_CONFIG"
fi

remove_managed_block "$HYPR_BINDINGS_CONFIG" "$HYPR_BIND_START" "$HYPR_BIND_END"

# Remove the legacy unmarked AGSwitch bind written by older installers.
if [[ -f "$HYPR_BINDINGS_CONFIG" ]]; then
  tmp="$(mktemp)"
  grep -Fv 'AGSwitch, exec,' "$HYPR_BINDINGS_CONFIG" > "$tmp" || true
  mv "$tmp" "$HYPR_BINDINGS_CONFIG"
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
