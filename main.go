package main

func main() {
	wantsRAG := true // Ask the User if they want to deploy RAG aswell
	status := NewDependencyStatus()
	DownloadMissingDependencies(*status)
	decompile()
	flatten()
	strip()
	if status.HasAllDependencies() && wantsRAG {
		// TODO: Use go:embed to write out the Dockerfile, docker-compose.yml, pyproject.toml,
		// and mcp_rag.py to a hidden/local directory, then triggers docker-compose to build and start the RAG server.
		deployRAG()
	}
}
