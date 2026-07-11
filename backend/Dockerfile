# ============================================================================
# Stage 1: Builder — install dependencies with uv
# ============================================================================
FROM ghcr.io/astral-sh/uv:python3.12-bookworm-slim AS builder

WORKDIR /app

# Install dependencies first (layer caching)
COPY pyproject.toml uv.lock ./
RUN uv sync --frozen --no-dev --no-install-project

# Copy application code
COPY main.py ./

# ============================================================================
# Stage 2: Runtime — distroless (no shell, non-root)
# ============================================================================
FROM gcr.io/distroless/python3-debian12

WORKDIR /app

# Copy the virtual environment and application from builder
COPY --from=builder /app/.venv/lib/python3.12/site-packages /app/site-packages
COPY --from=builder /app/main.py /app/main.py

ENV PYTHONPATH=/app/site-packages
ENV PORT=8000

EXPOSE 8000

USER nonroot

ENTRYPOINT ["python", "/app/main.py"]
