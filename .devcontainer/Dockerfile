# Development environment for the Marionette project.
# Target runtime is a Raspberry Pi running Ubuntu (linux/arm64), this image is only for local development.
FROM mcr.microsoft.com/devcontainers/go:1-1.24-bookworm

# Cross-compilation toolchain for building linux/arm64 binaries from any dev host.
USER root
RUN rm -f /etc/apt/sources.list.d/yarn.list \
	&& apt-get update && export DEBIAN_FRONTEND=noninteractive \
	&& apt-get -y install --no-install-recommends \
		gcc-aarch64-linux-gnu \
		g++-aarch64-linux-gnu \
	&& apt-get clean -y && rm -rf /var/lib/apt/lists/*

# Go toolchain pinned to the version in go.mod. The devcontainers/go image
# does not ship newer Go releases, so the image's Go is replaced by the
# official archive, verified against the checksum published on go.dev.
ARG GO_VERSION=1.27.4
ARG GO_SHA256_AMD64=7226736c2339719cda28fbf3a54d71aa63bcefe376e98cef00034268dab24840
ARG GO_SHA256_ARM64=3392582988a273a87ab5043ee1b2c9c17a2c3c00580bf9326c47a2e0aaaeac92
RUN arch="$(dpkg --print-architecture)" \
	&& case "$arch" in \
		amd64) sum="$GO_SHA256_AMD64" ;; \
		arm64) sum="$GO_SHA256_ARM64" ;; \
		*) echo "unsupported architecture $arch" >&2; exit 1 ;; \
	esac \
	&& curl -fsSLo /tmp/go.tgz "https://go.dev/dl/go${GO_VERSION}.linux-${arch}.tar.gz" \
	&& echo "$sum  /tmp/go.tgz" | sha256sum -c - \
	&& rm -rf /usr/local/go && tar -C /usr/local -xzf /tmp/go.tgz && rm /tmp/go.tgz

# Go tooling used during development (linting, hot reload, formatting).
# golangci-lint v2 is installed as a prebuilt binary (the same version as in
# CI); gofumpt matches the version bundled in that golangci-lint release.
USER vscode
RUN go install github.com/air-verse/air@v1.67.4 \
	&& go install mvdan.cc/gofumpt@v0.12.0 \
	&& curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/v2.15.0/install.sh \
		| sh -s -- -b "$(go env GOPATH)/bin" v2.15.0
