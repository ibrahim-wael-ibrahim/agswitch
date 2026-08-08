#!/usr/bin/env bash

set -Eeuo pipefail

REPO="ibrahim-wael-ibrahim/agswitch"
VERSION="${AGSWITCH_VERSION:-latest}"
REF="${AGSWITCH_REF:-master}"
BUILD_FROM_SOURCE="${AGSWITCH_BUILD_FROM_SOURCE:-false}"
BINDIR="${BINDIR:-$HOME/.local/bin}"
SCRIPT_PATH="${BASH_SOURCE[0]-}"
ROOT=""
if [[ -n "$SCRIPT_PATH" && -f "$SCRIPT_PATH" ]]; then
  ROOT="$(cd -- "$(dirname -- "$SCRIPT_PATH")/.." 2>/dev/null && pwd || true)"
fi
TMP="$(mktemp -d 2>/dev/null || mktemp -d -t agswitch)"
SOURCE=""
trap 'rm -rf "$TMP"' EXIT

log() { printf '\033[1m[agswitch]\033[0m %s\n' "$*"; }
warn() { printf '\033[33m[agswitch] warning:\033[0m %s\n' "$*" >&2; }
die() { printf '\033[31m[agswitch] error:\033[0m %s\n' "$*" >&2; exit 1; }
has() { command -v "$1" >/dev/null 2>&1; }
truthy() { case "${1,,}" in 1|true|yes|on) return 0 ;; *) return 1 ;; esac; }

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
    winget install --id Git.Git -e --accept-package-agreements --accept-source-agreements || true
    winget install --id GoLang.Go -e --accept-package-agreements --accept-source-agreements || true
  elif has choco; then
    choco install -y git golang
  else
    warn "winget/Chocolatey not found. Install Git for Windows and Go manually."
  fi
}

release_url() {
  local name="agswitch_${OS}_${ARCH}"
  [[ "$OS" == windows ]] && name="${name}.exe"
  if [[ "$VERSION" == latest ]]; then
    printf 'https://github.com/%s/releases/latest/download/%s' "$REPO" "$name"
  else
    printf 'https://github.com/%s/releases/download/%s/%s' "$REPO" "$VERSION" "$name"
  fi
}

atomic_install() {
  local source="$1" name target pending backup
  name="$(basename "$source")"
  target="$BINDIR/$name"
  pending="$BINDIR/.${name}.new"
  backup="$BINDIR/.${name}.backup"
  mkdir -p "$BINDIR"
  install -m 0755 "$source" "$pending"
  rm -f "$backup"
  if [[ -f "$target" ]]; then
    mv "$target" "$backup"
  fi
  if ! mv "$pending" "$target"; then
    [[ -f "$backup" ]] && mv "$backup" "$target"
    die "could not replace $target"
  fi
  if ! "$target" version >/dev/null 2>&1; then
    warn "the new binary failed its version check; restoring the previous binary"
    rm -f "$target"
    [[ -f "$backup" ]] && mv "$backup" "$target"
    die "installed binary validation failed"
  fi
  rm -f "$backup"
}

install_release() {
  truthy "$BUILD_FROM_SOURCE" && return 1
  [[ "$REF" == master ]] || return 1
  has curl || return 1
  local url output
  url="$(release_url)"
  output="$TMP/agswitch"
  [[ "$OS" == windows ]] && output="$TMP/agswitch.exe"
  log "Downloading $url"
  curl -fL --retry 3 --connect-timeout 10 "$url" -o "$output" || return 1
  chmod +x "$output" || true
  atomic_install "$output"
  return 0
}

prepare_source() {
  if [[ -n "$ROOT" && -f "$ROOT/go.mod" ]]; then
    SOURCE="$ROOT"
    return 0
  fi
  has git || die "Git is required to clone the repository"
  SOURCE="$TMP/source"
  log "Cloning $REPO at $REF"
  git clone --depth 1 --branch "$REF" "https://github.com/$REPO.git" "$SOURCE"
}

source_version() {
  local tag sha release_version
  tag="$(cd "$SOURCE" && git describe --tags --exact-match 2>/dev/null || true)"
  if [[ -n "$tag" ]]; then
    printf '%s' "$tag"
    return 0
  fi
  if [[ "$REF" == master && -f "$SOURCE/VERSION" ]]; then
    release_version="$(tr -d '[:space:]' < "$SOURCE/VERSION")"
    if [[ -n "$release_version" ]]; then
      printf '%s' "$release_version"
      return 0
    fi
  fi
  sha="$(cd "$SOURCE" && git rev-parse --short=8 HEAD 2>/dev/null || true)"
  if [[ -n "$sha" ]]; then
    printf 'v0.0.0-dev+%s' "$sha"
  else
    printf 'dev'
  fi
}

build_from_source() {
  [[ "$OS" == linux ]] && install_linux_dependencies || install_windows_dependencies
  has go || die "Go is required but was not found after dependency installation"
  prepare_source
  local output="$TMP/agswitch" build_version
  [[ "$OS" == windows ]] && output="$TMP/agswitch.exe"
  build_version="$(source_version)"
  log "Building AGSwitch $build_version from source ref $REF"
  (cd "$SOURCE" && go build -trimpath -ldflags "-s -w -X github.com/ibrahim-wael/agswitch/cmd.version=$build_version" -o "$output" .)
  atomic_install "$output"
}

install_desktop_files() {
  [[ "$OS" == linux ]] || return 0
  [[ -n "$SOURCE" ]] || prepare_source
  local app_dir icon_dir desktop_target launcher_target desktop_exec omarchy_launcher
  app_dir="${XDG_DATA_HOME:-$HOME/.local/share}/applications"
  icon_dir="${XDG_DATA_HOME:-$HOME/.local/share}/icons/hicolor/scalable/apps"
  desktop_target="$app_dir/agswitch.desktop"
  launcher_target="$BINDIR/agswitch-desktop"
  [[ -f "$SOURCE/packaging/agswitch.desktop" ]] || return 0
  [[ -f "$SOURCE/assets/agswitch.svg" ]] || return 0
  [[ -f "$SOURCE/scripts/agswitch-desktop" ]] || return 0
  mkdir -p "$app_dir" "$icon_dir" "$BINDIR"
  install -m 0755 "$SOURCE/scripts/agswitch-desktop" "$launcher_target"
  desktop_exec="$launcher_target"
  omarchy_launcher="$(command -v omarchy-launch-or-focus-tui 2>/dev/null || true)"
  if [[ -n "$omarchy_launcher" ]]; then
    desktop_exec="$omarchy_launcher agswitch"
  fi
  sed "s|^Exec=.*|Exec=$desktop_exec|" "$SOURCE/packaging/agswitch.desktop" > "$desktop_target"
  install -m 0644 "$SOURCE/assets/agswitch.svg" "$icon_dir/agswitch.svg"
  has update-desktop-database && update-desktop-database "$app_dir" >/dev/null 2>&1 || true
  log "Installed desktop launcher: $desktop_target"
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
  local omarchy_launcher hypr_dir main_config bindings_config app_config source_line start_marker end_marker
  omarchy_launcher="$(command -v omarchy-launch-or-focus-tui 2>/dev/null || true)"
  [[ -n "$omarchy_launcher" ]] || return 0
  if ! has Hyprland && ! has hyprctl && [[ -z "${HYPRLAND_INSTANCE_SIGNATURE:-}" ]] && [[ ! -d "${XDG_CONFIG_HOME:-$HOME/.config}/hypr" ]]; then
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
windowrule = float on, center on, size 82% 82%, match:initial_class ^(org\.omarchy\.agswitch)$
EOF
  touch "$main_config" "$bindings_config"
  grep -Fqx "$source_line" "$main_config" || printf '\n# AGSwitch window rule\n%s\n' "$source_line" >> "$main_config"
  remove_managed_block "$bindings_config" "$start_marker" "$end_marker"
  cat >> "$bindings_config" <<EOF

$start_marker
bindd = SUPER SHIFT CTRL, A, AGSwitch, exec, $omarchy_launcher agswitch
$end_marker
EOF
  if has hyprctl && [[ -n "${HYPRLAND_INSTANCE_SIGNATURE:-}" ]]; then
    hyprctl reload >/dev/null 2>&1 || warn "Hyprland integration was installed but could not be reloaded automatically"
  fi
  log "Installed Omarchy shortcut: SUPER+SHIFT+CTRL+A"
}

if ! install_release; then
  build_from_source
fi
install_desktop_files
install_hyprland_integration

case ":$PATH:" in
  *":$BINDIR:"*) ;;
  *) warn "$BINDIR is not in PATH; add: export PATH=\"$BINDIR:\$PATH\"" ;;
esac

log "Installed $BINDIR/agswitch"
if [[ "$OS" == linux ]]; then
  "$BINDIR/agswitch" doctor || true
else
  warn "Windows process integration remains preview-only."
fi
