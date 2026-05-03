'use client'

// Lazily loaded — only initialised when a Qdrant mode is first used.
let _embedder: ((text: string) => Promise<number[]>) | null = null

export async function getEmbedding(text: string): Promise<number[]> {
  if (!_embedder) {
    // Dynamic import keeps this out of the initial bundle.
    const { pipeline, env } = await import('@xenova/transformers')
    // Don't attempt to load local models — always fetch from the HuggingFace CDN.
    env.allowLocalModels = false

    const pipe = await pipeline('feature-extraction', 'Xenova/all-MiniLM-L6-v2', {
      quantized: true,
    })

    _embedder = async (t: string) => {
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      const out = await (pipe as any)(t, { pooling: 'mean', normalize: true })
      return Array.from(out.data as Float32Array)
    }
  }

  return _embedder(text)
}
