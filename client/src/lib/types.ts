export type ConnectionMode = 'local' | 'qdrant-default' | 'qdrant-custom'

export interface ConnectionConfig {
  mode: ConnectionMode
  localPort: number
  qdrantUrl: string
  qdrantApiKey: string
}

export interface RagResult {
  filepath: string
  content: string
  score?: number
}

export interface Message {
  id: string
  role: 'user' | 'assistant'
  content: string
  sources?: RagResult[]
}

export const QDRANT_COLLECTION = 'hytale_codebase'
export const DEFAULT_LOCAL_PORT = 8080

export const AVAILABLE_MODELS = [
  { id: 'Llama-3.2-1B-Instruct-q4f16_1-MLC', label: 'Llama 3.2 1B (fast, ~0.7 GB)' },
  { id: 'Llama-3.2-3B-Instruct-q4f16_1-MLC', label: 'Llama 3.2 3B (better, ~1.8 GB)' },
  { id: 'Phi-3.5-mini-instruct-q4f16_1-MLC', label: 'Phi 3.5 Mini (code-focused, ~2.2 GB)' },
] as const

export type ModelId = (typeof AVAILABLE_MODELS)[number]['id']
