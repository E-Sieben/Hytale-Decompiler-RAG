import { type NextRequest, NextResponse } from 'next/server'

const QDRANT_URL = process.env.QDRANT_URL
const QDRANT_API_KEY = process.env.QDRANT_API_KEY
const COLLECTION = 'hytale_codebase'

export async function POST(req: NextRequest) {
  if (!QDRANT_URL || !QDRANT_API_KEY) {
    return NextResponse.json(
      {
        error:
          'Qdrant credentials are not configured. ' +
          'Set QDRANT_URL and QDRANT_API_KEY in your Vercel environment variables.',
      },
      { status: 503 },
    )
  }

  const { vector, n = 5 } = (await req.json()) as { vector: number[]; n?: number }

  const qdrantRes = await fetch(
    `${QDRANT_URL}/collections/${COLLECTION}/points/search`,
    {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'api-key': QDRANT_API_KEY,
      },
      body: JSON.stringify({ vector, limit: n, with_payload: true }),
    },
  )

  if (!qdrantRes.ok) {
    const text = await qdrantRes.text()
    return NextResponse.json({ error: `Qdrant error: ${text}` }, { status: qdrantRes.status })
  }

  const { result } = await qdrantRes.json()
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const results = result.map((p: any) => ({
    filepath: p.payload?.filepath ?? '',
    content: p.payload?.text ?? '',
    score: p.score,
  }))

  return NextResponse.json({ results })
}
