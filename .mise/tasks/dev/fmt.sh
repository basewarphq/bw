#!/usr/bin/env bash
#MISE description="Format Go files and shell scripts"
#MISE depends=["dev:gen"]
set -euo pipefail

golangci-lint fmt ./...
find .mise -name "*.sh" -type f -exec shfmt -w {} +
