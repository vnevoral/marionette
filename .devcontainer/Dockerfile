# Development environment for the Marionette project.
# Target runtime is a Raspberry Pi running Ubuntu (linux/arm64), this image is only for local development.
FROM mcr.microsoft.com/devcontainers/go:1-1.23-bookworm

# Cross-compilation toolchain for building linux/arm64 binaries from any dev host.
USER root
RUN rm -f /etc/apt/sources.list.d/yarn.list \
	&& apt-get update && export DEBIAN_FRONTEND=noninteractive \
	&& apt-get -y install --no-install-recommends \
		gcc-aarch64-linux-gnu \
		g++-aarch64-linux-gnu \
	&& apt-get clean -y && rm -rf /var/lib/apt/lists/*

# Go tooling used during development (linting, hot reload, formatting).
USER vscode
RUN go install github.com/air-verse/air@v1.61.0 \
	&& go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.61.0 \
	&& go install mvdan.cc/gofumpt@v0.6.0
