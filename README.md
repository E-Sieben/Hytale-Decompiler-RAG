# Hytale decompiler with RAG support

This projects provides you with a .exe(Windows) or binary(Linux) to Auto-Download and Decompile the Hytale Server Jar.

# Prerequisites

- [Java 25](https://adoptium.net/temurin/releases)
- [Docker](https://docs.docker.com/get-started/get-docker/) (Not required for decompiled code only)

# Setup / Update

1. Download the latest Version for you operating system
2. Execute your .exe or binary
3. If this is your first time executing the Program you will have to authenticate yourself with Hytale
4. Select `Decompiled code only` or `Setup RAG`
5. If you selected `Setup RAG` you can now add the [JSON Config](#MCP-JSON) to your AI of choosing

# MCP-JSON

// TODO : Add JSON MCP Config

```JSON

```

# What the program does

1. Check dependencies (Docker(No Auto-Install), Java 25(No Auto-Install),
   [Hytale Server Downloader](https://downloader.hytale.com/hytale-downloader.zip),
   [Vineflower](https://github.com/Vineflower/vineflower/releases/latest)
2. Replace or Create HytaleServer.jar via Hytale Server downloader
3. Decompile HytaleServer.jar using VineFlower
4. Collapse Hierarchy (Delete everything except com.hypixel.hytale)
5. Clean Up decompiled code.
6. Zip up source code and put it accessible somewhere
7. Spin up Docker (Reference the existing code)

# WARNING
This is objectively speaking AI-slop-ware