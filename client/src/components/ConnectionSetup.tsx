'use client'

import { useState } from 'react'
import { type ConnectionConfig, DEFAULT_LOCAL_PORT } from '@/lib/types'
import { saveConfig } from '@/lib/config'

interface Props {
  initial: ConnectionConfig
  onSave: (config: ConnectionConfig) => void
}

export default function ConnectionSetup({ initial, onSave }: Props) {
  const [cfg, setCfg] = useState<ConnectionConfig>(initial)

  function update(patch: Partial<ConnectionConfig>) {
    setCfg(prev => ({ ...prev, ...patch }))
  }

  function handleSave() {
    saveConfig(cfg)
    onSave(cfg)
  }

  return (
    <div className="min-h-screen flex items-center justify-center p-6">
      <div className="w-full max-w-md bg-gray-900 border border-gray-800 rounded-xl p-8 space-y-6">
        <div>
          <h1 className="text-xl font-semibold text-white">Hytale Code Search</h1>
          <p className="text-sm text-gray-400 mt-1">Choose how to connect to the RAG index.</p>
        </div>

        {/* Mode selection */}
        <fieldset className="space-y-3">
          <legend className="text-xs font-medium text-gray-400 uppercase tracking-wider">
            Connection
          </legend>

          {(
            [
              ['local', 'Local Docker (ChromaDB)', 'Your Docker container on localhost'],
              ['qdrant-default', 'Qdrant Cloud — default', 'Uses pre-configured server credentials'],
              ['qdrant-custom', 'Qdrant Cloud — custom', 'Supply your own cluster URL and API key'],
            ] as const
          ).map(([mode, label, sub]) => (
            <label
              key={mode}
              className={`flex items-start gap-3 p-3 rounded-lg border cursor-pointer transition-colors ${
                cfg.mode === mode
                  ? 'border-blue-500 bg-blue-500/10'
                  : 'border-gray-800 hover:border-gray-700'
              }`}
            >
              <input
                type="radio"
                name="mode"
                value={mode}
                checked={cfg.mode === mode}
                onChange={() => update({ mode })}
                className="mt-0.5 accent-blue-500"
              />
              <div>
                <div className="text-sm font-medium text-white">{label}</div>
                <div className="text-xs text-gray-400">{sub}</div>
              </div>
            </label>
          ))}
        </fieldset>

        {/* Conditional fields */}
        {cfg.mode === 'local' && (
          <div className="space-y-1">
            <label className="text-xs text-gray-400">HTTP port</label>
            <input
              type="number"
              value={cfg.localPort}
              onChange={e => update({ localPort: Number(e.target.value) })}
              className="w-full bg-gray-950 border border-gray-700 rounded-lg px-3 py-2 text-sm text-white focus:outline-none focus:border-blue-500"
              placeholder={String(DEFAULT_LOCAL_PORT)}
            />
            <p className="text-xs text-gray-500">
              Make sure the RAG server is deployed with <code className="bg-gray-800 px-1 rounded">HTTP</code> mode enabled.
            </p>
          </div>
        )}

        {cfg.mode === 'qdrant-default' && (
          <div className="rounded-lg bg-gray-950 border border-gray-800 p-3 text-xs text-gray-400 space-y-1">
            <p>Credentials are read from <code className="bg-gray-800 px-1 rounded">QDRANT_URL</code> and <code className="bg-gray-800 px-1 rounded">QDRANT_API_KEY</code> on the server.</p>
            <p>Set these in your Vercel project → <strong className="text-gray-300">Settings → Environment Variables</strong>.</p>
          </div>
        )}

        {cfg.mode === 'qdrant-custom' && (
          <div className="space-y-3">
            <div className="space-y-1">
              <label className="text-xs text-gray-400">Cluster URL</label>
              <input
                type="url"
                value={cfg.qdrantUrl}
                onChange={e => update({ qdrantUrl: e.target.value })}
                className="w-full bg-gray-950 border border-gray-700 rounded-lg px-3 py-2 text-sm text-white focus:outline-none focus:border-blue-500"
                placeholder="https://xxx.cloud.qdrant.io"
              />
            </div>
            <div className="space-y-1">
              <label className="text-xs text-gray-400">API key</label>
              <input
                type="password"
                value={cfg.qdrantApiKey}
                onChange={e => update({ qdrantApiKey: e.target.value })}
                className="w-full bg-gray-950 border border-gray-700 rounded-lg px-3 py-2 text-sm text-white focus:outline-none focus:border-blue-500"
                placeholder="eyJ..."
              />
            </div>
            <p className="text-xs text-gray-500">
              Credentials are stored in <code className="bg-gray-800 px-1 rounded">localStorage</code> and sent directly from your browser to Qdrant — they never touch the server.
            </p>
          </div>
        )}

        <button
          onClick={handleSave}
          className="w-full bg-blue-600 hover:bg-blue-500 text-white font-medium text-sm py-2.5 rounded-lg transition-colors"
        >
          Save and continue
        </button>
      </div>
    </div>
  )
}
