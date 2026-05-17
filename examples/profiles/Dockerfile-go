FROM golang:1.24-bookworm

ARG OPENCODE_VERSION=latest

# Runtime tools: curl/unzip for the opencode installer, git for SCM.
RUN apt-get update && \
    apt-get install -y --no-install-recommends \
        curl \
        unzip \
        ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# Install opencode.
RUN curl -fsSL https://opencode.ai/install | bash
ENV PATH="/root/.local/bin:$PATH"

EXPOSE 4096
WORKDIR /workspace
