import type { Metadata } from 'next'
import './globals.css'

export const metadata: Metadata = {
  title: 'Hytale Code Search',
  description: 'Search the decompiled Hytale server source using RAG + WebLLM',
}

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en">
      <body className="min-h-screen">{children}</body>
    </html>
  )
}
