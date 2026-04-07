#!/bin/bash
set -euo pipefail

cd typescript
pnpm install --no-frozen-lockfile
pnpm run build
