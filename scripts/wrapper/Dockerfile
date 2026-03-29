# Stage 1: Python dependencies (production only)
FROM python:3.12-slim AS py-builder
WORKDIR /build
COPY scripts/wrapper/requirements.txt .
RUN pip install --no-cache-dir --prefix=/install -r requirements.txt

# Stage 2: Node.js + Claude Code CLI
FROM node:20-slim AS node-builder
RUN npm install -g @anthropic-ai/claude-code

# Stage 3: Runtime
FROM python:3.12-slim

ENV DEBIAN_FRONTEND=noninteractive

# Dev toolchain (gcc+g++ instead of build-essential to avoid bloat)
RUN apt-get update && apt-get install -y --no-install-recommends \
    git curl wget make gcc g++ bash openssh-client ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# Go (parameterized version)
ARG GO_VERSION=1.22.0
RUN ARCH=$(dpkg --print-architecture) && \
    curl -fSL --retry 3 --retry-delay 5 "https://go.dev/dl/go${GO_VERSION}.linux-${ARCH}.tar.gz" -o /tmp/go.tar.gz && \
    tar -C /usr/local -xzf /tmp/go.tar.gz && \
    rm /tmp/go.tar.gz
ENV PATH="/usr/local/go/bin:${PATH}"

# GitHub CLI
RUN curl -fsSL https://cli.github.com/packages/githubcli-archive-keyring.gpg \
    | dd of=/usr/share/keyrings/githubcli-archive-keyring.gpg \
    && echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/githubcli-archive-keyring.gpg] https://cli.github.com/packages stable main" \
    | tee /etc/apt/sources.list.d/github-cli.list > /dev/null \
    && apt-get update && apt-get install -y --no-install-recommends gh \
    && rm -rf /var/lib/apt/lists/*

# Node.js toolchain + Claude Code CLI from builder
COPY --from=node-builder /usr/local/bin/node /usr/local/bin/
COPY --from=node-builder /usr/local/bin/npm /usr/local/bin/
COPY --from=node-builder /usr/local/bin/npx /usr/local/bin/
COPY --from=node-builder /usr/local/bin/claude /usr/local/bin/
COPY --from=node-builder /usr/local/lib/node_modules /usr/local/lib/node_modules

# Python dependencies from builder
COPY --from=py-builder /install /usr/local

# Create non-root user (matches K8s SecurityConfig RunAsUser: 1000)
RUN useradd -m -u 1000 -s /bin/bash runner

# Workspace directory
RUN mkdir -p /workspace && chown runner:runner /workspace

# Wrapper application
WORKDIR /app
COPY scripts/wrapper/ .
RUN chown -R runner:runner /app

# Entrypoint script
COPY scripts/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh && chown runner:runner /entrypoint.sh

# Switch to non-root user
USER runner

EXPOSE 9090

HEALTHCHECK --interval=10s --timeout=3s --start-period=60s --retries=3 \
    CMD curl -f http://localhost:9090/health || exit 1

ENTRYPOINT ["/entrypoint.sh"]
