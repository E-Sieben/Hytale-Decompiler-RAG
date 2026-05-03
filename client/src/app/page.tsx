'use client'

import { useState } from 'react'
import type { ConnectionConfig } from '@/lib/types'
import { loadConfig } from '@/lib/config'
import ConnectionSetup from '@/components/ConnectionSetup'
import ChatWindow from '@/components/ChatWindow'

export default function Home() {
  const [config, setConfig] = useState<ConnectionConfig | null>(() => {
    // null = show setup screen on first visit (no saved config)
    const saved = loadConfig()
    // Consider 'qdrant-default' always ready to go; others need user confirmation
    return saved.mode === 'qdrant-default' ? saved : saved
  })
  const [showSetup, setShowSetup] = useState(() => {
    if (typeof window === 'undefined') return false
    return !localStorage.getItem('hytale-rag-config')
  })

  if (showSetup || !config) {
    return (
      <ConnectionSetup
        initial={config ?? { mode: 'qdrant-default', localPort: 8080, qdrantUrl: '', qdrantApiKey: '' }}
        onSave={cfg => {
          setConfig(cfg)
          setShowSetup(false)
        }}
      />
    )
  }

  return (
    <ChatWindow
      config={config}
      onOpenSettings={() => setShowSetup(true)}
    />
  )
}
