# Use the official uv image based on Debian slim
FROM ghcr.io/astral-sh/uv:python3.11-bookworm-slim

WORKDIR /app

# Enable bytecode compilation for faster startup
ENV UV_COMPILE_BYTECODE=1

# Copy dependency definition and install them first for Docker layer caching
COPY pyproject.toml .
RUN uv sync --no-dev --no-install-project

# Copy your source code
COPY mcp_rag.py .

# uv run automatically detects and uses the .venv created by uv sync
CMD ["uv", "run", "mcp_rag.py"]