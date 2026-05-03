from __future__ import annotations

import os
import sys
import json
import uuid
import hashlib

from fastmcp import FastMCP

RAG_BACKEND = os.getenv("RAG_BACKEND", "chroma")

if RAG_BACKEND == "qdrant":
    from qdrant_client import QdrantClient
    from qdrant_client.http.models import (
        Distance, VectorParams, PointStruct,
        Filter, FieldCondition, MatchValue, PayloadSchemaType,
    )
    from sentence_transformers import SentenceTransformer

    _QDRANT_URL = os.getenv("QDRANT_URL")
    _QDRANT_KEY = os.getenv("QDRANT_API_KEY")
    _COLLECTION = "hytale_codebase"
    _VECTOR_DIM = 384  # all-MiniLM-L6-v2
    _MANIFEST_PATH = "/app/manifest/qdrant_manifest.json"

    _embedder = SentenceTransformer("all-MiniLM-L6-v2")
    _qdrant = QdrantClient(url=_QDRANT_URL, api_key=_QDRANT_KEY)

    if _COLLECTION not in [c.name for c in _qdrant.get_collections().collections]:
        _qdrant.create_collection(
            _COLLECTION,
            vectors_config=VectorParams(size=_VECTOR_DIM, distance=Distance.COSINE),
        )
        _qdrant.create_payload_index(_COLLECTION, "filepath", PayloadSchemaType.KEYWORD)
else:
    import chromadb
    _chroma = chromadb.PersistentClient(path="/app/chroma_db")
    _collection = _chroma.get_or_create_collection(name="hytale_codebase")


mcp = FastMCP("HytaleRAG")


@mcp.tool()
def search_hytale_code(query: str, n_results: int = 5) -> str:
    """Search the decompiled Hytale server codebase for relevant Java context."""
    if RAG_BACKEND == "qdrant":
        embedding = _embedder.encode([query])[0].tolist()
        hits = _qdrant.query_points(_COLLECTION, query=embedding, limit=n_results, with_payload=True).points
        if not hits:
            return "No matching Hytale code found."
        return "".join(
            f"--- File: {h.payload.get('filepath', 'Unknown')} ---\n{h.payload.get('text', '')}\n\n"
            for h in hits
        )
    else:
        results = _collection.query(query_texts=[query], n_results=n_results)
        if not results["documents"] or not results["documents"][0]:
            return "No matching Hytale code found."
        context = ""
        for i, doc in enumerate(results["documents"][0]):
            meta = results["metadatas"][0][i]
            context += f"--- File: {meta.get('filepath', 'Unknown')} ---\n{doc}\n\n"
        return context


def _chunks(text: str) -> list[str]:
    return [text[i:i + 2000] for i in range(0, len(text), 2000)]


def _point_id(filepath: str, content_hash: str, idx: int) -> str:
    return str(uuid.uuid5(uuid.NAMESPACE_URL, f"{filepath}:{content_hash}:{idx}"))


def _filepath_filter(filepath: str) -> Filter:
    return Filter(must=[FieldCondition(key="filepath", match=MatchValue(value=filepath))])


def _ingest_qdrant(directory: str):
    os.makedirs(os.path.dirname(_MANIFEST_PATH), exist_ok=True)
    manifest: dict[str, str] = {}
    if os.path.exists(_MANIFEST_PATH):
        with open(_MANIFEST_PATH) as f:
            manifest = json.load(f)

    disk_files: set[str] = set()
    changed = skipped = 0

    for root, _, files in os.walk(directory):
        for filename in files:
            if not filename.endswith(".java"):
                continue
            filepath = os.path.join(root, filename)
            disk_files.add(filepath)
            try:
                with open(filepath, "r", encoding="utf-8", errors="ignore") as f:
                    text = f.read()
                content_hash = hashlib.md5(text.encode("utf-8")).hexdigest()

                if manifest.get(filepath) == content_hash:
                    skipped += 1
                    continue

                # Delete stale points for this file, then re-insert
                _qdrant.delete(_COLLECTION, points_selector=_filepath_filter(filepath))
                file_chunks = _chunks(text)
                embeddings = _embedder.encode(file_chunks, show_progress_bar=False)
                points = [
                    PointStruct(
                        id=_point_id(filepath, content_hash, i),
                        vector=emb.tolist(),
                        payload={"filename": filename, "filepath": filepath, "text": chunk},
                    )
                    for i, (chunk, emb) in enumerate(zip(file_chunks, embeddings))
                ]
                for i in range(0, len(points), 100):
                    _qdrant.upsert(_COLLECTION, points[i:i + 100])
                manifest[filepath] = content_hash
                changed += 1
            except Exception as e:
                print(f"Skipping {filename}: {e}")

    # Clean up deleted files
    for filepath in list(manifest):
        if filepath not in disk_files:
            _qdrant.delete(_COLLECTION, points_selector=_filepath_filter(filepath))
            del manifest[filepath]
            print(f"Removed deleted file: {os.path.basename(filepath)}")

    with open(_MANIFEST_PATH, "w") as f:
        json.dump(manifest, f)
    print(f"Qdrant ingestion complete — {changed} file(s) updated, {skipped} unchanged.")


def _ingest_chroma(directory: str):
    print(f"Scanning {directory} for .java files...")
    docs, metadatas, ids = [], [], []
    for root, _, files in os.walk(directory):
        for filename in files:
            if not filename.endswith(".java"):
                continue
            filepath = os.path.join(root, filename)
            try:
                with open(filepath, "r", encoding="utf-8", errors="ignore") as f:
                    text = f.read()
                path_hash = hashlib.md5(filepath.encode("utf-8")).hexdigest()[:8]
                for idx, chunk in enumerate(_chunks(text)):
                    docs.append(chunk)
                    metadatas.append({"filename": filename, "filepath": filepath})
                    ids.append(f"{filename}_{path_hash}_{idx}")
            except Exception as e:
                print(f"Skipping {filename}: {e}")

    if not docs:
        print("No Java files found to ingest.")
        return

    print(f"Embedding and ingesting {len(docs)} chunks into ChromaDB...")
    for i in range(0, len(docs), 100):
        _collection.add(
            documents=docs[i:i + 100],
            metadatas=metadatas[i:i + 100],
            ids=ids[i:i + 100],
        )
    print("Ingestion complete! The RAG database is ready.")


def ingest_code(directory: str):
    if RAG_BACKEND == "qdrant":
        _ingest_qdrant(directory)
    else:
        _ingest_chroma(directory)


if __name__ == "__main__":
    if len(sys.argv) > 1 and sys.argv[1] == "ingest":
        ingest_code("/app/hytale_src")
    else:
        mcp.run()
