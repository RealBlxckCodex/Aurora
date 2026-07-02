export interface TTSOptions {
  model?: string
  voice?: string
  responseFormat?: "wav" | "mp3"
  speed?: number
  stream?: boolean
}

export interface TTSChunk {
  audio: string
  format: string
  timestamp?: number
  index?: number
  totalChunks?: number
  final: boolean
}

export interface STTOptions {
  model?: string
  language?: string
  prompt?: string
}

export interface STTResult {
  text: string
  language: string
  duration: number
}

export interface ModelInfo {
  id: string
  type: string
  backend: string
  loaded: boolean
}

export interface ServerStatus {
  version: string
  models: ModelInfo[]
}

export interface LanguageInfo {
  code: string
  name: string
  nativeName: string
  ttsModels: string[]
  sttModels: string[]
}

export class AuroraClient {
  private baseURL: string
  private apiKey?: string

  constructor(baseURL = "http://localhost:11435", apiKey?: string) {
    this.baseURL = baseURL.replace(/\/+$/, "")
    this.apiKey = apiKey
  }

  private headers(extra?: Record<string, string>): Record<string, string> {
    const h: Record<string, string> = extra ?? {}
    if (this.apiKey) {
      h["Authorization"] = `Bearer ${this.apiKey}`
    }
    return h
  }

  async tts(
    input: string,
    options: TTSOptions = {},
  ): Promise<ArrayBuffer> {
    const resp = await fetch(`${this.baseURL}/v1/audio/speech`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        ...this.headers(),
      },
      body: JSON.stringify({
        model: options.model ?? "kokoro-v1",
        input,
        voice: options.voice ?? "af_heart",
        response_format: options.responseFormat ?? "wav",
        speed: options.speed ?? 1.0,
        stream: false,
      }),
    })
    if (!resp.ok) {
      const err = await resp.json().catch(() => ({ error: resp.statusText }))
      throw new Error(err.error)
    }
    return resp.arrayBuffer()
  }

  async streamTTS(
    input: string,
    options: TTSOptions = {},
    onChunk: (chunk: TTSChunk) => void,
    onDone?: () => void,
  ): Promise<void> {
    const resp = await fetch(`${this.baseURL}/v1/audio/speech`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        ...this.headers(),
      },
      body: JSON.stringify({
        model: options.model ?? "kokoro-v1",
        input,
        voice: options.voice ?? "af_heart",
        response_format: options.responseFormat ?? "wav",
        speed: options.speed ?? 1.0,
        stream: true,
      }),
    })
    if (!resp.ok || !resp.body) {
      const err = await resp.json().catch(() => ({ error: resp.statusText }))
      throw new Error(err.error)
    }

    const reader = resp.body.getReader()
    const decoder = new TextDecoder()
    let buffer = ""

    while (true) {
      const { done, value } = await reader.read()
      if (done) break

      buffer += decoder.decode(value, { stream: true })
      const lines = buffer.split("\n")
      buffer = lines.pop() ?? ""

      for (const line of lines) {
        const trimmed = line.trim()
        if (!trimmed.startsWith("data: ")) continue
        const payload = trimmed.slice(6)
        if (payload === "[DONE]") {
          onDone?.()
          return
        }
        onChunk(JSON.parse(payload) as TTSChunk)
      }
    }
  }

  async transcribe(
    file: Blob | File,
    options: STTOptions = {},
  ): Promise<STTResult> {
    const form = new FormData()
    form.append("model", options.model ?? "whisper-turbo")
    form.append("file", file)
    if (options.language) form.append("language", options.language)
    if (options.prompt) form.append("prompt", options.prompt)

    const resp = await fetch(`${this.baseURL}/v1/audio/transcriptions`, {
      method: "POST",
      body: form,
    })
    if (!resp.ok) {
      const err = await resp.json().catch(() => ({ error: resp.statusText }))
      throw new Error(err.error)
    }
    return resp.json()
  }

  async status(): Promise<ServerStatus> {
    const resp = await fetch(`${this.baseURL}/v1/status`, {
      headers: this.headers(),
    })
    return resp.json()
  }

  async listModels(): Promise<ModelInfo[]> {
    const resp = await fetch(`${this.baseURL}/v1/models`, {
      headers: this.headers(),
    })
    const data = await resp.json()
    return data.models as ModelInfo[]
  }

  async listLanguages(): Promise<LanguageInfo[]> {
    const resp = await fetch(`${this.baseURL}/v1/languages`, {
      headers: this.headers(),
    })
    const data = await resp.json()
    return data.languages as LanguageInfo[]
  }
}
