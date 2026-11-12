# ---- build ----
FROM python:3.13@sha256:ecbe213fb818d557fd19db4078244339e2e2e52874f9c81052d3094ea114d797 AS build
COPY --from=ghcr.io/astral-sh/uv:0.13.1@sha256:4a1f3666816dafadd20ceb71eac9a7d56d7ef30279f7d64048519ab901ada336 /uv /usr/local/bin/uv
WORKDIR /app
ENV UV_COMPILE_BYTECODE=1 UV_LINK_MODE=copy UV_PYTHON=/usr/local/bin/python3.13 UV_PYTHON_DOWNLOADS=never
COPY pyproject.toml uv.lock ./
RUN uv sync --frozen --no-install-project --no-dev
COPY . .
RUN uv sync --frozen --no-dev

# ---- runtime ----
# Slim drops the full image's dev/kernel/image libs, which only ever brought CVEs.
# Same 3.13 build as the build stage, so the venv's interpreter path still resolves.
FROM python:3.15-slim@sha256:65a65fc7b30b06cb6df4930fd4650697cb23f4c2698ab81fc87e4b031bc36928
WORKDIR /app
COPY --from=build /app /app
ENV PATH="/app/.venv/bin:$PATH"
# The venv is uv-managed, so the base image's pip (and the deps it vendors) is dead weight for scanners
RUN rm -rf /usr/local/lib/python3.13/site-packages/pip /usr/local/lib/python3.13/site-packages/pip-*.dist-info \
    /usr/local/lib/python3.13/ensurepip/_bundled /usr/local/bin/pip* \
 && useradd -m app && chown -R app /app
USER app
EXPOSE 8080
CMD ["sh", "-c", "uvicorn src.server:app --host 0.0.0.0 --port ${PORT:-8080}"]
