#!/usr/bin/env bash
set -euo pipefail

# ═══════════════════════════════════════════════════════════════════════════════
# Bump all package versions to the next patch version.
#
# Finds the highest version across all packages, increments the patch number,
# and applies it uniformly to every package.json, pyproject.toml, and the
# release-please manifest.
#
# Usage:
#   ./scripts/bump-version.sh          # auto-increment patch
#   ./scripts/bump-version.sh 2.0.0    # set explicit version
# ═══════════════════════════════════════════════════════════════════════════════

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

# ── Collect all current versions ──────────────────────────────────────────────

TS_PACKAGES=(
  typescript/packages/core/package.json
  typescript/packages/client/package.json
  typescript/packages/cli/package.json
  typescript/packages/adapters/zod/package.json
  typescript/packages/zod-extractor/package.json
  typescript/packages/zod-bridge/package.json
)

PY_PACKAGES=(
  python/packages/core/pyproject.toml
  python/packages/client/pyproject.toml
  python/packages/cli/pyproject.toml
  python/packages/adapters/pydantic/pyproject.toml
  python/packages/pydantic-extractor/pyproject.toml
  python/packages/pydantic-bridge/pyproject.toml
)

highest="0.0.0"

version_gt() {
  # Returns 0 if $1 > $2 using sort -V
  [ "$1" != "$2" ] && [ "$(printf '%s\n%s' "$1" "$2" | sort -V | tail -1)" = "$1" ]
}

# Scan TypeScript packages
for pkg in "${TS_PACKAGES[@]}"; do
  v=$(python3 -c "import json; print(json.load(open('${REPO_ROOT}/${pkg}'))['version'])")
  if version_gt "$v" "$highest"; then highest="$v"; fi
done

# Scan Python packages
for pkg in "${PY_PACKAGES[@]}"; do
  v=$(grep '^version' "${REPO_ROOT}/${pkg}" | head -1 | sed 's/version = "\(.*\)"/\1/')
  if version_gt "$v" "$highest"; then highest="$v"; fi
done

# Scan manifest
manifest_max=$(python3 -c "
import json
versions = json.load(open('${REPO_ROOT}/.release-please-manifest.json')).values()
print(sorted(versions, key=lambda v: list(map(int, v.split('.'))))[-1])
")
if version_gt "$manifest_max" "$highest"; then highest="$manifest_max"; fi

# ── Compute next version ─────────────────────────────────────────────────────

if [[ $# -ge 1 ]]; then
  NEXT="$1"
else
  IFS='.' read -r major minor patch <<< "$highest"
  patch=$((patch + 1))
  if [[ $patch -gt 99 ]]; then
    patch=0
    minor=$((minor + 1))
  fi
  if [[ $minor -gt 99 ]]; then
    minor=0
    major=$((major + 1))
  fi
  NEXT="${major}.${minor}.${patch}"
fi

echo "Highest current version: ${highest}"
echo "Bumping all packages to: ${NEXT}"
echo ""

# ── Apply to TypeScript packages ──────────────────────────────────────────────

for pkg in "${TS_PACKAGES[@]}"; do
  python3 -c "
import json, pathlib
p = pathlib.Path('${REPO_ROOT}/${pkg}')
d = json.loads(p.read_text())
d['version'] = '${NEXT}'
p.write_text(json.dumps(d, indent=2) + '\n')
"
  echo "  ${pkg} -> ${NEXT}"
done

# ── Apply to Python packages ─────────────────────────────────────────────────

for pkg in "${PY_PACKAGES[@]}"; do
  python3 -c "
from pathlib import Path

p = Path('${REPO_ROOT}/${pkg}')
lines = p.read_text().splitlines()
updated = []
replaced = False

for line in lines:
    if not replaced and line.startswith('version = \"'):
        updated.append('version = \"${NEXT}\"')
        replaced = True
    else:
        updated.append(line)

if not replaced:
    raise SystemExit(f'No version field found in {p}')

p.write_text('\n'.join(updated) + '\n')
"
  echo "  ${pkg} -> ${NEXT}"
done

# ── Apply to release-please manifest ─────────────────────────────────────────

python3 -c "
import json, pathlib
p = pathlib.Path('${REPO_ROOT}/.release-please-manifest.json')
d = json.loads(p.read_text())
for k in d: d[k] = '${NEXT}'
p.write_text(json.dumps(d, indent=2) + '\n')
"
echo "  .release-please-manifest.json -> all ${NEXT}"

echo ""
echo "Done. All packages are now at ${NEXT}."
echo "Run 'git add -p && git commit' to commit the bump."
