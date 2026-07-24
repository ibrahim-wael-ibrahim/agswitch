#!/usr/bin/env bash

set -Eeuo pipefail

REPO="ibrahim-wael-ibrahim/agswitch"
VERSION="${AGSWITCH_VERSION:-latest}"
BINDIR="${BINDIR:-$HOME/.local/bin}"
ROOT="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." 2>/dev/null && pwd || true)"
TMP="$(mktemp -d 2>/dev/null || mktemp -d -t agswitch)"
trap 'rm -rf "$TMP"' EXIT

log() { printf '\033[1m[agswitch]\033[0m %s\n' "$*"; }
warn() { printf '\033[33m[agswitch] warning:\033[0m %s\n' "$*" >&2; }
die() { printf '\033[31m[agswitch] error:\033[0m %s\n' "$*" >&2; exit 1; }
has() { command -v "$1" >/dev/null 2>&1; }

OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"
case "$OS" in
  linux*) OS=linux ;;
  mingw*|msys*|cygwin*) OS=windows ;;
  *) die "unsupported operating system: $OS" ;;
esac
case "$ARCH" in
  x86_64|amd64) ARCH=amd64 ;;
  aarch64|arm64) ARCH=arm64 ;;
  *) die "unsupported architecture: $ARCH" ;;
esac

install_linux_dependencies() {
  if has apt-get; then
    log "Installing Debian/Ubuntu dependencies"
    sudo apt-get update
    sudo apt-get install -y ca-certificates curl git golang-go libsecret-tools procps systemd
  elif has dnf; then
    log "Installing Fedora dependencies"
    sudo dnf install -y ca-certificates curl git golang libsecret procps-ng systemd
  elif has pacman; then
    log "Installing Arch dependencies"
    sudo pacman -Sy --needed --noconfirm ca-certificates curl git go libsecret procps-ng systemd
  else
    warn "Unknown package manager. Install curl, git, Go, libsecret/secret-tool and procps manually."
  fi
}

install_windows_dependencies() {
  if has winget; then
    log "Installing Windows build dependencies with winget"
    winget install --id Git.Git -e --accept-package-agreements --accept-source-agreements || true
    winget install --id GoLang.Go -e --accept-package-agreements --accept-source-agreements || true
  elif has choco; then
    log "Installing Windows build dependencies with Chocolatey"
    choco install -y git golang
  else
    warn "winget/Chocolatey not found. Install Git for Windows and Go manually."
  fi
}

release_url() {
  local name="agswitch_${OS}_${ARCH}"
  if [[ "$OS" == windows ]]; then name="${name}.exe"; fi
  if [[ "$VERSION" == latest ]]; then
    printf 'https://github.com/%s/releases/latest/download/%s' "$REPO" "$name"
  else
    printf 'https://github.com/%s/releases/download/%s/%s' "$REPO" "$VERSION" "$name"
  fi
}

install_release() {
  has curl || return 1
  local url output
  url="$(release_url)"
  output="$TMP/agswitch"
  [[ "$OS" == windows ]] && output="$TMP/agswitch.exe"
  log "Downloading $url"
  curl -fL --retry 3 --connect-timeout 10 "$url" -o "$output" || return 1
  chmod +x "$output" || true
  mkdir -p "$BINDIR"
  cp "$output" "$BINDIR/$(basename "$output")"
  return 0
}

build_from_source() {
  [[ "$OS" == linux ]] && install_linux_dependencies || install_windows_dependencies
  has go || die "Go is required but was not found after dependency installation"
  local source="$ROOT"
  if [[ ! -f "$source/go.mod" ]]; then
    has git || die "Git is required to clone the repository"
    source="$TMP/source"
    git clone --depth 1 "https://github.com/$REPO.git" "$source"
  fi
  log "Building agswitch from source"
  mkdir -p "$BINDIR"
  local output="$BINDIR/agswitch"
  [[ "$OS" == windows ]] && output="$BINDIR/agswitch.exe"
  (cd "$source" && go build -trimpath -ldflags "-s -w" -o "$output" .)
}

fetch_asset() {
  local relative="$1" output="$2"
  if [[ -f "$ROOT/$relative" ]]; then
    cp "$ROOT/$relative" "$output"
    return 0
  fi
  has curl || return 1
  curl -fsSL --retry 3 "https://raw.githubusercontent.com/$REPO/master/$relative" -o "$output"
}

omarchy_tui_launcher() {
  command -v omarchy-launch-or-focus-tui 2>/dev/null || true
}

install_desktop_files() {
  [[ "$OS" == linux ]] || return 0

  local app_dir icon_dir desktop_source icon_source launcher_source desktop_target launcher_target desktop_exec omarchy_launcher
  app_dir="${XDG_DATA_HOME:-$HOME/.local/share}/applications"
  icon_dir="${XDG_DATA_HOME:-$HOME/.local/share}/icons/hicolor/scalable/apps"
  desktop_source="$TMP/agswitch.desktop"
  icon_source="$TMP/agswitch.svg"
  launcher_source="$TMP/agswitch-desktop"
  desktop_target="$app_dir/agswitch.desktop"
  launcher_target="$BINDIR/agswitch-desktop"

  if ! fetch_asset "packaging/agswitch.desktop" "$desktop_source"; then
    warn "Could not download the desktop launcher"
    return 0
  fi
  if ! fetch_asset "assets/agswitch.svg" "$icon_source"; then
    warn "Could not download the desktop icon"
    return 0
  fi
  if ! fetch_asset "scripts/agswitch-desktop" "$launcher_source"; then
    warn "Could not download the terminal launcher"
    return 0
  fi

  mkdir -p "$app_dir" "$icon_dir" "$BINDIR"
  install -m 0755 "$launcher_source" "$launcher_target"

  desktop_exec="$launcher_target"
  omarchy_launcher="$(omarchy_tui_launcher)"
  if [[ -n "$omarchy_launcher" ]]; then
    desktop_exec="$omarchy_launcher agswitch"
  fi

  sed "s|^Exec=.*|Exec=$desktop_exec|" "$desktop_source" > "$desktop_target"
  install -m 0644 "$icon_source" "$icon_dir/agswitch.svg"
  chmod 0644 "$desktop_target"

  if has update-desktop-database; then
    update-desktop-database "$app_dir" >/dev/null 2>&1 || true
  fi
  if has gtk-update-icon-cache; then
    gtk-update-icon-cache -f -t "${XDG_DATA_HOME:-$HOME/.local/share}/icons/hicolor" >/dev/null 2>&1 || true
  fi
  log "Installed desktop launcher: $desktop_target"
}

hyprland_installed() {
  has Hyprland || has hyprctl || [[ -n "${HYPRLAND_INSTANCE_SIGNATURE:-}" ]] || [[ -d "${XDG_CONFIG_HOME:-$HOME/.config}/hypr" ]]
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

install_hyprland_integration() {
  [[ "$OS" == linux ]] || return 0
  hyprland_installed || return 0

  local omarchy_launcher hypr_dir main_config bindings_config app_config source_line start_marker end_marker
  omarchy_launcher="$(omarchy_tui_launcher)"
  if [[ -z "$omarchy_launcher" ]]; then
    warn "Hyprland was found, but omarchy-launch-or-focus-tui is unavailable; skipped the Omarchy shortcut"
    return 0
  fi

  hypr_dir="${XDG_CONFIG_HOME:-$HOME/.config}/hypr"
  main_config="$hypr_dir/hyprland.conf"
  bindings_config="$hypr_dir/bindings.conf"
  app_config="$hypr_dir/agswitch.conf"
  source_line="source = $app_config"
  start_marker="# BEGIN AGSwitch managed binding"
  end_marker="# END AGSwitch managed binding"

  mkdir -p "$hypr_dir"
  cat > "$app_config" <<'EOF'
# Managed by AGSwitch installer.
# Omarchy creates this TUI with initial class org.omarchy.agswitch.
windowrule = float on, center on, size 82% 82%, match:initial_class ^(org\.omarchy\.agswitch)$
EOF

  touch "$main_config" "$bindings_config"
  if ! grep -Fqx "$source_line" "$main_config"; then
    printf '\n# AGSwitch window rule\n%s\n' "$source_line" >> "$main_config"
  fi

  remove_managed_block "$bindings_config" "$start_marker" "$end_marker"
  cat >> "$bindings_config" <<EOF

$start_marker
bindd = SUPER SHIFT CTRL, A, AGSwitch, exec, $omarchy_launcher agswitch
$end_marker
EOF

  if has hyprctl && [[ -n "${HYPRLAND_INSTANCE_SIGNATURE:-}" ]]; then
    hyprctl reload >/dev/null 2>&1 || warn "Hyprland integration was installed but could not be reloaded automatically"
    if config_errors="$(hyprctl configerrors 2>/dev/null)" && [[ -n "${config_errors//[[:space:]]/}" ]]; then
      warn "Hyprland reports configuration errors after reload:"
      printf '%s\n' "$config_errors" >&2
    fi
  fi

  log "Installed Omarchy shortcut: SUPER+SHIFT+CTRL+A"
  log "Installed Hyprland window rule: $app_config"
  log "Installed Hyprland binding: $bindings_config"
}

if ! install_release; then
  warn "No compatible release binary was available; falling back to source build."
  build_from_source
fi

install_desktop_files
install_hyprland_integration

case ":$PATH:" in
  *":$BINDIR:"*) ;;
  *)
    warn "$BINDIR is not in PATH"
    printf 'Add this line to your shell profile:\n  export PATH="%s:$PATH"\n' "$BINDIR"
    ;;
esac

log "Installed $(ls "$BINDIR"/agswitch* 2>/dev/null | head -n1)"
if [[ "$OS" == linux ]]; then
  "$BINDIR/agswitch" doctor || true
else
  warn "Windows support requires an Antigravity Windows installation and uses Windows Credential Manager."
fi
