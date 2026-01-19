# Enclaude - Ubuntu devcontainer for Claude Code
FROM ubuntu:24.04

# Labels
LABEL org.opencontainers.image.title="enclaude"
LABEL org.opencontainers.image.description="Ubuntu devcontainer for Claude Code"

# Avoid prompts during package installation
ENV DEBIAN_FRONTEND=noninteractive

# Install common development utilities
RUN apt-get update && apt-get install -y --no-install-recommends \
    # Core utilities
    ca-certificates \
    curl \
    wget \
    git \
    jq \
    less \
    tree \
    zip \
    unzip \
    openssh-client \
    # Editors
    vim \
    nano \
    # Build tools
    build-essential \
    pkg-config \
    # Python
    python3 \
    python3-pip \
    python3-venv \
    # Search tools
    ripgrep \
    fd-find \
    # Process tools
    htop \
    && rm -rf /var/lib/apt/lists/* \
    && apt-get clean

# Install Node.js LTS via NodeSource
RUN curl -fsSL https://deb.nodesource.com/setup_lts.x | bash - \
    && apt-get install -y nodejs \
    && rm -rf /var/lib/apt/lists/*

# Set up workspace
RUN mkdir -p /workspace

# Install Claude via official script and copy to shared location
RUN curl -fsSL https://claude.ai/install.sh | bash \
    && cp -L /root/.local/bin/claude /usr/local/bin/claude \
    && chmod 755 /usr/local/bin/claude

# Set up environment
ENV HOME=/root

# Default working directory
WORKDIR /workspace

# Default command
ENTRYPOINT ["/usr/local/bin/claude"]
