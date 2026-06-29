# Aurora

**Self-hosted, CPU-first audio inference engine** — TTS and STT with an OpenAI-compatible API.

[![Go](https://img.shields.io/badge/Go-1.24+-00ADD8?logo=go)](https://go.dev)
[![License](https://img.shields.io/badge/License-MIT-green)](LICENSE)
[![Build](https://img.shields.io/badge/build-passing-brightgreen)](#)

Aurora is the Ollama for audio: pull models, run them locally, and integrate via a drop-in OpenAI `v1/audio` replacement. Everything runs on CPU (GPU optional), making it ideal for Raspberry Pi, edge servers, and air-gapped environments.

---

## Quick start

```bash
# Build from source
make build

# Start the API server
./bin/aurora serve

# In another terminal, pull a model
./bin/aurora pull kokoro-v1

# Generate speech
curl http://localhost:11435/v1/audio/speech \
  -H "Content-Type: application/json" \
  -d '{"model":"kokoro-v1","input":"Hello from Aurora","voice":"af_heart","response_format":"wav"}' \
  --output hello.wav

# Transcribe
curl http://localhost:11435/v1/audio/transcriptions \
  -F "model=whisper-turbo" \
  -F "file=@hello.wav"
```

---

## Installation

### Prerequisites

- **Go** 1.24+
- **espeak-ng** (for graphene-to-phoneme, used by Kokoro TTS)
  ```bash
  apt install espeak-ng   # Debian/Ubuntu
  brew install espeak-ng  # macOS
  ```
- **ONNX Runtime** shared library (optional, for Kokoro/Piper)
  ```bash
  # Download from https://github.com/microsoft/onnxruntime/releases
  # Install to /usr/local/onnxruntime/
  ```

### From source

```bash
git clone https://github.com/hyper-it/aurora.git
cd aurora
make build
```

The build produces three binaries in `bin/`:
| Binary | Purpose |
|---|---|
| `bin/aurora` | Full CLI — serve, pull, list, rm, g2p |
| `bin/aurora-tts` | Standalone TTS (Piper/Kokoro) |
| `bin/aurora-stt` | Standalone STT (Whisper) |

### Config file

Aurora reads `/etc/aurora/aurora.yaml` by default. A minimal config:

```yaml
server:
  host: "0.0.0.0"
  port: 11435
  workers: 4

models:
  dir: "/var/aurora/models"
  registry_url: "http://localhost:8000"

hardware:
  cpu:
    enabled: true
    threads: 0
```

See `config/aurora.yaml` for all options.

---

## CLI reference

### `aurora serve`

Start the API server.

```bash
aurora serve [--port 11435] [--host 0.0.0.0] [--cpu-only] [--config /path/to/config.yaml]
```

### `aurora pull`

Download a model from the registry or HuggingFace.

```bash
aurora pull kokoro-v1              # from registry (latest)
aurora pull kokoro-v1:1.0.0        # specific version
aurora pull hf.co/namespace/repo:file  # from HuggingFace
```

Registry URL defaults to `http://localhost:8000`. Override with `--registry` or `models.registry_url` in config.

### `aurora list`

List available and installed models.

```bash
aurora list                  # table output
aurora list --type tts       # filter by type
aurora list --installed      # only installed
aurora list --format json    # JSON output
```

### `aurora rm`

Remove an installed model.

```bash
aurora rm kokoro-v1
aurora rm kokoro-v1 --force
```

### `aurora g2p`

Convert text to IPA phonemes (grapheme-to-phoneme).

```bash
aurora g2p "Hallo Welt"              # German (default)
aurora g2p "Hello world" --lang en   # English
aurora g2p "Hallo" --format phonemes # compact phoneme format
```

---

## API reference

Aurora exposes an OpenAI-compatible API at the configured server address.

### `GET /v1/status`

Server health and loaded models.

```json
{
  "version": "1.0.0",
  "models": [
    {"id": "kokoro-v1", "backend": "onnx", "loaded": true}
  ]
}
```

### `POST /v1/audio/speech`

Generate speech from text (OpenAI `POST /v1/audio/speech` compatible).

**Request:**
```json
{
  "model": "kokoro-v1",
  "input": "Hello from Aurora",
  "voice": "af_heart",
  "response_format": "wav",
  "speed": 1.0
}
```

| Parameter | Type | Default | Description |
|---|---|---|---|
| `model` | string | required | Model ID (`kokoro-v1`, `kokoro-de`, `piper-de_DE`, `orpheus-en`, `orpheus-de`) |
| `input` | string | required | Text to synthesize |
| `voice` | string | `af_heart` | Voice ID (see model docs for available voices) |
| `response_format` | string | `wav` | Audio format (`wav`, `mp3`) |
| `speed` | float | `1.0` | Speech speed multiplier (0.5–2.0) |

**Response:** Raw audio binary (`audio/wav` or `audio/mpeg`).

### `POST /v1/audio/transcriptions`

Transcribe speech to text (OpenAI `POST /v1/audio/transcriptions` compatible).

**Request:** `multipart/form-data`

| Field | Type | Default | Description |
|---|---|---|---|
| `file` | file | required | Audio file (wav, mp3, ogg, flac) |
| `model` | string | required | Model ID (`whisper-turbo`, `whisper-large-v3`) |
| `language` | string | auto | Language code (`de`, `en`, ...) |

**Response:**
```json
{
  "text": "Hello from Aurora",
  "language": "en",
  "duration": 3.5
}
```

### `GET /v1/models`

List loaded inference models.

### `GET /v1/languages`

List supported languages and compatible models.

---

## Models

### TTS models

| Model | Format | Size | Voices | Languages |
|---|---|---|---|---|
| `kokoro-v1` | ONNX | 337 MB | 7 built-in | en, fr, ja, ko, zh, it, pt |
| `kokoro-de` | ONNX | 311 MB | martin | de |
| `piper-de_DE` | ONNX | 150 MB | thorsten | de |
| `orpheus-en` | GGUF 3B | 2.25 GB | tara, leah, jesper + more | en |
| `orpheus-de` | GGUF 3B | 2.0 GB | default | de |

### STT models

| Model | Format | Size | Languages |
|---|---|---|---|
| `whisper-turbo` | GGUF (q5_0) | 548 MB | 99+ (multilingual) |
| `whisper-large-v3` | GGUF (q5_0) | 3.1 GB | 99+ (multilingual) |

### Voice features

- **Kokoro:** natural prosody, warm voices, Flow Matching architecture (~82M params)
- **Piper:** ultra-fast CPU inference, MIT licensed
- **Orpheus:** emotion/laughter/singing via prompt tags (`[laugh]`, `[whisper]`, etc.), 3B param Llama-based
- **Whisper:** industry-standard STT, word-level timestamps, auto language detection

---

## Architecture

```
┌──────────────────────────────────────────┐
│                CLI (cobra)                 │
│  serve  pull  list  rm  g2p              │
└────────────┬─────────────────────────────┘
             │
┌────────────▼─────────────────────────────┐
│           API Server (net/http)           │
│  POST /v1/audio/speech                   │
│  POST /v1/audio/transcriptions           │
│  GET  /v1/status /v1/models /v1/languages│
└────────────┬─────────────────────────────┘
             │
┌────────────▼─────────────────────────────┐
│         Inference Backends                │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐  │
│  │   ONNX   │ │   GGML   │ │ Whisper  │  │
│  │ Kokoro   │ │ Orpheus  │ │ STT      │  │
│  │ Piper    │ │          │ │          │  │
│  └──────────┘ └──────────┘ └──────────┘  │
└────────────┬─────────────────────────────┘
             │
┌────────────▼─────────────────────────────┐
│       Hardware Abstraction Layer          │
│  CPU (AVX-512, AMX) | GPU (CUDA/Vulkan)  │
│  HybridScheduler for multi-backend        │
└──────────────────────────────────────────┘
```

### Backends

| Backend | Technology | Models |
|---|---|---|
| ONNX | ONNX Runtime Go bindings | Kokoro TTS, Piper TTS |
| GGML | llama.cpp process pool | Orpheus TTS (GGUF) |
| Whisper | whisper.cpp subprocess | Whisper STT |

---

## Configuration

Full configuration reference (`aurora.yaml`):

```yaml
server:
  host: "0.0.0.0"          # Listen address
  port: 11435               # Listen port
  workers: 4                # Worker thread pool

hardware:
  cpu:
    enabled: true
    threads: 0              # 0 = auto-detect
    avx512: auto            # auto, enable, disable
    amx: auto
    memory_limit: "8GB"
  gpu:
    enabled: auto           # auto, true, false
    devices: [0]
    vram_limit: "12GB"
    fallback_to_cpu: true   # fall back if GPU init fails

models:
  dir: "/var/aurora/models"  # Model storage directory
  registry_url: "http://localhost:8000"  # Registry API URL

api:
  auth:
    enabled: false
  rate_limit:
    enabled: false
  cors:
    enabled: true
    origins: ["*"]

logging:
  level: "info"            # debug, info, warn, error
  format: "json"           # json, text
  output: "stdout"
```

---

## Development

### Quick build & test

```bash
make build    # builds all binaries
make test     # runs go test ./...
make lint     # runs go vet + golint
make clean    # removes bin/
```

### Project structure

```
cmd/
  aurora/               # Main CLI entry point
  aurora-tts/           # Standalone TTS
  aurora-stt/           # Standalone STT
internal/
  api/                  # HTTP server & handlers
  cli/                  # Cobra commands
  config/               # YAML config loading
  g2p/                  # Grapheme-to-phoneme
  hardware/             # CPU/GPU detection & scheduling
  inference/            # Inference backends
  models/               # Model manager, downloader, store
pkg/
  domain/               # Domain types (Model, Voice, Language)
ext/
  whisper.cpp/          # Bundled whisper.cpp
  llama.cpp/            # Bundled llama.cpp
config/
  aurora.yaml           # Sample config
models/
  manifest.json         # Model definitions
```

---

## License

MIT. See [LICENSE](LICENSE).

Aurora is community-driven and built for self-hosters, edge deployments, and anyone who wants private, local audio inference without external dependencies.
