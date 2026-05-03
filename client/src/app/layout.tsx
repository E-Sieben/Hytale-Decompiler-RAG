import type { Metadata } from 'next'
import './globals.css'

export const metadata: Metadata = {
  title: 'Hytale Code Search',
  description: 'Search the decompiled Hytale server source using RAG + WebLLM',
  verification: {
    google: 'fq9YoKzQHYIxgwpgQLx7psO6yDkw7UP4Ljof7BCfbOI',
  },
}

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en">
      <body className="min-h-screen">{children}</body>
    </html>
  )
}
