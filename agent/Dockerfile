# ---- build ----
FROM python@sha256:297cf11d0b98b38ac26a56136f0279df845314bcd0347c1f6383fee6e75125ee AS build
COPY --from=ghcr.io/astral-sh/uv@sha256:3d868e555f8f1dbc324afa005066cd11e1053fc4743b9808ca8025283e65efa5 /uv /usr/local/bin/uv
WORKDIR /app
ENV UV_COMPILE_BYTECODE=1 UV_LINK_MODE=copy
COPY pyproject.toml uv.lock ./
RUN uv sync --frozen --no-install-project --no-dev
COPY . .
RUN uv sync --frozen --no-dev

# ---- runtime ----
FROM python@sha256:297cf11d0b98b38ac26a56136f0279df845314bcd0347c1f6383fee6e75125ee
WORKDIR /app
COPY --from=build /app /app
ENV PATH="/app/.venv/bin:$PATH"
RUN useradd -m app && chown -R app /app
USER app
EXPOSE 8080
CMD ["sh", "-c", "uvicorn src.server:app --host 0.0.0.0 --port ${PORT:-8080}"]
