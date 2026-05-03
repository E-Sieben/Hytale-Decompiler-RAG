import { type ConnectionConfig, DEFAULT_LOCAL_PORT } from './types'

const STORAGE_KEY = 'hytale-rag-config'
const MODEL_KEY = 'hytale-rag-model'

export const DEFAULT_CONFIG: ConnectionConfig = {
  mode: 'qdrant-default',
  localPort: DEFAULT_LOCAL_PORT,
  qdrantUrl: '',
  qdrantApiKey: '',
}

export const DEFAULT_MODEL = 'Llama-3.2-1B-Instruct-q4f16_1-MLC'

export function loadConfig(): ConnectionConfig {
  if (typeof window === 'undefined') return DEFAULT_CONFIG
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    return raw ? { ...DEFAULT_CONFIG, ...JSON.parse(raw) } : DEFAULT_CONFIG
  } catch {
    return DEFAULT_CONFIG
  }
}

export function saveConfig(config: ConnectionConfig) {
  localStorage.setItem(STORAGE_KEY, JSON.stringify(config))
}

export function loadModel(): string {
  if (typeof window === 'undefined') return DEFAULT_MODEL
  return localStorage.getItem(MODEL_KEY) ?? DEFAULT_MODEL
}

export function saveModel(modelId: string) {
  localStorage.setItem(MODEL_KEY, modelId)
}
