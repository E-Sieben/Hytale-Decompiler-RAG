package main

import (
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

//go:embed Dockerfile
var dockerfileContent []byte

//go:embed pyproject.toml
var pyprojectContent []byte

//go:embed mcp_rag.py
var mcpRagContent []byte

const ragDir = ".hytale-rag"

// deployRAG builds, ingests, and starts the RAG container.
// Pass non-empty qdrantURL/qdrantKey to use Qdrant; otherwise uses local ChromaDB.
func deployRAG(qdrantURL, qdrantKey string) {
	if err := os.MkdirAll(ragDir, 0755); err != nil {
		panic(fmt.Sprintf("Failed to create RAG directory: %v", err))
	}

	writeEmbedded := map[string][]byte{
		"Dockerfile":     dockerfileContent,
		"pyproject.toml": pyprojectContent,
		"mcp_rag.py":     mcpRagContent,
	}
	for name, content := range writeEmbedded {
		if err := os.WriteFile(filepath.Join(ragDir, name), content, 0644); err != nil {
			panic(fmt.Sprintf("Failed to write %s: %v", name, err))
		}
	}

	absCodeDir, err := filepath.Abs(CodeDir)
	if err != nil {
		panic(fmt.Sprintf("Failed to resolve code directory path: %v", err))
	}
	absCodeDirFwd := filepath.ToSlash(absCodeDir)

	var composeContent string
	if qdrantURL != "" {
		composeContent = fmt.Sprintf(`services:
  hytale-rag:
    build: .
    container_name: hytale-mcp-container
    environment:
      - RAG_BACKEND=qdrant
      - QDRANT_URL=%s
      - QDRANT_API_KEY=%s
    volumes:
      - %s:/app/hytale_src:ro
      - qdrant_manifest:/app/manifest
    stdin_open: true
    tty: true

volumes:
  qdrant_manifest:
`, qdrantURL, qdrantKey, absCodeDirFwd)
	} else {
		composeContent = fmt.Sprintf(`services:
  hytale-rag:
    build: .
    container_name: hytale-mcp-container
    volumes:
      - %s:/app/hytale_src:ro
      - chroma_data:/app/chroma_db
    stdin_open: true
    tty: true

volumes:
  chroma_data:
`, absCodeDirFwd)
	}

	if err := os.WriteFile(filepath.Join(ragDir, "docker-compose.yml"), []byte(composeContent), 0644); err != nil {
		panic(fmt.Sprintf("Failed to write docker-compose.yml: %v", err))
	}

	fmt.Println("Building Docker image (this may take a few minutes)...")
	run(ragDir, "docker", "compose", "build")

	if qdrantURL != "" {
		fmt.Println("Incrementally ingesting decompiled code into Qdrant...")
	} else {
		fmt.Println("Ingesting decompiled code into ChromaDB...")
	}
	run(ragDir, "docker", "compose", "run", "--rm", "hytale-rag", "uv", "run", "mcp_rag.py", "ingest")

	fmt.Println("Starting MCP server...")
	run(ragDir, "docker", "compose", "up", "-d")

	fmt.Println("\nRAG server is running!")
	printMCPConfig()
}

func run(dir string, name string, args ...string) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		panic(fmt.Sprintf("Command %q failed: %v", name, err))
	}
}

func printMCPConfig() {
	fmt.Println(`
=== MCP Configuration ===
Add the following to your AI tool's MCP server config:

{
  "mcpServers": {
    "HytaleRAG": {
      "command": "docker",
      "args": ["exec", "-i", "hytale-mcp-container", "uv", "run", "mcp_rag.py"]
    }
  }
}`)
}
