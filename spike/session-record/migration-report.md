# Migration Report: bw CLI v0.0.16 → v0.0.17

## Summary

Migrated the project configuration and documentation from bw CLI v0.0.16 to v0.0.17, updating 4 files to match the new command structure and configuration format.

## Changes

### `bw.toml` — Configuration restructure
- Moved CDK settings from the top-level `[cdk]` section into `[project.tool.cdk]` under the "infra" project, following the new per-project tool configuration format.
- Adjusted the `pre-bootstrap` template path to be relative to the project directory (`cdk/pre-bootstrap.cfn.yaml` instead of `infra/aws/cdk/pre-bootstrap.cfn.yaml`).

### `AGENTS.md` — Command reference updates (10 changes)
| Old Command | New Command |
|---|---|
| `bw check compiles -p infra` | `bw build -p infra` |
| `bw check-all` (×3) | `bw preflight` |
| `bw cdk bootstrap` | `bw infra bootstrap` |
| `bw cdk diff` | `bw infra diff` |
| `bw cdk deploy [--hotswap]` | `bw infra deploy [--hotswap]` |
| `bw cdk endpoints` (×2) | `bw infra inspect -l endpoints` |
| `bw cdk log-groups` | `bw infra inspect -l logs` |
| `bw dev gen` | `bw gen` |

### `.github/workflows/checks.yml` — CI pipeline update
- `bw check-all` → `bw preflight`

### `infra/aws/cdk/SETUP_AWS.md` — Documentation update
- `bw cdk bootstrap` → `bw infra bootstrap`
