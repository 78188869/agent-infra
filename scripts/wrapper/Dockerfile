# Stage 1: Python dependencies
FROM python:3.12-slim AS py-builder
WORKDIR /build
COPY requirements.txt .
RUN pip install --no-cache-dir --prefix=/install -r requirements.txt

# Stage 2: Node.js + Claude Code CLI
FROM node:20-slim AS node-builder
RUN npm install -g @anthropic-ai/claude-code

# Stage 3: Runtime
FROM python:3.12-slim

# Dev toolchain
RUN apt-get update && apt-get install -y --no-install-recommends \
    git curl wget make gcc bash \
    && rm -rf /var/lib/apt/lists/*

# Go
RUN ARCH=$(dpkg --print-architecture) && \
    curl -sL "https://go.dev/dl/go1.22.0.linux-${ARCH}.tar.gz" | tar -C /usr/local -xz
ENV PATH="/usr/local/go/bin:${PATH}"

# GitHub CLI
RUN curl -fsSL https://cli.github.com/packages/githubcli-archive-keyring.gpg \
    | dd of=/usr/share/keyrings/githubcli-archive-keyring.gpg \
    && echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/githubcli-archive-keyring.gpg] https://cli.github.com/packages stable main" \
    | tee /etc/apt/sources.list.d/github-cli.list > /dev/null \
    && apt-get update && apt-get install -y --no-install-recommends gh \
    && rm -rf /var/lib/apt/lists/*

# Node.js + Claude Code CLI from builder
COPY --from=node-builder /usr/local/bin/claude /usr/local/bin/
COPY --from=node-builder /usr/local/lib/node_modules /usr/local/lib/node_modules

# Python dependencies from builder
COPY --from=py-builder /install /usr/local

# Wrapper application
WORKDIR /app
COPY scripts/wrapper/ .

# Entrypoint script
COPY scripts/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

EXPOSE 9090
HEALTHCHECK --interval=10s --timeout=3s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:9090/health || exit 1

ENTRYPOINT ["/entrypoint.sh"]
