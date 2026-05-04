# Hytale decompiler with RAG support

This project provides the following Hytale Server code features:
- Code decompilation with JavaDocs
- Web search Engine
- Local RAG MCP
- Qdrant RAG MCP

# Easy Usage

Visit [this link](https://hytale-decompiler-rag.vercel.app/) for access to a public "search engine"

This runs a tiny model in your browser providing (More flexability -> Less Power), especially good for folks with weak machines

# Prerequisites

- Eight Gigs of RAM
- [Java 25](https://adoptium.net/temurin/releases)
- [Docker](https://docs.docker.com/get-started/get-docker/) (Not required for decompiled code only)

# Setup / Update

1. Download the latest Version for your operating system (or compile yourself using `go build`)
2. Execute your .exe or binary
3. If this is your first time executing the Program, you will have to authenticate yourself with Hytale
4. Decide whether you want to `Setup RAG` as well
5. Decide if you want to use the `pre-release` or `release` channel
6. Wait a bit decompilation takes a while
7. If you said yes to `Setup RAG` you can now add the [JSON Config](#MCP-JSON) to your AI of choosing

# CLI Flags

All flags are optional. When a flag is omitted the program asks interactively instead.

| Flag                      | Description                                              |
|---------------------------|----------------------------------------------------------|
| `--rag`                   | Set up the RAG server (requires Docker)                  |
| `--no-rag`                | Skip RAG server setup                                    |
| `--prerelease`            | Use the pre-release channel                              |
| `--no-prerelease`         | Use the stable release channel                           |
| `--redownload-jar`        | Re-download `HytaleServer.jar` even if it already exists |
| `--no-redownload-jar`     | Keep the existing `HytaleServer.jar` without prompting   |
| `--qdrant`                | Use Qdrant cloud instead of local ChromaDB               |
| `--no-qdrant`             | Use local ChromaDB (skip Qdrant)                         |
| `--qdrant-endpoint <url>` | Qdrant cloud endpoint URL (implies `--qdrant`)           |
| `--qdrant-key <key>`      | Qdrant API key (implies `--qdrant`)                      |

`--qdrant` and `--no-qdrant` are only relevant when `--rag` is set.

Credentials passed via `--qdrant-endpoint` / `--qdrant-key` are automatically saved to `credentials.json`
next to the binary so you do not need to supply them again on the next run.

**Fully non-interactive example:**

```sh
./Hytale-Decompiler-RAG --rag --no-prerelease --no-redownload-jar \
  --qdrant-endpoint https://your-cluster.qdrant.io \
  --qdrant-key YOUR_API_KEY
```

# MCP-JSON
For local Docker usage
```JSON
{
  "mcpServers": {
    "HytaleRAG": {
      "command": "docker",
      "args": [
        "exec",
        "-i",
        "hytale-mcp-container",
        "uv",
        "run",
        "mcp_rag.py"
      ]
    }
  }
}
```
For Qdrant-Usage
```JSON
{
  "mcpServers": {
    "HytaleRAG": {
      "command": "your\\path\\to\\uv.exe",
      "args": [
        "run",
        "--project",
        "your\\path\\to\\main",
        "python",
        "your\\path\\to\\mcp_rag.py"
      ],
      "cwd": "path\\to\\you\\work\\directory",
      "env": {
        "RAG_BACKEND": "qdrant",
        "QDRANT_URL": "your_endpoint_url.cloud.qdrant.io",
        "QDRANT_API_KEY": "your_api_key"
      }
    }
  }
}
```

# What the program does

1. Check dependencies (Docker (No Auto-Install), Java 25 (No Auto-Install),
   [Hytale Server Downloader](https://downloader.hytale.com/hytale-downloader.zip),
   [Vineflower](https://github.com/Vineflower/vineflower/releases/latest)
2. Replace or Create HytaleServer.jar via Hytale Server downloader
3. Decompile HytaleServer.jar using VineFlower
4. Collapse Hierarchy (Delete everything except com.hypixel.hytale)
5. Clean Up decompiled code.
6. Add in Javadocs
7. Zip up source code and put it accessible somewhere
8. Spin up Docker (Reference the existing code)

# WARNING

- This Software is objectively speaking AI-slopware
- This Software does not provide any safety, except for vercel and qdrant being it's target cloud architecture
- This Software is intended for local development and not for unsupervised AI Code runners
- This Software is best used as a Search Engine, AI Code generation is prone to hallucination and over-personalisation of systems
