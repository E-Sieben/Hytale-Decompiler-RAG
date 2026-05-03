import type { NextConfig } from 'next'

const config: NextConfig = {
  // SharedArrayBuffer (required by WebLLM for threading) needs these two headers together.
  async headers() {
    return [
      {
        source: '/(.*)',
        headers: [
          { key: 'Cross-Origin-Opener-Policy', value: 'same-origin' },
          // 'credentialless' (not 'require-corp') lets us still fetch cross-origin
          // APIs (Qdrant REST) without CORP headers while keeping SharedArrayBuffer.
          { key: 'Cross-Origin-Embedder-Policy', value: 'credentialless' },
        ],
      },
    ]
  },

  webpack(config, { isServer }) {
    // Prevent onnxruntime-node (server-only) from being bundled for the browser.
    if (!isServer) {
      config.resolve.alias = {
        ...config.resolve.alias,
        'sharp$': false,
        'onnxruntime-node$': false,
      }
    }
    // Enable async WASM imports used by @xenova/transformers and WebLLM.
    config.experiments = { ...config.experiments, asyncWebAssembly: true }
    return config
  },
}

export default config
