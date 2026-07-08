#!/bin/bash
set -euo pipefail

REPO="RealBlxckCodex/Aurora"
BINARY="aurora"
VERSION="${AURORA_VERSION:-latest}"
RELEASE_TAG="${AURORA_RELEASE_TAG:-}"
DOWNLOAD_BASE="https://github.com/$REPO/releases/download"
CONFIG_DIR="/etc/aurora"
MODELS_DIR="/var/aurora/models"
BIN_DIR="/usr/local/bin"
SYSTEMD_DIR="/etc/systemd/system"

GREEN='\033[0;32m'
CYAN='\033[0;36m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

info()  { echo -e "${CYAN}  →${NC} $1"; }
ok()    { echo -e "${GREEN}  ✓${NC} $1"; }
warn()  { echo -e "${YELLOW}  ⚠${NC} $1"; }
err()   { echo -e "${RED}  ✗${NC} $1"; }

# ── Header ──
cat <<EOF

  ╔══════════════════════════════════════╗
  ║          Aurora Installer            ║
  ║   Self-hosted Audio Inference        ║
  ╚══════════════════════════════════════╝

EOF

# ── Root check ──
if [ "$(id -u)" -ne 0 ]; then
  err "This installer must be run as root (or with sudo)"
  exit 1
fi

# ── Architecture ──
ARCH=$(uname -m)
OS=$(uname -s | tr '[:upper:]' '[:lower:]')

case "$ARCH" in
  x86_64)  ARCH_GO="amd64" ;;
  aarch64) ARCH_GO="arm64" ;;
  armv7l)  ARCH_GO="arm"   ;;
  *)
    err "Unsupported architecture: $ARCH"
    exit 1
    ;;
esac

info "Detected: $OS / $ARCH"

# ── Existing install check ──
EXISTING_VERSION=""
if command -v $BINARY >/dev/null 2>&1; then
  EXISTING_VERSION=$($BINARY --version 2>/dev/null || echo "unknown")
fi

# ── Parse args for --force / --update-models ──
FORCE=0
UPDATE_MODELS=0
for arg in "$@"; do
  case "$arg" in
    --force|-f) FORCE=1 ;;
    --update-models|-m) UPDATE_MODELS=1 ;;
  esac
done

# ── Stop service if running ──
SERVICE_RUNNING=0
if command -v systemctl >/dev/null 2>&1 && systemctl is-active --quiet aurora.service 2>/dev/null; then
  SERVICE_RUNNING=1
  if [ "$FORCE" -eq 1 ]; then
    info "Stopping aurora.service..."
    systemctl stop aurora.service
  fi
fi

# ── Prompt for update vs fresh install ──
if [ -n "$EXISTING_VERSION" ]; then
  warn "$BINARY already installed ($EXISTING_VERSION)"
  AUTO_OVERWRITE=0
  if [ "$FORCE" -eq 1 ]; then
    AUTO_OVERWRITE=1
  elif [ -t 0 ]; then
    echo
    read -rp "  Update to latest? [Y/n] " REPLY
    if printf '%s' "$REPLY" | grep -iq '^n'; then
      info "Aborted."
      exit 0
    fi
    echo
    read -rp "  Also update models? [y/N] " REPLY
    if printf '%s' "$REPLY" | grep -iq '^y'; then
      UPDATE_MODELS=1
    fi
  else
    info "Proceeding with update (non-interactive). Use --force to skip prompts."
  fi
  info "Updating $BINARY $EXISTING_VERSION → latest..."
fi

# ── Install system dependencies ──
DEPENDENCIES=""
if command -v apt-get >/dev/null 2>&1; then
  PKG_MANAGER="apt-get"
  DEPENDENCIES="espeak-ng"
elif command -v yum >/dev/null 2>&1; then
  PKG_MANAGER="yum"
  DEPENDENCIES="espeak-ng"
elif command -v apk >/dev/null 2>&1; then
  PKG_MANAGER="apk"
  DEPENDENCIES="espeak-ng"
else
  PKG_MANAGER=""
  warn "No package manager found; skip system deps"
fi

if [ -n "$PKG_MANAGER" ]; then
  info "Installing system dependencies ($DEPENDENCIES)..."
  case "$PKG_MANAGER" in
    apt-get) apt-get update -qq && apt-get install -y -qq $DEPENDENCIES ;;
    yum)     yum install -y -q $DEPENDENCIES ;;
    apk)     apk add --no-cache $DEPENDENCIES ;;
  esac
  ok "System dependencies installed"
fi

# ── Download or build ──
TMP_DIR=$(mktemp -d)
trap 'rm -rf "$TMP_DIR"' EXIT

BINARY_PATH="$TMP_DIR/$BINARY"
BUILD_FROM_SOURCE=0

if [ "$VERSION" = "latest" ] || curl -sfI "$DOWNLOAD_BASE/$VERSION/$BINARY-$OS-$ARCH_GO" >/dev/null 2>&1; then
  if [ "$VERSION" = "latest" ]; then
    info "Fetching latest release..."
    TAG=$(curl -sf "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name"' | head -1 | sed 's/.*"tag_name": "\(.*\)",/\1/')
    if [ -z "$TAG" ]; then
      warn "Could not determine latest release tag, building from source"
      BUILD_FROM_SOURCE=1
    else
      URL="$DOWNLOAD_BASE/$TAG/$BINARY-$OS-$ARCH_GO"
      RELEASE_TAG="${RELEASE_TAG:-$TAG}"
      info "Downloading $BINARY $TAG..."
      curl -#fL "$URL" -o "$BINARY_PATH"
      chmod +x "$BINARY_PATH"
      ok "Downloaded $BINARY $TAG"
    fi
  else
    URL="$DOWNLOAD_BASE/$VERSION/$BINARY-$OS-$ARCH_GO"
    RELEASE_TAG="${RELEASE_TAG:-$VERSION}"
    info "Downloading $BINARY $VERSION..."
    curl -#fL "$URL" -o "$BINARY_PATH"
    chmod +x "$BINARY_PATH"
    ok "Downloaded $BINARY $VERSION"
  fi
else
  BUILD_FROM_SOURCE=1
fi

if [ "$BUILD_FROM_SOURCE" -eq 1 ]; then
  if ! command -v go >/dev/null 2>&1; then
    info "Go not found — installing Go $ARCH_GO..."
    GO_VERSION="1.24.1"
    GO_TAR="go$GO_VERSION.$OS-$ARCH_GO.tar.gz"
    curl -sfL "https://go.dev/dl/$GO_TAR" -o "$TMP_DIR/$GO_TAR"
    tar -C "$TMP_DIR" -xzf "$TMP_DIR/$GO_TAR"
    export PATH="$TMP_DIR/go/bin:$PATH"
    ok "Go $GO_VERSION installed"
  fi

  info "Building $BINARY from source..."
  SRC_DIR="$TMP_DIR/src"
  git clone --depth 1 "https://github.com/$REPO.git" "$SRC_DIR" 2>/dev/null || {
    err "Failed to clone repository"
    exit 1
  }
  (cd "$SRC_DIR" && go build -o "$BINARY_PATH" ./cmd/aurora)
  chmod +x "$BINARY_PATH"
  ok "Built $BINARY from source"
fi

# ── Install binary ──
install -m 755 "$BINARY_PATH" "$BIN_DIR/$BINARY"
ok "Installed $BINARY to $BIN_DIR/$BINARY"

# ── Create directories ──
mkdir -p "$CONFIG_DIR" "$MODELS_DIR"

# ── Default config (only if not exists) ──
if [ ! -f "$CONFIG_DIR/aurora.yaml" ]; then
  cat > "$CONFIG_DIR/aurora.yaml" <<CONF
server:
  host: "0.0.0.0"
  port: 11435
  workers: 4

models:
  dir: "$MODELS_DIR"
  registry_url: "http://localhost:8000"

hardware:
  cpu:
    enabled: true
    threads: 0
  gpu:
    enabled: auto

api:
  auth:
    enabled: false
  rate_limit:
    enabled: false
  cors:
    enabled: true
    origins: ["*"]

logging:
  level: "info"
  format: "json"
  output: "stdout"
CONF
  ok "Created default config at $CONFIG_DIR/aurora.yaml"
else
  ok "Config exists at $CONFIG_DIR/aurora.yaml"
fi

# ── systemd service (always update on reinstall) ──
if command -v systemctl >/dev/null 2>&1; then
  SERVICE_FILE="$SYSTEMD_DIR/aurora.service"
  cat > "$SERVICE_FILE" <<UNIT
[Unit]
Description=Aurora Audio Inference Engine
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=$BIN_DIR/$BINARY serve --config $CONFIG_DIR/aurora.yaml
Restart=on-failure
RestartSec=5
LimitNOFILE=65536

[Install]
WantedBy=multi-user.target
UNIT
  systemctl daemon-reload
  ok "Updated systemd service (aurora.service)"
fi

# ── Restart service if it was running ──
if [ "$SERVICE_RUNNING" -eq 1 ]; then
  info "Restarting aurora.service..."
  systemctl start aurora.service
  ok "aurora.service restarted"
fi

# ── Models (optional) ──
if [ "$UPDATE_MODELS" -eq 1 ] || [ "${AURORA_INSTALL_MODELS:-0}" = "1" ]; then
  if [ -n "$RELEASE_TAG" ]; then
    info "Installing models from release $RELEASE_TAG..."
    $BIN_DIR/$BINARY pull --release "$RELEASE_TAG" --all || warn "Model installation incomplete"
  elif [ -z "$RELEASE_TAG" ] && [ "${AURORA_INSTALL_MODELS:-0}" = "1" ]; then
    info "Installing all models from manifest..."
    $BIN_DIR/$BINARY pull --all || warn "Model installation incomplete"
  else
    warn "No release tag set. Use AURORA_RELEASE_TAG=v0.1.0 or run: aurora pull <model>"
  fi
fi

# ── Done ──
cat <<EOF

  ${GREEN}✓ Aurora installed successfully${NC}

  ${CYAN}Commands:${NC}
    aurora serve              Start the API server
    aurora pull <model>       Pull a model from registry
    aurora pull --all         Install all models from manifest
    aurora pull hf.co/...     Pull from HuggingFace
    aurora list               List available models

  ${CYAN}API:${NC}
    curl http://localhost:11435/v1/status

  ${CYAN}Systemd:${NC}
    systemctl start aurora
    systemctl enable aurora
    journalctl -u aurora -f

  ${CYAN}Update:${NC}
    curl -fsSL https://raw.githubusercontent.com/$REPO/main/install.sh | sudo sh
    curl -fsSL https://raw.githubusercontent.com/$REPO/main/install.sh | sudo sh -s -- -m   # + update models

EOF
