import os
import sys
import hashlib
import chromadb
from fastmcp import FastMCP

# 1. Initialize Database
db_path = "/app/chroma_db"
client = chromadb.PersistentClient(path=db_path)
collection = client.get_or_create_collection(name="hytale_codebase")

# 2. Initialize MCP Server
mcp = FastMCP("HytaleRAG")

@mcp.tool()
def search_hytale_code(query: str, n_results: int = 5) -> str:
    """Search the decompiled Hytale server codebase for relevant Java context."""
    results = collection.query(query_texts=[query], n_results=n_results)

    if not results['documents'] or not results['documents'][0]:
        return "No matching Hytale code found."

    context = ""
    for i, doc in enumerate(results['documents'][0]):
        meta = results['metadatas'][0][i]
        context += f"--- File: {meta.get('filepath', 'Unknown')} ---\n{doc}\n\n"
    return context

def ingest_code(directory: str):
    """Reads, chunks, and embeds the decompiled .java files."""
    print(f"Scanning {directory} for .java files...")
    docs, metadatas, ids = [], [], []

    for root, _, files in os.walk(directory):
        for file in files:
            if file.endswith('.java'):
                filepath = os.path.join(root, file)
                try:
                    with open(filepath, 'r', encoding='utf-8', errors='ignore') as f:
                        text = f.read()

                        # Generate a unique short hash of the filepath to prevent ID collisions
                        path_hash = hashlib.md5(filepath.encode('utf-8')).hexdigest()[:8]

                        # Simple chunking (approx 2000 chars per chunk)
                        chunks = [text[i:i+2000] for i in range(0, len(text), 2000)]
                        for idx, chunk in enumerate(chunks):
                            docs.append(chunk)
                            metadatas.append({"filename": file, "filepath": filepath})
                            # Now the ID is guaranteed unique: e.g., ServerCookieEncoder.java_a1b2c3d4_0
                            ids.append(f"{file}_{path_hash}_{idx}")
                except Exception as e:
                    print(f"Skipping {file}: {e}")

    if not docs:
        print("No Java files found to ingest.")
        return

    print(f"Embedding and ingesting {len(docs)} chunks into ChromaDB... (This may take a minute)")
    batch_size = 100
    for i in range(0, len(docs), batch_size):
        collection.add(
            documents=docs[i:i+batch_size],
            metadatas=metadatas[i:i+batch_size],
            ids=ids[i:i+batch_size]
        )
    print("Ingestion complete! The RAG database is ready.")

if __name__ == "__main__":
    if len(sys.argv) > 1 and sys.argv[1] == "ingest":
        ingest_code("/app/hytale_src")
    else:
        # Run as standard MCP stdio server
        mcp.run()