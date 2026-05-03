'use client'

import { useEffect, useRef, useState, useCallback } from 'react'
import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import type { ConnectionConfig, Message, RagResult } from '@/lib/types'
import { AVAILABLE_MODELS } from '@/lib/types'
import { loadModel, saveModel } from '@/lib/config'
import { search } from '@/lib/rag'

// ---------------------------------------------------------------------------
// System prompt — navigator mode, no code generation, prefer Hytale built-ins
// ---------------------------------------------------------------------------
const SYSTEM_PROMPT = `You are a Hytale server codebase navigator. Your job is to help \
developers find the right existing classes and patterns in the decompiled source — not to \
generate new code.

Rules:
- Never write or suggest new code. Only reference classes and files that exist in the provided context.
- Always prefer built-in Hytale abstractions. The codebase already has purpose-built solutions: \
CommandBase for commands, BuilderCodec for serialisation, existing event classes for game events, \
packet handlers, registries, etc. Point to these instead of external libraries or custom implementations.
- For each relevant finding state the full file path, the class or interface name, and what it does \
in one sentence. Then explain how it applies to the question.
- If the search context does not contain what is needed, say so clearly and suggest a more precise \
search term the user can try.
- Keep answers short — two to four sentences is usually enough.`

// ---------------------------------------------------------------------------
// Query reformulation — extract precise class/method terms before hitting RAG
// ---------------------------------------------------------------------------
// eslint-disable-next-line @typescript-eslint/no-explicit-any
async function reformulateForSearch(question: string, engine: any): Promise<string> {
  try {
    const res = await engine.chat.completions.create({
      messages: [
        {
          role: 'system',
          content:
            'You are a search query extractor for the Hytale server Java codebase. ' +
            'Given a developer question, output ONLY specific Java class names, interface names, ' +
            'method names, or subsystem keywords that would appear in the relevant source files. ' +
            'One line, comma-separated, nothing else.',
        },
        { role: 'user', content: question },
      ],
      temperature: 0,
      max_tokens: 40,
      stream: false,
    })
    const terms = res.choices[0]?.message?.content?.trim()
    return terms && terms.length > 0 ? terms : question
  } catch {
    return question
  }
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------
function modeLabel(config: ConnectionConfig) {
  if (config.mode === 'local') return `Local :${config.localPort}`
  if (config.mode === 'qdrant-default') return 'Qdrant (default)'
  return 'Qdrant (custom)'
}

type ModelStatus = 'idle' | 'loading' | 'ready' | 'error'

interface Props {
  config: ConnectionConfig
  onOpenSettings: () => void
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------
export default function ChatWindow({ config, onOpenSettings }: Props) {
  const [messages, setMessages] = useState<Message[]>([])
  const [input, setInput] = useState('')
  const [modelStatus, setModelStatus] = useState<ModelStatus>('idle')
  const [loadProgress, setLoadProgress] = useState(0)
  const [loadText, setLoadText] = useState('')
  const [isGenerating, setIsGenerating] = useState(false)
  const [isSearching, setIsSearching] = useState(false)
  const [selectedModel, setSelectedModel] = useState<string>(loadModel)
  const [expandedSources, setExpandedSources] = useState<Set<string>>(new Set())

  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const engineRef = useRef<any>(null)
  const bottomRef = useRef<HTMLDivElement>(null)

  // Load WebLLM engine
  useEffect(() => {
    let cancelled = false
    setModelStatus('loading')
    setLoadProgress(0)
    engineRef.current = null

    import('@mlc-ai/web-llm')
      .then(({ CreateMLCEngine }) =>
        CreateMLCEngine(selectedModel, {
          initProgressCallback: ({ progress, text }: { progress: number; text: string }) => {
            if (!cancelled) {
              setLoadProgress(Math.round(progress * 100))
              setLoadText(text)
            }
          },
        }),
      )
      .then(engine => {
        if (!cancelled) {
          engineRef.current = engine
          setModelStatus('ready')
        }
      })
      .catch(() => { if (!cancelled) setModelStatus('error') })

    return () => { cancelled = true }
  }, [selectedModel])

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages, isSearching])

  const handleModelChange = useCallback((modelId: string) => {
    saveModel(modelId)
    setSelectedModel(modelId)
    setMessages([])
  }, [])

  const handleSend = useCallback(async () => {
    const text = input.trim()
    if (!text || isGenerating || modelStatus !== 'ready') return

    setInput('')
    setMessages(prev => [...prev, { id: crypto.randomUUID(), role: 'user', content: text }])

    // Reformulate → search (both covered by the "Searching…" indicator)
    setIsSearching(true)
    let sources: RagResult[] = []
    try {
      const searchQuery = await reformulateForSearch(text, engineRef.current)
      sources = await search(searchQuery, config)
    } catch (e) {
      console.error('RAG search failed:', e)
    } finally {
      setIsSearching(false)
    }

    const contextBlock = sources
      .map(r => `File: ${r.filepath}\n\`\`\`java\n${r.content}\n\`\`\``)
      .join('\n\n')
    const userContent = contextBlock
      ? `Relevant source files:\n\n${contextBlock}\n\n---\n\nQuestion: ${text}`
      : text

    const assistantId = crypto.randomUUID()
    setMessages(prev => [...prev, { id: assistantId, role: 'assistant', content: '', sources }])

    setIsGenerating(true)
    try {
      const history = messages.slice(-8).map(m => ({ role: m.role, content: m.content }))
      const stream = await engineRef.current.chat.completions.create({
        messages: [
          { role: 'system', content: SYSTEM_PROMPT },
          ...history,
          { role: 'user', content: userContent },
        ],
        stream: true,
        temperature: 0.2,
      })
      for await (const chunk of stream) {
        const delta = chunk.choices[0]?.delta?.content ?? ''
        if (delta) {
          setMessages(prev => {
            const last = prev[prev.length - 1]
            return [...prev.slice(0, -1), { ...last, content: last.content + delta }]
          })
        }
      }
    } catch {
      setMessages(prev => {
        const last = prev[prev.length - 1]
        return [...prev.slice(0, -1), { ...last, content: '_Error generating response._' }]
      })
    } finally {
      setIsGenerating(false)
    }
  }, [input, isGenerating, modelStatus, config, messages])

  function handleKeyDown(e: React.KeyboardEvent<HTMLTextAreaElement>) {
    if (e.key === 'Enter' && !e.shiftKey) { e.preventDefault(); handleSend() }
  }

  function toggleSources(id: string) {
    setExpandedSources(prev => {
      const next = new Set(prev)
      next.has(id) ? next.delete(id) : next.add(id)
      return next
    })
  }

  return (
    <div className="flex flex-col h-screen">
      {/* Header */}
      <header className="flex items-center justify-between px-4 py-3 border-b border-gray-800 bg-gray-900/50 backdrop-blur shrink-0">
        <div className="flex items-center gap-3">
          <span className="text-white font-semibold text-sm">Hytale Code Search</span>
          <span className="text-xs px-2 py-0.5 rounded-full bg-gray-800 text-gray-400">
            {modeLabel(config)}
          </span>
        </div>
        <div className="flex items-center gap-2">
          <select
            value={selectedModel}
            onChange={e => handleModelChange(e.target.value)}
            className="text-xs bg-gray-800 border border-gray-700 text-gray-300 rounded-lg px-2 py-1 focus:outline-none"
          >
            {AVAILABLE_MODELS.map(m => (
              <option key={m.id} value={m.id}>{m.label}</option>
            ))}
          </select>
          <button
            onClick={onOpenSettings}
            className="text-xs text-gray-400 hover:text-white px-2 py-1 rounded-lg hover:bg-gray-800 transition-colors"
          >
            Settings
          </button>
        </div>
      </header>

      {/* Model loading */}
      {modelStatus !== 'ready' && (
        <div className="flex-1 flex flex-col items-center justify-center gap-4 p-8">
          {modelStatus === 'loading' && (
            <>
              <p className="text-sm text-gray-400">{loadText || 'Initializing model…'}</p>
              <div className="w-72 bg-gray-800 rounded-full h-2">
                <div
                  className="bg-blue-500 h-2 rounded-full transition-all duration-300"
                  style={{ width: `${loadProgress}%` }}
                />
              </div>
              <p className="text-xs text-gray-500">{loadProgress}%</p>
            </>
          )}
          {modelStatus === 'error' && (
            <p className="text-sm text-red-400">
              Failed to load model. WebGPU may not be supported in this browser.
            </p>
          )}
        </div>
      )}

      {/* Messages */}
      {modelStatus === 'ready' && (
        <div className="flex-1 overflow-y-auto px-4 py-6 space-y-6 min-w-0">
          {messages.length === 0 && (
            <div className="text-center text-gray-500 text-sm mt-20">
              <p className="text-base text-gray-400 mb-2">Ask anything about the Hytale server source.</p>
              <p>e.g. "How are commands registered?" or "Where are packets handled?"</p>
            </div>
          )}

          {messages.map(msg => (
            <div key={msg.id} className={`flex min-w-0 ${msg.role === 'user' ? 'justify-end' : 'justify-start'}`}>
              <div className={`min-w-0 max-w-2xl w-full ${msg.role === 'user' ? 'pl-12' : 'pr-12'}`}>
                <div
                  className={`rounded-xl px-4 py-3 text-sm leading-relaxed min-w-0 ${
                    msg.role === 'user'
                      ? 'bg-blue-600 text-white whitespace-pre-wrap break-words'
                      : 'bg-gray-900 border border-gray-800 text-gray-100'
                  }`}
                >
                  {msg.role === 'user' ? (
                    msg.content
                  ) : msg.content ? (
                    <ReactMarkdown
                      remarkPlugins={[remarkGfm]}
                      components={{
                        // Inline code
                        code: ({ className, children, ...props }) => {
                          const isBlock = className?.includes('language-')
                          return isBlock ? (
                            <code className={`${className ?? ''} block`} {...props}>{children}</code>
                          ) : (
                            <code className="bg-gray-800 text-blue-300 rounded px-1 py-0.5 text-xs font-mono" {...props}>
                              {children}
                            </code>
                          )
                        },
                        // Fenced code blocks
                        pre: ({ children }) => (
                          <pre className="bg-gray-950 border border-gray-800 rounded-lg p-3 text-xs font-mono overflow-x-auto my-2 max-w-full">
                            {children}
                          </pre>
                        ),
                        // Paragraphs — prevent margin collapse at top of bubble
                        p: ({ children }) => <p className="mb-2 last:mb-0 break-words">{children}</p>,
                        // Links
                        a: ({ children, href }) => (
                          <a href={href} className="text-blue-400 underline" target="_blank" rel="noreferrer">
                            {children}
                          </a>
                        ),
                        // Lists
                        ul: ({ children }) => <ul className="list-disc list-inside space-y-0.5 mb-2">{children}</ul>,
                        ol: ({ children }) => <ol className="list-decimal list-inside space-y-0.5 mb-2">{children}</ol>,
                      }}
                    >
                      {msg.content}
                    </ReactMarkdown>
                  ) : (
                    <span className="animate-pulse text-gray-500">▍</span>
                  )}
                </div>

                {/* Sources */}
                {msg.role === 'assistant' && msg.sources && msg.sources.length > 0 && (
                  <div className="mt-1">
                    <button
                      onClick={() => toggleSources(msg.id)}
                      className="text-xs text-gray-500 hover:text-gray-300 transition-colors"
                    >
                      {expandedSources.has(msg.id) ? '▾' : '▸'}{' '}
                      {msg.sources.length} source{msg.sources.length !== 1 ? 's' : ''}
                    </button>
                    {expandedSources.has(msg.id) && (
                      <div className="mt-2 space-y-2">
                        {msg.sources.map((src, i) => (
                          <div key={i} className="bg-gray-950 border border-gray-800 rounded-lg p-3 min-w-0">
                            <p className="text-xs text-blue-400 font-mono mb-2 truncate">{src.filepath}</p>
                            <pre className="text-xs text-gray-400 font-mono leading-relaxed max-h-48 overflow-y-auto overflow-x-auto whitespace-pre">
                              {src.content}
                            </pre>
                          </div>
                        ))}
                      </div>
                    )}
                  </div>
                )}
              </div>
            </div>
          ))}

          {isSearching && (
            <div className="flex justify-start">
              <div className="bg-gray-900 border border-gray-800 rounded-xl px-4 py-3 text-xs text-gray-500 animate-pulse">
                Searching codebase…
              </div>
            </div>
          )}

          <div ref={bottomRef} />
        </div>
      )}

      {/* Input */}
      {modelStatus === 'ready' && (
        <div className="px-4 py-4 border-t border-gray-800 bg-gray-900/30 shrink-0">
          <div className="flex items-end gap-2 bg-gray-900 border border-gray-700 rounded-xl px-4 py-3 focus-within:border-blue-500 transition-colors">
            <textarea
              value={input}
              onChange={e => setInput(e.target.value)}
              onKeyDown={handleKeyDown}
              placeholder="Ask about the Hytale codebase…"
              rows={1}
              className="flex-1 bg-transparent text-sm text-white placeholder-gray-500 resize-none focus:outline-none max-h-40"
              style={{ fieldSizing: 'content' } as React.CSSProperties}
              disabled={isGenerating}
            />
            <button
              onClick={handleSend}
              disabled={!input.trim() || isGenerating}
              className="text-sm px-3 py-1.5 bg-blue-600 hover:bg-blue-500 disabled:opacity-40 disabled:cursor-not-allowed text-white rounded-lg transition-colors shrink-0"
            >
              {isGenerating ? '…' : 'Send'}
            </button>
          </div>
          <p className="text-xs text-gray-600 mt-1.5 text-center">Enter to send · Shift+Enter for newline</p>
        </div>
      )}
    </div>
  )
}
