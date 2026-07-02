# Aurora API Reference

## Base URL

- **Local server**: `http://localhost:11435`
- **Registry**: `http://localhost:8000/api/v1`

All endpoints return JSON unless noted otherwise.

---

## Inference API (OpenAI-compatible)

### Authentication

If `api.auth.enabled` is `true` in config:

```
Authorization: Bearer <api-key>
```

### `POST /v1/audio/speech` — Text-to-Speech

Generate speech from text. Compatible with OpenAI `POST /v1/audio/speech`.

#### Request

```json
{
  "model": "kokoro-v1",
  "input": "Hello from Aurora",
  "voice": "af_heart",
  "response_format": "wav",
  "speed": 1.0,
  "stream": false
}
```

| Field | Type | Default | Description |
|---|---|---|---|
| `model` | string | required | Model ID (`kokoro-v1`, `piper-de_DE`, `orpheus-en`) |
| `input` | string | required | Text to synthesize (max 4096 chars) |
| `voice` | string | `af_heart` | Voice ID (see model docs for available voices) |
| `response_format` | string | `wav` | Audio format (`wav`, `mp3`) |
| `speed` | float | `1.0` | Speech speed (0.5–2.0) |
| `stream` | bool | `false` | Enable SSE streaming of audio chunks |

#### Response (non-streaming)

`Content-Type: audio/wav` or `audio/mpeg`

Raw audio binary.

#### Response (streaming)

`Content-Type: text/event-stream`

```
data: {"audio":"<base64>","format":"wav","timestamp":0.0,"final":false}

data: {"audio":"<base64>","format":"wav","timestamp":0.1,"final":false}

...

data: {"audio":"<base64>","format":"wav","timestamp":1.5,"final":true}

data: [DONE]

```

Each SSE event payload:

| Field | Type | Description |
|---|---|---|
| `audio` | string | Base64-encoded audio chunk |
| `format` | string | Audio format (`wav`, `mp3`) |
| `timestamp` | float | Timestamp of chunk in seconds (native stream only) |
| `index` | int | Chunk index (fallback stream only) |
| `total_chunks` | int | Total number of chunks (fallback stream only) |
| `final` | bool | Whether this is the last audio chunk |

The stream terminates with `data: [DONE]\n\n`.

#### Errors

```json
{"error":"model is required"}
{"error":"input is required"}
{"error":"model not found"}
{"error":"inference failed: <details>"}
```

---

### `POST /v1/audio/transcriptions` — Speech-to-Text

Transcribe audio to text. Compatible with OpenAI `POST /v1/audio/transcriptions`.

#### Request

`Content-Type: multipart/form-data`

| Field | Type | Default | Description |
|---|---|---|---|
| `file` | file | required | Audio file (wav, mp3, ogg, flac, webm, m4a) |
| `model` | string | required | Model ID (`whisper-turbo`, `whisper-large-v3`) |
| `language` | string | auto | Language code (`de`, `en`, `fr`, ...) |
| `prompt` | string | — | Optional prompt for context |

#### Response

```json
{
  "text": "Hello from Aurora",
  "language": "en",
  "duration": 3.5
}
```

---

### `GET /v1/status` — Server Status

Server health and loaded models.

#### Response

```json
{
  "version": "1.0.0",
  "models": [
    {"id": "kokoro-v1", "backend": "onnx", "loaded": true}
  ]
}
```

---

### `GET /v1/models` — List Models

List all registered models.

#### Response

```json
{
  "models": [
    {"id": "kokoro-v1", "type": "tts", "backend": "onnx", "loaded": true},
    {"id": "whisper-turbo", "type": "stt", "backend": "whisper", "loaded": true}
  ]
}
```

---

### `GET /v1/languages` — List Languages

List supported languages and compatible models.

#### Response

```json
{
  "languages": [
    {
      "code": "de",
      "name": "German",
      "native_name": "Deutsch",
      "tts_models": ["kokoro-v1", "piper-de_DE", "orpheus-de"],
      "stt_models": ["whisper-large-v3", "whisper-turbo"]
    },
    {
      "code": "en",
      "name": "English",
      "native_name": "English",
      "tts_models": ["kokoro-v1", "orpheus-en"],
      "stt_models": ["whisper-large-v3", "whisper-turbo"]
    }
  ]
}
```

---

## Registry API

### `GET /api/v1/manifest.json` — Full Registry Manifest

Lists all public models with their latest version.

#### Response

```json
{
  "schema_version": "1.0",
  "base_url": "http://localhost:8000/api/v1",
  "models": {
    "kokoro-v1": {
      "name": "Kokoro TTS",
      "type": "tts",
      "format": "onnx",
      "version": "1.0.0",
      "files": {
        "model.onnx": {
          "url": "http://localhost:8000/api/v1/models/kokoro-v1/download/model.onnx",
          "sha256": "abc123...",
          "size": 337000000
        }
      },
      "voices": ["af_heart", "af_bella", ...],
      "languages": ["en", "fr", "ja", "ko", "zh", "it", "pt"]
    }
  }
}
```

---

### `GET /api/v1/models` — List Registry Models

Public facing model list.

#### Response

```json
{
  "data": [
    {
      "model_id": "kokoro-v1",
      "name": "Kokoro TTS",
      "type": "tts",
      "format": "onnx",
      "downloads": 42,
      "versions_count": 3,
      "languages": ["en", "fr"]
    }
  ]
}
```

Query params: `?type=tts&format=onnx&page=1&per_page=20`

---

### `GET /api/v1/models/{model_id}` — Model Detail

Full model details with readme, versions, files, voices, and preview URLs.

#### Response

```json
{
  "data": {
    "model_id": "kokoro-v1",
    "name": "Kokoro TTS",
    "description": "Lightweight multilingual TTS model",
    "type": "tts",
    "format": "onnx",
    "license": "MIT",
    "downloads": 42,
    "versions_count": 3,
    "readme": "<h1>Kokoro TTS</h1><p>...</p>",
    "languages": ["en", "fr", "ja", "ko", "zh", "it", "pt"],
    "versions": [
      {
        "version": "1.0.0",
        "changelog": "Initial release",
        "files": [
          {
            "filename": "model.onnx",
            "path": "models/kokoro-v1/1.0.0/model.onnx",
            "size": 337000000,
            "sha256": "abc123def456..."
          }
        ]
      }
    ],
    "voices": [
      {
        "voice_id": "af_heart",
        "name": "Heart",
        "language": "en-US",
        "gender": "female",
        "preview_url": "http://localhost:8000/storage/samples/kokoro-v1/af_heart.wav"
      }
    ]
  }
}
```

---

### `GET /api/v1/models/{model_id}/download/{file}` — Download Model File

Direct file download. Sets `Content-Disposition: attachment`.

---

## Code Examples

### cURL

```bash
# TTS (non-streaming)
curl http://localhost:11435/v1/audio/speech \
  -H "Content-Type: application/json" \
  -d '{"model":"kokoro-v1","input":"Hello world","voice":"af_heart","response_format":"wav"}' \
  --output hello.wav

# TTS (streaming)
curl -N http://localhost:11435/v1/audio/speech \
  -H "Content-Type: application/json" \
  -d '{"model":"kokoro-v1","input":"Hello world","voice":"af_heart","stream":true}'

# STT
curl http://localhost:11435/v1/audio/transcriptions \
  -F "model=whisper-turbo" \
  -F "file=@hello.wav" \
  -F "language=en"

# Status
curl http://localhost:11435/v1/status

# Registry manifest
curl http://localhost:8000/api/v1/manifest.json
```

---

### TypeScript

```typescript
// Using Web Audio API for playback
interface TTSChunk {
  audio: string;
  format: string;
  timestamp?: number;
  index?: number;
  total_chunks?: number;
  final: boolean;
}

async function streamTTS(text: string, model = "kokoro-v1", voice = "af_heart") {
  const response = await fetch("http://localhost:11435/v1/audio/speech", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ model, input: text, voice, stream: true }),
  });

  if (!response.ok || !response.body) {
    throw new Error(`TTS failed: ${response.statusText}`);
  }

  const reader = response.body.getReader();
  const decoder = new TextDecoder();
  let buffer = "";

  while (true) {
    const { done, value } = await reader.read();
    if (done) break;

    buffer += decoder.decode(value, { stream: true });
    const lines = buffer.split("\n");
    buffer = lines.pop() || "";

    for (const line of lines) {
      if (!line.startsWith("data: ")) continue;
      const payload = line.slice(6);

      if (payload === "[DONE]") {
        console.log("Stream complete");
        return;
      }

      const chunk: TTSChunk = JSON.parse(payload);
      const binary = atob(chunk.audio);
      const bytes = new Uint8Array(binary.length);
      for (let i = 0; i < binary.length; i++) {
        bytes[i] = binary.charCodeAt(i);
      }

      // Play or buffer the audio chunk
      const audioCtx = new AudioContext();
      const audioBuffer = await audioCtx.decodeAudioData(bytes.buffer);
      const source = audioCtx.createBufferSource();
      source.buffer = audioBuffer;
      source.connect(audioCtx.destination);
      source.start();
    }
  }
}

// OpenAI-compatible SDK usage
async function ttsNonStreaming(text: string): Promise<ArrayBuffer> {
  const response = await fetch("http://localhost:11435/v1/audio/speech", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      model: "kokoro-v1",
      input: text,
      voice: "af_heart",
      response_format: "wav",
    }),
  });
  return response.arrayBuffer();
}

// STT
async function transcribe(file: File): Promise<string> {
  const form = new FormData();
  form.append("model", "whisper-turbo");
  form.append("file", file);
  form.append("language", "en");

  const response = await fetch("http://localhost:11435/v1/audio/transcriptions", {
    method: "POST",
    body: form,
  });

  const data = await response.json();
  return data.text;
}
```

---

### PHP

```php
<?php

// TTS (non-streaming)
function tts(string $text, string $model = 'kokoro-v1', string $voice = 'af_heart'): string {
    $ch = curl_init('http://localhost:11435/v1/audio/speech');
    curl_setopt_array($ch, [
        CURLOPT_POST => true,
        CURLOPT_POSTFIELDS => json_encode([
            'model' => $model,
            'input' => $text,
            'voice' => $voice,
            'response_format' => 'wav',
        ]),
        CURLOPT_HTTPHEADER => ['Content-Type: application/json'],
        CURLOPT_RETURNTRANSFER => true,
    ]);
    $audio = curl_exec($ch);
    curl_close($ch);
    return $audio; // raw WAV bytes
}

// STT
function stt(string $audioPath, string $model = 'whisper-turbo'): array {
    $ch = curl_init('http://localhost:11435/v1/audio/transcriptions');
    curl_setopt_array($ch, [
        CURLOPT_POST => true,
        CURLOPT_POSTFIELDS => [
            'model' => $model,
            'file' => curl_file_create($audioPath),
            'language' => 'en',
        ],
        CURLOPT_RETURNTRANSFER => true,
    ]);
    $response = curl_exec($ch);
    curl_close($ch);
    return json_decode($response, true);
}

// Streaming TTS (SSE)
function streamTts(string $text, callable $onChunk): void {
    $ch = curl_init('http://localhost:11435/v1/audio/speech');
    curl_setopt_array($ch, [
        CURLOPT_POST => true,
        CURLOPT_POSTFIELDS => json_encode([
            'model' => 'kokoro-v1',
            'input' => $text,
            'voice' => 'af_heart',
            'stream' => true,
        ]),
        CURLOPT_HTTPHEADER => ['Content-Type: application/json'],
        CURLOPT_WRITEFUNCTION => function ($ch, $data) use ($onChunk) {
            foreach (explode("\n", $data) as $line) {
                if (!str_starts_with($line, 'data: ')) continue;
                $payload = substr($line, 6);
                if ($payload === '[DONE]') return 0;
                $onChunk(json_decode($payload, true));
            }
            return strlen($data);
        },
    ]);
    curl_exec($ch);
    curl_close($ch);
}

// Registry: list models
function listModels(): array {
    $response = file_get_contents('http://localhost:8000/api/v1/models');
    return json_decode($response, true)['data'] ?? [];
}

// Registry: model detail
function modelDetail(string $modelId): array {
    $response = file_get_contents("http://localhost:8000/api/v1/models/{$modelId}");
    return json_decode($response, true)['data'] ?? [];
}
```

---

### Python

```python
import requests
import json
import base64

# TTS (non-streaming)
def tts(text: str, model: str = "kokoro-v1", voice: str = "af_heart") -> bytes:
    resp = requests.post(
        "http://localhost:11435/v1/audio/speech",
        json={"model": model, "input": text, "voice": voice, "response_format": "wav"},
    )
    resp.raise_for_status()
    return resp.content  # raw WAV bytes

# STT
def stt(audio_path: str, model: str = "whisper-turbo") -> dict:
    with open(audio_path, "rb") as f:
        resp = requests.post(
            "http://localhost:11435/v1/audio/transcriptions",
            files={"file": f},
            data={"model": model, "language": "en"},
        )
    resp.raise_for_status()
    return resp.json()

# TTS (streaming)
def stream_tts(text: str, model: str = "kokoro-v1", voice: str = "af_heart"):
    resp = requests.post(
        "http://localhost:11435/v1/audio/speech",
        json={"model": model, "input": text, "voice": voice, "stream": True},
        stream=True,
    )
    resp.raise_for_status()

    for line in resp.iter_lines():
        if not line:
            continue
        if not line.startswith(b"data: "):
            continue
        payload = line[6:]

        if payload == b"[DONE]":
            print("Stream complete")
            break

        chunk = json.loads(payload)
        audio_bytes = base64.b64decode(chunk["audio"])
        yield audio_bytes

# Usage
for chunk in stream_tts("Hello from Python"):
    # Play, save, or process each chunk
    print(f"Got chunk: {len(chunk)} bytes")

# Registry manifest
manifest = requests.get("http://localhost:8000/api/v1/manifest.json").json()
for model_id, info in manifest["models"].items():
    print(f"{model_id}: {info['version']} ({info['type']})")
```

---

### Go

```go
package main

import (
    "bytes"
    "encoding/base64"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "strings"
)

type TTSRequest struct {
    Model          string  `json:"model"`
    Input          string  `json:"input"`
    Voice          string  `json:"voice"`
    ResponseFormat string  `json:"response_format"`
    Speed          float64 `json:"speed"`
    Stream         bool    `json:"stream"`
}

type TTSChunk struct {
    Audio   string  `json:"audio"`
    Format  string  `json:"format"`
    Index   int     `json:"index,omitempty"`
    Final   bool    `json:"final"`
}

// TTS non-streaming
func TTS(text, model, voice string) ([]byte, error) {
    body, _ := json.Marshal(TTSRequest{
        Model: model, Input: text, Voice: voice,
        ResponseFormat: "wav",
    })
    resp, err := http.Post(
        "http://localhost:11435/v1/audio/speech",
        "application/json",
        bytes.NewReader(body),
    )
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    return io.ReadAll(resp.Body)
}

// STT
func STT(audioPath, model string) (string, error) {
    // Use multipart/form-data
    var b bytes.Buffer
    w := multipart.NewWriter(&b)
    w.WriteField("model", model)
    fw, _ := w.CreateFormFile("file", "audio.wav")
    audio, _ := os.ReadFile(audioPath)
    fw.Write(audio)
    w.Close()

    resp, err := http.Post(
        "http://localhost:11435/v1/audio/transcriptions",
        w.FormDataContentType(),
        &b,
    )
    if err != nil {
        return "", err
    }
    defer resp.Body.Close()

    var result struct {
        Text string `json:"text"`
    }
    json.NewDecoder(resp.Body).Decode(&result)
    return result.Text, nil
}

// TTS streaming (SSE)
func StreamTTS(text, model, voice string, onChunk func([]byte)) error {
    body, _ := json.Marshal(TTSRequest{
        Model: model, Input: text, Voice: voice,
        Stream: true,
    })
    resp, err := http.Post(
        "http://localhost:11435/v1/audio/speech",
        "application/json",
        bytes.NewReader(body),
    )
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    reader := bufio.NewReader(resp.Body)
    for {
        line, err := reader.ReadString('\n')
        if err != nil {
            return err
        }
        line = strings.TrimSpace(line)
        if !strings.HasPrefix(line, "data: ") {
            continue
        }
        payload := strings.TrimPrefix(line, "data: ")
        if payload == "[DONE]" {
            return nil
        }
        var chunk TTSChunk
        if err := json.Unmarshal([]byte(payload), &chunk); err != nil {
            continue
        }
        audio, err := base64.StdEncoding.DecodeString(chunk.Audio)
        if err != nil {
            continue
        }
        onChunk(audio)
    }
}

// Registry manifest
func FetchManifest(registryURL string) (*Manifest, error) {
    resp, err := http.Get(registryURL + "/api/v1/manifest.json")
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    var manifest Manifest
    return &manifest, json.NewDecoder(resp.Body).Decode(&manifest)
}
```

---

## Status Codes

| Code | Description |
|---|---|
| `200` | Success (JSON body or raw audio) |
| `400` | Bad request (missing required fields) |
| `404` | Model not found |
| `500` | Internal server error (inference failure) |

## Content Types

| Endpoint | Request | Response |
|---|---|---|
| `POST /v1/audio/speech` (non-stream) | `application/json` | `audio/wav` or `audio/mpeg` |
| `POST /v1/audio/speech` (stream) | `application/json` | `text/event-stream` |
| `POST /v1/audio/transcriptions` | `multipart/form-data` | `application/json` |
| `GET /v1/status` | — | `application/json` |
| `GET /v1/models` | — | `application/json` |
| `GET /v1/languages` | — | `application/json` |
| `GET /api/v1/manifest.json` | — | `application/json` |
| `GET /api/v1/models` | — | `application/json` |
| `GET /api/v1/models/{id}` | — | `application/json` |
| `GET /api/v1/models/{id}/download/{file}` | — | `application/octet-stream` |
