# Hytale decompiler with RAG support

This project provides you with a .exe (Windows) or binary (Linux) to Auto-Download and Decompile the Hytale Server Jar,
with Javadocs. It also provides you with the ability to add RAG support to your AI.

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

This is objectively speaking AI-slopware
