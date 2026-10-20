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
ARG GO_VERSION=1.27.3
ARG GO_SHA256_AMD64=65efbc80fdc1fa2cd2631cad5b481777a68e8725f34e89f50fc6ac714e379b40
ARG GO_SHA256_ARM64=a06e99309dbcbdca1a8ccd2282c3ee146e55930886e26b4f1193269587165f49
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
