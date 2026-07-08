#!/bin/bash
set -euo pipefail

# ─────────────────────────────────────────────
#  Aurora Model Bundle Script
#  Packages local models for GitHub Releases
# ─────────────────────────────────────────────

GREEN='\033[0;32m'
CYAN='\033[0;36m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

info()  { echo -e "${CYAN}  →${NC} $1"; }
ok()    { echo -e "${GREEN}  ✓${NC} $1"; }
warn()  { echo -e "${YELLOW}  ⚠${NC} $1"; }
err()   { echo -e "${RED}  ✗${NC} $1"; }

MODELS_DIR="${1:-/var/aurora/models}"
OUTPUT_DIR="${2:-/tmp/aurora-bundle}"
RELEASE_TAG="${3:-v0.1.0}"
REPO="${REPO:-RealBlxckCodex/Aurora}"

mkdir -p "$OUTPUT_DIR"

# Models to bundle (small/medium ones only)
BUNDLE_MODELS=("kokoro-v1" "kokoro-de" "piper-de_DE" "whisper-turbo")

# Models that come from HuggingFace (too large for GH releases)
HF_MODELS=("orpheus-en" "orpheus-de")

echo ""
echo "  ╔══════════════════════════════════════╗"
echo "  ║     Aurora Model Bundle Script        ║"
echo "  ╚══════════════════════════════════════╝"
echo ""

info "Models dir: $MODELS_DIR"
info "Output dir: $OUTPUT_DIR"
info "Release tag: $RELEASE_TAG"
echo ""

RELEASE_BASE="https://github.com/$REPO/releases/download/$RELEASE_TAG"

# ── Generate manifest header ──
cat > "$OUTPUT_DIR/manifest.json" <<MANIFEST
{
  "schema_version": "1.0",
  "base_url": "$RELEASE_BASE",
  "models": {
MANIFEST

first_model=true

# ── Bundle small models ──
for model_id in "${BUNDLE_MODELS[@]}"; do
  model_dir="$MODELS_DIR/$model_id"
  if [ ! -d "$model_dir" ]; then
    warn "Model directory not found: $model_dir, skipping"
    continue
  fi

  archive_name="models-${model_id}.tar.gz"
  archive_path="$OUTPUT_DIR/$archive_name"

  info "Packaging $model_id..."

  # Resolve symlinks and package into tar.gz
  # --dereference follows symlinks and stores the actual file
  tar czf "$archive_path" --dereference --transform "s|.*/||" -C "$(dirname "$model_dir")" "$model_id" 2>/dev/null || {
    warn "Failed to package $model_id (stale symlink?), trying direct files..."
    # Fallback: find all actual files in the model dir
    tmp_dir=$(mktemp -d)
    find "$model_dir" -type f -exec cp -L {} "$tmp_dir/" \; 2>/dev/null || true
    tar czf "$archive_path" -C "$tmp_dir" .
    rm -rf "$tmp_dir"
  }

  size=$(stat -c%s "$archive_path" 2>/dev/null || echo 0)
  sha=$(sha256sum "$archive_path" | cut -d' ' -f1)
  ok "Created $archive_name ($(numfmt --to=iec-i $size))"

  # ── Determine file list from model dir ──
  files_json=""
  for f in "$model_dir"/*; do
    fname=$(basename "$f")
    if [ -z "$files_json" ]; then
      files_json=""
    fi
    fsize=$(stat -c%s "$f" 2>/dev/null || echo 0)
    fsha=$(sha256sum "$f" 2>/dev/null | cut -d' ' -f1 || echo "")
  done

  # Use the archive URL and SHA for model download
  model_json=$(cat <<MODELJSON
    "$model_id": {
      "name": "$model_id",
      "type": "tts",
      "format": "onnx",
      "version": "$RELEASE_TAG",
      "bundle_url": "$RELEASE_BASE/$archive_name",
      "bundle_sha256": "$sha",
      "bundle_size": $size,
      "files": {}
MODELJSON
)

  # Add individual file entries for backward compat
  first_file=true
  model_json="$model_json"$'\n'
  for f in "$model_dir"/*; do
    fname=$(basename "$f")
    if [ -L "$f" ]; then
      real_file=$(readlink -f "$f") || continue
    else
      real_file="$f"
    fi
    if [ ! -f "$real_file" ]; then
      continue
    fi
    fsize=$(stat -c%s "$real_file" 2>/dev/null || echo 0)
    fsha=$(sha256sum "$real_file" 2>/dev/null | cut -d' ' -f1 || echo "")
    if [ "$first_file" = true ]; then
      model_json="$model_json"$'      "files": {\n'
      first_file=false
    fi
  done
  if [ "$first_file" = false ]; then
    model_json="$model_json"$'      }'
  fi
  model_json="$model_json"$'\n    },'

  if [ "$first_model" = true ]; then
    echo "${model_json%,}" >> "$OUTPUT_DIR/manifest.json"
    first_model=false
  else
    echo "" >> "$OUTPUT_DIR/manifest.json"
    echo "${model_json%,}" >> "$OUTPUT_DIR/manifest.json"
  fi
done

# ── Add HF-only models to manifest ──
for model_id in "${HF_MODELS[@]}"; do
  model_dir="$MODELS_DIR/$model_id"
  if [ ! -d "$model_dir" ]; then
    warn "Model directory not found: $model_dir, skipping"
    continue
  fi

  # Get file info from real file (follow symlinks)
  for f in "$model_dir"/*; do
    fname=$(basename "$f")
    real_file=""
    if [ -L "$f" ]; then
      real_file=$(readlink -f "$f") || continue
    else
      real_file="$f"
    fi
    if [ ! -f "$real_file" ]; then
      continue
    fi
    fsize=$(stat -c%s "$real_file" 2>/dev/null || echo 0)
    fsha=$(sha256sum "$real_file" 2>/dev/null | cut -d' ' -f1 || echo "")

    # Determine HF URL based on model
    hf_url=""
    case "$model_id" in
      orpheus-en)
        hf_url="https://huggingface.co/isaiahbjork/orpheus-3b-0.1-ft-Q4_K_M-GGUF/resolve/main/orpheus-3b-0.1-ft-q4_k_m.gguf"
        ;;
      orpheus-de)
        hf_url="https://huggingface.co/freddyaboulton/3b-de-ft-research_release-Q4_K_M-GGUF/resolve/main/3b-de-ft-research_release-q4_k_m.gguf"
        ;;
    esac

    if [ -n "$hf_url" ]; then
      cat >> "$OUTPUT_DIR/manifest.json" <<MODEL
    "$model_id": {
      "name": "$model_id",
      "type": "tts",
      "format": "gguf",
      "version": "$RELEASE_TAG",
      "files": {
        "$fname": {
          "url": "$hf_url",
          "sha256": "$fsha",
          "size": $fsize
        }
      }
    },
MODEL
      ok "Added $model_id (HF source: ${fsize} bytes)"
    fi
  done
done

# ── Close manifest JSON — remove trailing comma and close ──
# Remove trailing comma from last model entry
if [[ "$(uname)" == "Darwin" ]]; then
  sed -i '' -e '$ s/,$//' "$OUTPUT_DIR/manifest.json"
else
  sed -i '$ s/,$//' "$OUTPUT_DIR/manifest.json"
fi
echo "  }" >> "$OUTPUT_DIR/manifest.json"
echo "}" >> "$OUTPUT_DIR/manifest.json"

ok "Manifest written to $OUTPUT_DIR/manifest.json"
echo ""

# ── Show usage instructions ──
echo "  ─────────────────────────────────────────────"
echo "  Bundle ready in: $OUTPUT_DIR"
echo ""
echo "  Upload to GitHub Release:"
echo "    $ gh release upload $RELEASE_TAG $OUTPUT_DIR/*.tar.gz"
echo ""
echo "  On target machine:"
echo "    # Install Aurora"
echo "    curl -fsSL https://raw.githubusercontent.com/$REPO/$RELEASE_TAG/install.sh | sh"
echo ""
echo "    # Install models (after uploading bundles)"
echo "    aurora pull --release $RELEASE_TAG kokoro-v1"
echo "    aurora pull --release $RELEASE_TAG kokoro-de"
echo "    aurora pull --release $RELEASE_TAG piper-de_DE"
echo "    aurora pull --release $RELEASE_TAG whisper-turbo"
echo ""
echo "    # Orpheus from HuggingFace:"
echo "    aurora pull hf.co/isaiahbjork/orpheus-3b-0.1-ft-Q4_K_M-GGUF:orpheus-3b-0.1-ft-q4_k_m.gguf"
echo "    aurora pull hf.co/freddyaboulton/3b-de-ft-research_release-Q4_K_M-GGUF:3b-de-ft-research_release-q4_k_m.gguf"
echo "  ─────────────────────────────────────────────"
echo ""
