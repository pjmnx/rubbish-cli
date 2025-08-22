#!/usr/bin/env bash
set -euo pipefail

# rubbish installer
# - Detect OS/ARCH
# - Download matching release tarball from GitHub
# - Install binary to /usr/local/bin (or custom prefix)
# - Install sample config to /etc/rubbish/config.cfg (or user path)
# - Clean up temp files

REPO_OWNER="pjmnx"
REPO_NAME="rubbish-cli"
APP_NAME="rubbish"
PREFIX="/usr/local"
BIN_DIR="$PREFIX/bin"
ETC_DIR="/etc/rubbish"
CONFIG_TARGET="$ETC_DIR/config.cfg"
CHANNEL="stable"   # stable|pre
VERSION=""         # empty -> latest

usage() {
  cat <<EOF
Usage: curl -fsSL https://raw.githubusercontent.com/${REPO_OWNER}/${REPO_NAME}/main/scripts/install.sh | bash -s -- [options]

Options:
  --version <tag>     Install specific tag (e.g., v1.2.3). Defaults to latest ${CHANNEL}.
  --pre               Use latest pre-release (alpha/beta) instead of stable.
  --prefix <dir>      Install prefix (default: ${PREFIX}). Binary goes to <prefix>/bin.
  --bin-dir <dir>     Install binary to directory (overrides --prefix path).
  --etc-dir <dir>     Config directory (default: ${ETC_DIR}).
  --user-config       Install config to user home (~/.config/rubbish.cfg) instead of /etc.
  --no-alias          Do not add shell alias: toss='rubbish toss'.
  --no-sudo           Do not use sudo (assumes you have permissions).
  --dry-run           Print actions without executing.
  -h, --help          Show this help.
EOF
}

require() {
  command -v "$1" >/dev/null 2>&1 || { echo "Error: missing required command '$1'" >&2; exit 1; }
}

echo_step() { echo "==> $*"; }

echo_err() { echo "[error] $*" >&2; }

dryrun=false
use_sudo=true
user_config=false
add_alias=true

# Parse args
while [[ $# -gt 0 ]]; do
  case "$1" in
    --version) VERSION="$2"; shift 2 ;;
    --pre) CHANNEL="pre"; shift ;;
    --prefix) PREFIX="$2"; BIN_DIR="$PREFIX/bin"; shift 2 ;;
    --bin-dir) BIN_DIR="$2"; shift 2 ;;
    --etc-dir) ETC_DIR="$2"; CONFIG_TARGET="$ETC_DIR/config.cfg"; shift 2 ;;
    --user-config) user_config=true; shift ;;
  --no-alias) add_alias=false; shift ;;
    --no-sudo) use_sudo=false; shift ;;
    --dry-run) dryrun=true; shift ;;
    -h|--help) usage; exit 0 ;;
    *) echo_err "Unknown option: $1"; usage; exit 1 ;;
  esac
done

SUDO=""
if $use_sudo; then
  if [[ $EUID -ne 0 ]]; then SUDO="sudo"; fi
fi

# Requirements
require uname
require mktemp
require tar
require curl

# Add shell alias: toss => 'rubbish toss'
add_shell_alias() {
  local user_name user_home shell_name rc_file alias_line
  user_name="${SUDO_USER:-$USER}"
  # Resolve the home of the invoking user (not root when using sudo)
  user_home=$(eval echo "~$user_name")
  shell_name="${SHELL##*/}"
  case "$shell_name" in
    zsh) rc_file="$user_home/.zshrc" ;;
    bash)
      rc_file="$user_home/.bashrc"
      if [[ "$(uname -s)" == "Darwin" ]] && [[ -f "$user_home/.bash_profile" ]]; then
        rc_file="$user_home/.bash_profile"
      fi
      ;;
    *) rc_file="$user_home/.profile" ;;
  esac
  alias_line="alias toss='rubbish toss'"
  if $dryrun; then
    echo "Would append alias to $rc_file: $alias_line"
    return 0
  fi
  mkdir -p "$(dirname "$rc_file")"
  if [[ -f "$rc_file" ]] && grep -Fq "$alias_line" "$rc_file"; then
    echo_step "Shell alias already present in $rc_file"
  else
    {
      echo
      echo "# Added by rubbish installer on $(date -u +'%Y-%m-%dT%H:%M:%SZ')"
      echo "$alias_line"
    } >> "$rc_file"
    echo_step "Added shell alias to $rc_file. Reload your shell or run: source \"$rc_file\""
  fi
}

# Detect OS/ARCH mapping to Go tuples
OS="$(uname -s)"
ARCH="$(uname -m)"
case "$OS" in
  Linux)  GOOS="linux" ;;
  Darwin) GOOS="darwin" ;;
  *) echo_err "Unsupported OS: $OS"; exit 1 ;;
esac
case "$ARCH" in
  x86_64|amd64) GOARCH="amd64" ;;
  aarch64|arm64) GOARCH="arm64" ;;
  i386|i686) GOARCH="386" ;;
  *) echo_err "Unsupported ARCH: $ARCH"; exit 1 ;;
esac

# Resolve version and asset base name
select_latest() {
  local url="https://api.github.com/repos/${REPO_OWNER}/${REPO_NAME}/releases"
  if [[ "$CHANNEL" == "pre" ]]; then
    curl -fsSL "$url" | awk '/"tag_name":/ {print $2}' | tr -d '"',',' | head -n1
  else
    curl -fsSL "$url/latest" | awk -F '"' '/tag_name/ {print $4}'
  fi
}

if [[ -z "$VERSION" ]]; then
  echo_step "Resolving latest ${CHANNEL} version"
  VERSION="$(select_latest)"
  if [[ -z "$VERSION" ]]; then echo_err "Could not determine latest release"; exit 1; fi
fi

BASE="${APP_NAME}_${VERSION}_${GOOS}_${GOARCH}"
TARBALL="${BASE}.tar.gz"
DL_URL="https://github.com/${REPO_OWNER}/${REPO_NAME}/releases/download/${VERSION}/${TARBALL}"
SAMPLE_URL="https://github.com/${REPO_OWNER}/${REPO_NAME}/releases/download/${VERSION}/sample.rubbish.cfg"

# Prepare temp dir
WORKDIR="$(mktemp -d)"
trap 'rm -rf "$WORKDIR"' EXIT
cd "$WORKDIR"

echo_step "Downloading ${TARBALL}"
if $dryrun; then echo "curl -fL -o ${TARBALL} ${DL_URL}"; else curl -fL -o "$TARBALL" "$DL_URL"; fi

echo_step "Extracting archive"
if $dryrun; then echo "tar -xzf ${TARBALL}"; else tar -xzf "$TARBALL"; fi

# Find extracted folder (could be current dir contents)
if [[ -d "${BASE}" ]]; then SRC_DIR="${BASE}"; else SRC_DIR="."; fi

# Install binary
mkdir_cmd="$SUDO mkdir -p \"$BIN_DIR\""
install_cmd="$SUDO install -m 0755 \"$SRC_DIR/${APP_NAME}\" \"$BIN_DIR/${APP_NAME}\""

echo_step "Installing binary to ${BIN_DIR}"
if $dryrun; then echo "$mkdir_cmd"; echo "$install_cmd"; else eval "$mkdir_cmd"; eval "$install_cmd"; fi

# Install config
if $user_config; then
  CONFIG_TARGET="$HOME/.config/rubbish.cfg"
  CONFIG_DIR="$(dirname "$CONFIG_TARGET")"
  mkcfg_cmd="mkdir -p \"$CONFIG_DIR\""
  copycfg_cmd="cp -n \"$SRC_DIR/sample.rubbish.cfg\" \"$CONFIG_TARGET\" || true"
else
  mkcfg_cmd="$SUDO mkdir -p \"$ETC_DIR\""
  copycfg_cmd="$SUDO cp -n \"$SRC_DIR/sample.rubbish.cfg\" \"$CONFIG_TARGET\" || true"
fi

echo_step "Installing sample config to ${CONFIG_TARGET} (won't overwrite existing)"
# Prefer the separately attached asset if available (skip network in dry-run)
if ! $dryrun && curl -fsI "$SAMPLE_URL" >/dev/null 2>&1; then
  echo_step "Downloading sample.rubbish.cfg from release"
  if $dryrun; then echo "curl -fL -o sample.rubbish.cfg ${SAMPLE_URL}"; else curl -fL -o sample.rubbish.cfg "$SAMPLE_URL"; fi
  SRC_CFG="sample.rubbish.cfg"
else
  SRC_CFG="$SRC_DIR/sample.rubbish.cfg"
fi

if $dryrun; then echo "$mkcfg_cmd"; echo "cp -n \"$SRC_CFG\" \"$CONFIG_TARGET\" || true"; else
  eval "$mkcfg_cmd"
  # shellcheck disable=SC2086
  if [[ -f "$SRC_CFG" ]]; then
    if $user_config; then
      cp -n "$SRC_CFG" "$CONFIG_TARGET" || true
    else
      $SUDO cp -n "$SRC_CFG" "$CONFIG_TARGET" || true
    fi
  else
    if $user_config; then
      echo "[DEFAULT]
wipeout_time = 30
container_path = ~/.local/share/rubbish
[notifications]
enabled = false
days_in_advance = 3
timeout = 5" | tee "$CONFIG_TARGET" >/dev/null
    else
      echo "[DEFAULT]
wipeout_time = 30
container_path = ~/.local/share/rubbish
[notifications]
enabled = false
days_in_advance = 3
timeout = 5" | $SUDO tee "$CONFIG_TARGET" >/dev/null
    fi
  fi
fi

echo_step "Installation complete"
if command -v "$APP_NAME" >/dev/null 2>&1; then
  echo "$APP_NAME version: $($APP_NAME --version 2>/dev/null || true)"
  echo "Binary: $(command -v $APP_NAME)"
fi

# Add convenient 'toss' alias unless disabled
if $add_alias; then
  add_shell_alias || true
fi

# Cleanup handled by trap
