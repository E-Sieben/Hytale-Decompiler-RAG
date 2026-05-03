'use client'

import type { ConnectionConfig, RagResult } from './types'
import { QDRANT_COLLECTION } from './types'
import { getEmbedding } from './embed'

export async function search(
  query: string,
  config: ConnectionConfig,
  n = 5,
): Promise<RagResult[]> {
  switch (config.mode) {
    case 'local':
      return searchLocal(query, config.localPort, n)
    case 'qdrant-default':
      return searchQdrantDefault(query, n)
    case 'qdrant-custom':
      return searchQdrantCustom(query, config.qdrantUrl, config.qdrantApiKey, n)
  }
}

// Local Docker (ChromaDB) — server embeds the query text itself.
async function searchLocal(query: string, port: number, n: number): Promise<RagResult[]> {
  const url = `http://localhost:${port}/search?q=${encodeURIComponent(query)}&n=${n}`
  const res = await fetch(url)
  if (!res.ok) throw new Error(`Local RAG server returned ${res.status}. Is Docker running?`)
  const { results } = (await res.json()) as { results: RagResult[] }
  return results
}

// Qdrant via Vercel serverless route — API key stays server-side.
async function searchQdrantDefault(query: string, n: number): Promise<RagResult[]> {
  const vector = await getEmbedding(query)
  const res = await fetch('/api/rag', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ vector, n }),
  })
  if (!res.ok) {
    const { error } = await res.json().catch(() => ({ error: res.statusText }))
    throw new Error(error)
  }
  const { results } = (await res.json()) as { results: RagResult[] }
  return results
}

// Qdrant with user-supplied credentials — queries the REST API directly from the browser.
async function searchQdrantCustom(
  query: string,
  url: string,
  apiKey: string,
  n: number,
): Promise<RagResult[]> {
  const vector = await getEmbedding(query)
  const res = await fetch(`${url}/collections/${QDRANT_COLLECTION}/points/search`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', 'api-key': apiKey },
    body: JSON.stringify({ vector, limit: n, with_payload: true }),
  })
  if (!res.ok) throw new Error(`Qdrant responded with ${res.status}`)
  const { result } = await res.json()
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  return result.map((p: any) => ({
    filepath: p.payload?.filepath ?? '',
    content: p.payload?.text ?? '',
    score: p.score,
  }))
}
