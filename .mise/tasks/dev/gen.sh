#!/usr/bin/env bash
#MISE description="Generate code"
set -euo pipefail

go generate ./...
