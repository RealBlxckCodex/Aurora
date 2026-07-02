# Aurora TypeScript Client

OpenAI-compatible client for [Aurora](https://github.com/RealBlxckCodex/Aurora) audio inference engine.

## Install

```bash
npm install aurora-client
```

## Usage

```typescript
import { AuroraClient } from "aurora-client"

const aurora = new AuroraClient("http://localhost:11435")

// TTS
const audio = await aurora.tts("Hello world", {
  model: "kokoro-v1",
  voice: "af_heart",
})
const blob = new Blob([audio], { type: "audio/wav" })

// Streaming TTS
await aurora.streamTTS(
  "Hello from streaming Aurora",
  { model: "kokoro-v1", voice: "af_heart" },
  (chunk) => {
    const binary = atob(chunk.audio)
    const bytes = new Uint8Array(binary.length)
    for (let i = 0; i < binary.length; i++) bytes[i] = binary.charCodeAt(i)
    // play or buffer the chunk
  },
  () => console.log("Stream complete"),
)

// STT
const result = await aurora.transcribe(audioFile, {
  model: "whisper-turbo",
  language: "en",
})
console.log(result.text)

// Status / models / languages
const status = await aurora.status()
const models = await aurora.listModels()
const langs = await aurora.listLanguages()
```
