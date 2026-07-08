#!/bin/bash
set -euo pipefail

# ─────────────────────────────────────────────
#  Aurora Installer  —  one-liner wie Ollama
#  curl -fsSL https://aurora.sh/install | sh
# ─────────────────────────────────────────────

REPO="RealBlxckCodex/Aurora"
BINARY="aurora"
VERSION="${AURORA_VERSION:-latest}"
DOWNLOAD_BASE="https://github.com/$REPO/releases/download"
CONFIG_DIR="/etc/aurora"
MODELS_DIR="/var/aurora/models"
BIN_DIR="/usr/local/bin"
SYSTEMD_DIR="/etc/systemd/system"

# ── Farben ──
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

# ── Check existing install ──
if command -v $BINARY &>/dev/null; then
  EXISTING_VERSION=$($BINARY --version 2>/dev/null || true)
  warn "$BINARY already installed${EXISTING_VERSION:+ ($EXISTING_VERSION)}"
  echo
  read -rp "  Overwrite? [Y/n] " REPLY
  if [[ "$REPLY" =~ ^[Nn] ]]; then
    info "Aborted."
    exit 0
  fi
fi

# ── Install system dependencies ──
DEPENDENCIES=""
if command -v apt-get &>/dev/null; then
  PKG_MANAGER="apt-get"
  DEPENDENCIES="espeak-ng"
elif command -v yum &>/dev/null; then
  PKG_MANAGER="yum"
  DEPENDENCIES="espeak-ng"
elif command -v apk &>/dev/null; then
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

if [ "$VERSION" = "latest" ] || curl -sfI "$DOWNLOAD_BASE/$VERSION/$BINARY-$OS-$ARCH_GO" &>/dev/null; then
  # ── Download pre-built ──
  if [ "$VERSION" = "latest" ]; then
    info "Fetching latest release..."
    TAG=$(curl -sf "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name"' | head -1 | sed 's/.*"tag_name": "\(.*\)",/\1/')
    if [ -z "$TAG" ]; then
      warn "Could not determine latest release tag, building from source"
      BUILD_FROM_SOURCE=1
    else
      URL="$DOWNLOAD_BASE/$TAG/$BINARY-$OS-$ARCH_GO"
      info "Downloading $BINARY $TAG..."
      curl -#fL "$URL" -o "$BINARY_PATH"
      chmod +x "$BINARY_PATH"
      ok "Downloaded $BINARY $TAG"
    fi
  else
    URL="$DOWNLOAD_BASE/$VERSION/$BINARY-$OS-$ARCH_GO"
    info "Downloading $BINARY $VERSION..."
    curl -#fL "$URL" -o "$BINARY_PATH"
    chmod +x "$BINARY_PATH"
    ok "Downloaded $BINARY $VERSION"
  fi
else
  BUILD_FROM_SOURCE=1
fi

if [ "${BUILD_FROM_SOURCE:-0}" = "1" ]; then
  # ── Build from source ──
  if ! command -v go &>/dev/null; then
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
ok "Installed to $BIN_DIR/$BINARY"

# ── Create directories ──
mkdir -p "$CONFIG_DIR" "$MODELS_DIR"
ok "Created $CONFIG_DIR and $MODELS_DIR"

# ── Default config ──
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
  warn "Config already exists, skipping"
fi

# ── systemd service ──
if command -v systemctl &>/dev/null; then
  SERVICE_FILE="$SYSTEMD_DIR/aurora.service"
  if [ ! -f "$SERVICE_FILE" ]; then
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
    ok "Created systemd service (aurora.service)"
    info "Start with: systemctl start aurora"
    info "Enable on boot: systemctl enable aurora"
  else
    warn "Systemd service already exists, skipping"
  fi
fi

# ── Done ──
cat <<EOF

  ${GREEN}✓ Aurora installed successfully${NC}

  ${CYAN}Commands:${NC}
    aurora serve              Start the API server
    aurora pull kokoro-v1     Pull a model
    aurora list               List available models

  ${CYAN}API:${NC}
    curl http://localhost:11435/v1/status

  ${CYAN}Systemd:${NC}
    systemctl start aurora
    systemctl enable aurora
    journalctl -u aurora -f

  ${CYAN}Docs:${NC}
    https://github.com/$REPO#readme

EOF
